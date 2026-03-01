package order

import (
	"context"
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	autherrors "go-gadget-api/internal/auth/errors"
	"go-gadget-api/internal/cart"
	"go-gadget-api/internal/midtrans"
	"go-gadget-api/internal/outbox"
	"go-gadget-api/internal/shared/database/dbgen"
	"go-gadget-api/internal/shared/database/helper"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

//go:generate mockgen -source=order_service.go -destination=../mocks/order/order_service_mock.go -package=mock
type Service interface {
	// Customer Actions
	Checkout(ctx context.Context, userID string, req CheckoutRequest) (OrderResponse, error)
	List(ctx context.Context, userID string, status string, page, limit int) ([]OrderResponse, int64, error)
	Detail(ctx context.Context, orderID string) (OrderResponse, error)
	Cancel(ctx context.Context, orderID string) error
	Complete(ctx context.Context, orderID string, userID string, nextStatus string) (OrderResponse, error)
	ContinuePayment(ctx context.Context, orderID string, userID string) (*midtrans.CreateTransactionResponse, error)

	// Shared/Admin Actions
	ListAdmin(ctx context.Context, status string, search string, page, limit int) ([]OrderResponse, int64, error)
	UpdateStatusByAdmin(ctx context.Context, orderID string, nextStatus string, receiptNo *string) (OrderResponse, error)
	UpdatePaymentStatus(ctx context.Context, orderID string, input UpdatePaymentStatusInput) (OrderResponse, error)
	UpdatePaymentStatusByOrderNumber(ctx context.Context, orderNumber string, input UpdatePaymentStatusInput) (OrderResponse, error)
	HandleMidtransNotification(ctx context.Context, payload MidtransNotificationRequest) error
}

type service struct {
	db          *sql.DB
	repo        Repository
	outboxRepo  outbox.Repository
	cartSvc     cart.Service
	midtransSvc midtrans.Service
	logger      *zap.Logger
}

type Deps struct {
	DB          *sql.DB
	Repo        Repository
	OutboxRepo  outbox.Repository
	CartSvc     cart.Service
	MidtransSvc midtrans.Service
	Logger      *zap.Logger
}

var paymentStatusTransitions = map[string]map[string]struct{}{
	"UNPAID": {
		"PAID":     {},
		"REFUNDED": {},
	},
	"PAID": {
		"UNPAID":   {},
		"REFUNDED": {},
	},
	"REFUNDED": {},
}

func NewService(deps Deps) Service {
	// 1. Validasi Dependencies
	if deps.DB == nil {
		panic("db cannot be nil")
	}
	if deps.Repo == nil {
		panic("order repository cannot be nil")
	}
	if deps.OutboxRepo == nil {
		panic("outbox repository cannot be nil")
	}
	if deps.CartSvc == nil {
		panic("cart service cannot be nil")
	}
	if deps.MidtransSvc == nil {
		panic("midtrans service cannot be nil")
	}
	if deps.Logger == nil {
		deps.Logger = zap.NewNop()
	}

	// 2. Inisialisasi Service
	return &service{
		db:          deps.DB,
		repo:        deps.Repo,
		outboxRepo:  deps.OutboxRepo,
		cartSvc:     deps.CartSvc,
		midtransSvc: deps.MidtransSvc,
		logger:      deps.Logger, // Pastikan ini dipetakan
	}
}

func (s *service) ContinuePayment(ctx context.Context, orderID string, userID string) (*midtrans.CreateTransactionResponse, error) {
	parsedOrderID, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order id: %w", err)
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	order, err := s.repo.GetByID(ctx, parsedOrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order.UserID != parsedUserID {
		return nil, fmt.Errorf("order does not belong to user")
	}

	if order.PaymentStatus != "UNPAID" {
		return nil, fmt.Errorf("order payment cannot be retried unless it is still unpaid")
	}

	// Check if token already exists and not expired
	if order.SnapToken.Valid && order.SnapTokenExpiredAt.Valid && order.SnapTokenExpiredAt.Time.After(time.Now()) {
		return &midtrans.CreateTransactionResponse{
			Token:       order.SnapToken.String,
			RedirectURL: order.SnapRedirectUrl.String,
		}, nil
	}

	// Create new token
	items, err := s.repo.GetItems(ctx, parsedOrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	midtransItems := make([]midtrans.ItemDetail, 0, len(items))
	for _, item := range items {
		price, _ := strconv.ParseFloat(item.UnitPrice, 64)
		midtransItems = append(midtransItems, midtrans.ItemDetail{
			ID:    item.ProductID.String(),
			Price: int64(price),
			Qty:   item.Quantity,
			Name:  item.NameSnapshot,
		})
	}

	totalPrice, _ := strconv.ParseFloat(order.TotalPrice, 64)
	midtransReq := &midtrans.CreateTransactionRequest{
		OrderID:     fmt.Sprintf("%s_%d", order.OrderNumber, time.Now().Unix()),
		GrossAmount: int64(totalPrice),
		Items:       midtransItems,
		Customer:    &midtrans.CustomerDetails{}, // Empty for now, or load from profile if needed
	}

	midtransResp, err := s.midtransSvc.CreateTransactionToken(midtransReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create midtrans transaction: %w", err)
	}

	if midtransResp == nil {
		return nil, fmt.Errorf("received nil response from midtrans")
	}

	// Update order with new token
	_, err = s.repo.UpdateOrderSnapToken(ctx, dbgen.UpdateOrderSnapTokenParams{
		ID:                 parsedOrderID,
		SnapToken:          sql.NullString{String: midtransResp.Token, Valid: true},
		SnapRedirectUrl:    sql.NullString{String: midtransResp.RedirectURL, Valid: true},
		SnapTokenExpiredAt: sql.NullTime{Time: time.Now().Add(24 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update order snap token: %w", err)
	}

	return midtransResp, nil
}

func (s *service) Checkout(
	ctx context.Context,
	userID string,
	req CheckoutRequest,
) (OrderResponse, error) {
	// Logger dengan context awal
	logger := s.logger.With(zap.String("user_id", userID))

	// 1. Validasi & Ambil Cart
	cartData, err := s.cartSvc.Detail(ctx, userID)
	if err != nil {
		logger.Error("failed to fetch cart detail", zap.Error(err))
		return OrderResponse{}, err
	}
	if len(cartData.Items) == 0 {
		return OrderResponse{}, ErrCartEmpty
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		logger.Warn("invalid user id format", zap.Error(err))
		return OrderResponse{}, autherrors.ErrInvalidUserID
	}

	// 2. Hitung Harga
	var subtotal float64
	for _, item := range cartData.Items {
		subtotal += float64(item.Price) * float64(item.Qty)
	}

	shippingPrice := 0.0
	total := subtotal + shippingPrice

	// 3. Address Handling
	var addressID uuid.NullUUID
	if req.AddressID != "" {
		parsedID, err := uuid.Parse(req.AddressID)
		if err != nil {
			return OrderResponse{}, autherrors.ErrInvalidUserID
		}
		addressID = uuid.NullUUID{UUID: parsedID, Valid: true}
	}

	addressSnapshot, _ := json.Marshal(map[string]string{"address_id": req.AddressID})

	// 4. Generate Order Number & Info Dasar
	orderNumber := fmt.Sprintf("GGS-%d-%s", time.Now().Unix(), strings.ToUpper(uuid.New().String()[:4]))
	logger = logger.With(zap.String("order_number", orderNumber))

	// fetch user info for midtrans
	userData, err := s.repo.GetUserByID(ctx, uid)
	if err != nil {
		logger.Error("failed to fetch user info", zap.Error(err))
		return OrderResponse{}, err
	}

	// 5. Midtrans Integration
	var midtransItems []midtrans.ItemDetail
	for _, item := range cartData.Items {
		midtransItems = append(midtransItems, midtrans.ItemDetail{
			ID:    item.ProductID,
			Price: int64(item.Price),
			Qty:   item.Qty,
			Name:  item.ProductName,
		})
	}

	midtransReq := &midtrans.CreateTransactionRequest{
		OrderID:     orderNumber,
		GrossAmount: int64(total),
		Customer: &midtrans.CustomerDetails{
			FirstName: userData.Name,
			Email:     userData.Email,
		},
		Items: midtransItems,
	}

	midtransResp, err := s.midtransSvc.CreateTransactionToken(midtransReq)
	if err != nil {
		logger.Error("failed to create midtrans transaction", zap.Error(err))
		return OrderResponse{}, err
	}

	if midtransResp == nil {
		logger.Error("midtrans response is nil")
		return OrderResponse{}, ErrOrderFailed
	}

	// 6. Begin Transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return OrderResponse{}, ErrOrderFailed
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
			logger.Warn("transaction rolled back")
		}
	}()

	qtx := s.repo.WithTx(tx)

	// 7. Create Order
	order, err := qtx.CreateOrder(ctx, dbgen.CreateOrderParams{
		OrderNumber:     orderNumber,
		UserID:          uid,
		Status:          "PENDING",
		AddressID:       addressID,
		AddressSnapshot: addressSnapshot,
		SubtotalPrice:   fmt.Sprintf("%.2f", subtotal),
		ShippingPrice:   fmt.Sprintf("%.2f", shippingPrice),
		TotalPrice:      fmt.Sprintf("%.2f", total),
		Note:            helper.StringToNull(&req.Note),
		SnapToken:       sql.NullString{String: midtransResp.Token, Valid: midtransResp.Token != ""},
		SnapRedirectUrl: sql.NullString{String: midtransResp.RedirectURL, Valid: midtransResp.RedirectURL != ""},
	})
	if err != nil {
		logger.Error("failed to create order record", zap.Error(err))
		return OrderResponse{}, err
	}

	// 6. Create Order Items
	for _, item := range cartData.Items {
		productID, _ := uuid.Parse(item.ProductID)
		err = qtx.CreateOrderItem(ctx, dbgen.CreateOrderItemParams{
			OrderID:      order.ID,
			ProductID:    productID,
			NameSnapshot: item.ProductName,
			UnitPrice:    fmt.Sprintf("%.2f", float64(item.Price)),
			Quantity:     item.Qty,
			TotalPrice:   fmt.Sprintf("%.2f", float64(item.Price)*float64(item.Qty)),
		})
		if err != nil {
			logger.Error("failed to create order item", zap.String("product_id", item.ProductID), zap.Error(err))
			return OrderResponse{}, err
		}
	}

	// 7. Outbox Event
	if s.outboxRepo == nil {
		logger.DPanic("outboxRepo is missing in service") // DPanic akan panic di dev, error di prod
		return OrderResponse{}, ErrOrderFailed
	}

	payload, _ := json.Marshal(map[string]string{
		"user_id":  userID,
		"order_id": order.ID.String(),
	})

	err = s.outboxRepo.WithTx(tx).CreateOutboxEvent(ctx, dbgen.CreateOutboxEventParams{
		ID:            uuid.New(),
		AggregateType: "ORDER",
		AggregateID:   order.ID,
		EventType:     "DELETE_CART",
		Payload:       payload,
	})
	if err != nil {
		logger.Error("failed to create outbox event", zap.Error(err))
		return OrderResponse{}, err
	}

	// 8. Commit
	if err := tx.Commit(); err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return OrderResponse{}, ErrOrderFailed
	}
	committed = true

	logger.Info("checkout success", zap.String("order_id", order.ID.String()))

	return s.mapOrderToResponse(order, nil), nil
}

// internal/order/order.service.ts

func (s *service) List(ctx context.Context, userID string, status string, page, limit int) ([]OrderResponse, int64, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid user id format: %w", err)
	}

	var statusArg sql.NullString
	if status != "" {
		statusArg = sql.NullString{String: status, Valid: true}
	}

	rows, err := s.repo.List(ctx, dbgen.ListOrdersParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
		UserID: uid,
		Status: statusArg,
	})
	if err != nil {
		log.Printf("[ListOrders] repo.List error: %+v\n", err)
		return nil, 0, err
	}
	log.Printf("[ListOrders] repo.List success: %d rows returned\n", len(rows))
	res := make([]OrderResponse, 0, len(rows))
	var total int64

	for _, r := range rows {
		// 1. Log data mentah dari DB (untuk kepastian 100%)
		log.Printf("[ListOrders] Order ID: %s | Raw JSON: %s\n", r.ID, string(r.ItemsJson))

		total = r.TotalCount

		// 2. Inisialisasi slice kosong (bukan nil) di setiap iterasi
		currentItems := make([]OrderItemResponse, 0)

		if len(r.ItemsJson) > 0 {
			// Unmarshal ke variabel lokal yang benar-benar baru
			if err := json.Unmarshal(r.ItemsJson, &currentItems); err != nil {
				log.Printf("[ListOrders] Error unmarshal: %v\n", err)
			}
		}

		// 3. Log hasil setelah unmarshal (sebelum append)
		if len(currentItems) > 0 {
			log.Printf("[ListOrders] Mapped NameSnapshot: %s\n", currentItems[0].NameSnapshot)
		}

		totalPrice, _ := strconv.ParseFloat(r.TotalPrice, 64)

		// 4. Masukkan data ke struct response
		res = append(res, OrderResponse{
			ID:          r.ID.String(),
			OrderNumber: r.OrderNumber,
			Status:      r.Status,
			TotalPrice:  totalPrice,
			PlacedAt:    r.PlacedAt,
			Items:       currentItems,
		})
	}

	// Pastikan tidak mengembalikan nil slice ke frontend
	if len(res) == 0 {
		res = []OrderResponse{}
	}

	return res, total, nil
}

func (s *service) ListAdmin(ctx context.Context, status string, search string, page int, limit int) ([]OrderResponse, int64, error) {
	rows, err := s.repo.ListAdmin(ctx, dbgen.ListOrdersAdminParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
		// Menggunakan helper ToText untuk mengonversi string ke sql.NullString
		Status: helper.StringToNull(&status),
		Search: helper.StringToNull(&search),
	})
	if err != nil {
		return nil, 0, err
	}

	var res []OrderResponse
	var total int64
	if len(rows) > 0 {
		for _, r := range rows {
			total = r.TotalCount
			// Melakukan type casting dari row result ke dbgen.Order
			res = append(res, s.mapOrderToResponse(dbgen.Order{
				ID:          r.ID,
				OrderNumber: r.OrderNumber,
				UserID:      r.UserID,
				Status:      r.Status,
				TotalPrice:  r.TotalPrice,
				PlacedAt:    r.PlacedAt,
				// ... field lain sesuai ketersediaan di ListOrdersAdminRow
			}, nil))
		}
	}

	return res, total, nil
}

// CUSTOMER & ADMIN: Detail
func (s *service) Detail(ctx context.Context, orderID string) (OrderResponse, error) {
	oid, err := uuid.Parse(orderID)
	if err != nil {
		return OrderResponse{}, ErrInvalidOrderID // Sesuaikan dengan package error Anda
	}

	// row sekarang sudah mengandung ItemsJson hasil subquery SQL
	row, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OrderResponse{}, ErrOrderNotFound
		}
		return OrderResponse{}, err
	}

	// 1. Unmarshal ItemsJson (Hasil subquery)
	var items []OrderItemResponse
	if len(row.ItemsJson) > 0 {
		if err := json.Unmarshal(row.ItemsJson, &items); err != nil {
			log.Printf("[Order.Detail] Error unmarshal items: %v", err)
			// Kita tetap lanjut meskipun items gagal, atau return err sesuai kebijakan
		}
	}

	// 2. Mapping Manual (atau panggil fungsi helper mapOrderToResponse)
	totalPrice, _ := strconv.ParseFloat(row.TotalPrice, 64)
	shippingPrice, _ := strconv.ParseFloat(row.ShippingPrice, 64)
	subtotalPrice, _ := strconv.ParseFloat(row.SubtotalPrice, 64)

	res := OrderResponse{
		ID:            row.ID.String(),
		OrderNumber:   row.OrderNumber,
		Status:        row.Status,
		PaymentStatus: row.PaymentStatus,
		SubtotalPrice: subtotalPrice,
		TotalPrice:    totalPrice,
		ShippingPrice: shippingPrice,
		PlacedAt:      row.PlacedAt,
		Items:         items, // Langsung hasil unmarshal tadi
	}

	// 3. Handle AddressSnapshot jika diperlukan di Next.js
	// res.Address = row.AddressSnapshot ...

	return res, nil
}

// CUSTOMER: Cancel
func (s *service) Cancel(ctx context.Context, orderID string) error {
	oid, err := uuid.Parse(orderID)
	if err != nil {
		return ErrInvalidOrderID // Pastikan error ini ada di order_errors.go
	}

	// 1. Ambil data order (Bisa di luar transaksi untuk cek awal)
	o, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return err
	}

	// 2. Validasi status
	if o.Status != "PENDING" {
		return ErrCannotCancel
	}

	// 3. Mulai Transaksi
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 4. Gunakan WithTx
	qtx := s.repo.WithTx(tx)

	// 5. Update Status melalui qtx
	_, err = qtx.UpdateStatus(ctx, oid, "CANCELLED")
	if err != nil {
		fmt.Println("DEBUG: Cancel Order error:", err)
		return err
	}

	// Jika ke depannya ada logika kembalikan stok:
	// err = s.productSvc.RestoreStock(ctx, o.Items)
	// if err != nil { return err }

	return tx.Commit()
}

// CUSTOMER: Update (DELIVERED -> COMPLETED)
func (s *service) Complete(ctx context.Context, orderID string, userID string, status string) (OrderResponse, error) {
	oid, err := uuid.Parse(orderID)
	if err != nil {
		return OrderResponse{}, ErrInvalidOrderID
	}

	// 1. Mulai Transaksi
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return OrderResponse{}, err
	}
	defer tx.Rollback()

	// 2. Hubungkan Repository dengan Transaksi
	qtx := s.repo.WithTx(tx)

	currentOrder, err := qtx.GetByID(ctx, oid)
	if err != nil {
		return OrderResponse{}, err
	}

	if currentOrder.UserID != uuid.MustParse(userID) {
		return OrderResponse{}, autherrors.ErrUnauthorized
	}

	// 3. Eksekusi Update Status
	o, err := qtx.UpdateStatus(ctx, oid, status)
	if err != nil {
		// Jika error (misal: order tidak ketemu atau DB error)
		return OrderResponse{}, err
	}

	// --- LOGIKA TAMBAHAN (Opsional di masa depan) ---
	// Jika status == "SHIPPED", mungkin Anda ingin otomatis kirim email/notifikasi
	// if status == "SHIPPED" {
	//    s.notificationSvc.Send(o.UserID, "Pesanan Anda sedang dikirim!")
	// }

	// 4. Commit Transaksi
	if err := tx.Commit(); err != nil {
		return OrderResponse{}, err
	}

	return s.mapOrderToResponse(o, nil), nil
}

// Implementasi UpdateStatusByAdmin
func (s *service) UpdateStatusByAdmin(ctx context.Context, orderID string, nextStatus string, receiptNo *string) (OrderResponse, error) {
	oid, err := uuid.Parse(orderID)
	if err != nil {
		return OrderResponse{}, ErrInvalidOrderID
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return OrderResponse{}, err
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)
	order, err := qtx.GetByID(ctx, oid)
	if err != nil {
		// Menggunakan ErrOrderNotFound jika data tidak ada di DB
		return OrderResponse{}, ErrOrderNotFound
	}

	// --- VALIDASI TRANSISI STATUS ---
	switch nextStatus {
	case "PROCESSING":
		if order.Status != "PAID" {
			return OrderResponse{}, ErrInvalidStatusTransition
		}
	case "SHIPPED":
		if order.Status != "PROCESSING" {
			return OrderResponse{}, ErrInvalidStatusTransition
		}
		if receiptNo == nil || *receiptNo == "" {
			return OrderResponse{}, ErrReceiptRequired
		}
	case "DELIVERED":
		if order.Status != "SHIPPED" {
			return OrderResponse{}, ErrInvalidStatusTransition
		}
	default:
		// Jika admin mencoba status yang tidak diizinkan di sini
		return OrderResponse{}, ErrInvalidStatusTransition
	}

	// Update Status
	o, err := qtx.UpdateStatus(ctx, oid, nextStatus)
	if err != nil {
		return OrderResponse{}, ErrOrderFailed
	}

	if err := tx.Commit(); err != nil {
		return OrderResponse{}, ErrOrderFailed
	}

	return s.mapOrderToResponse(o, nil), nil
}

func (s *service) UpdatePaymentStatus(ctx context.Context, orderID string, input UpdatePaymentStatusInput) (OrderResponse, error) {
	oid, err := uuid.Parse(orderID)
	if err != nil {
		return OrderResponse{}, ErrInvalidOrderID
	}

	return s.updatePaymentStatusWithFilter(ctx, input, "id = $1", oid)
}

func (s *service) UpdatePaymentStatusByOrderNumber(ctx context.Context, orderNumber string, input UpdatePaymentStatusInput) (OrderResponse, error) {
	orderNumber = strings.TrimSpace(orderNumber)
	if orderNumber == "" {
		return OrderResponse{}, ErrInvalidOrderNumber
	}

	return s.updatePaymentStatusWithFilter(ctx, input, "order_number = $1", orderNumber)
}

func (s *service) HandleMidtransNotification(ctx context.Context, payload MidtransNotificationRequest) error {
	if err := validateMidtransNotification(payload); err != nil {
		return err
	}

	if err := verifyMidtransSignature(payload); err != nil {
		return err
	}

	orderSummary, err := s.getOrderSummaryByOrderNumber(ctx, payload.OrderID)
	if err != nil {
		return err
	}

	if strings.EqualFold(payload.TransactionStatus, "expire") {
		_, err = s.updatePaymentStatusWithFilter(
			ctx,
			UpdatePaymentStatusInput{
				PaymentStatus: "REFUNDED",
				CancelledAt:   timePtr(time.Now()),
				Note:          stringPtr("expired by midtrans"),
			},
			"order_number = $1",
			payload.OrderID,
		)
		return err
	}

	shouldMarkPaid := strings.EqualFold(payload.TransactionStatus, "settlement") ||
		(strings.EqualFold(payload.TransactionStatus, "capture") && strings.EqualFold(payload.FraudStatus, "accept"))
	if !shouldMarkPaid {
		return nil
	}

	grossAmount, err := parseCurrencyToCents(payload.GrossAmount)
	if err != nil {
		return ErrInvalidGrossAmount
	}

	expectedGross, err := calculateExpectedGrossCents(orderSummary.SubtotalPrice, orderSummary.DiscountPrice, orderSummary.ShippingPrice)
	if err != nil {
		return err
	}
	if grossAmount != expectedGross {
		return ErrGrossAmountMismatch
	}

	paidAt, err := parseMidtransTransactionTime(payload.TransactionTime)
	if err != nil {
		return err
	}

	_, err = s.UpdatePaymentStatusByOrderNumber(ctx, payload.OrderID, UpdatePaymentStatusInput{
		PaymentStatus: "PAID",
		PaymentMethod: payload.PaymentType,
		PaidAt:        &paidAt,
	})
	return err
}

// Removed manual structs as we now use dbgen types

func (s *service) updatePaymentStatusWithFilter(ctx context.Context, input UpdatePaymentStatusInput, filter string, filterValue any) (OrderResponse, error) {
	nextStatus := strings.ToUpper(strings.TrimSpace(input.PaymentStatus))
	if nextStatus == "" {
		return OrderResponse{}, ErrInvalidPaymentStatus
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return OrderResponse{}, ErrOrderFailed
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	var row struct {
		ID            uuid.UUID
		Status        string
		PaymentStatus string
		PaidAt        sql.NullTime
		CancelledAt   sql.NullTime
	}

	if filter == "id = $1" {
		res, err := qtx.GetOrderPaymentForUpdateByID(ctx, filterValue.(uuid.UUID))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return OrderResponse{}, ErrOrderNotFound
			}
			return OrderResponse{}, ErrOrderFailed
		}
		row.ID = res.ID
		row.Status = res.Status
		row.PaymentStatus = res.PaymentStatus
		row.PaidAt = res.PaidAt
		row.CancelledAt = res.CancelledAt
	} else {
		res, err := qtx.GetOrderPaymentForUpdateByOrderNumber(ctx, filterValue.(string))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return OrderResponse{}, ErrOrderNotFound
			}
			return OrderResponse{}, ErrOrderFailed
		}
		row.ID = res.ID
		row.Status = res.Status
		row.PaymentStatus = res.PaymentStatus
		row.PaidAt = res.PaidAt
		row.CancelledAt = res.CancelledAt
	}

	currentStatus := strings.ToUpper(strings.TrimSpace(row.PaymentStatus))
	if currentStatus == nextStatus {
		if err := tx.Commit(); err != nil {
			return OrderResponse{}, ErrOrderFailed
		}
		return s.Detail(ctx, row.ID.String())
	}

	allowedTransitions, exists := paymentStatusTransitions[currentStatus]
	if !exists {
		return OrderResponse{}, ErrInvalidPaymentStatus
	}
	if _, allowed := allowedTransitions[nextStatus]; !allowed {
		return OrderResponse{}, ErrInvalidPaymentStatusTransition
	}

	now := time.Now()
	paymentMethod := strings.TrimSpace(input.PaymentMethod)
	var method sql.NullString
	if paymentMethod != "" {
		method = sql.NullString{String: paymentMethod, Valid: true}
	}

	paidAt := row.PaidAt
	cancelledAt := row.CancelledAt
	nextOrderStatus := row.Status

	switch nextStatus {
	case "PAID":
		if input.PaidAt != nil {
			paidAt = sql.NullTime{Time: *input.PaidAt, Valid: true}
		} else if !paidAt.Valid {
			paidAt = sql.NullTime{Time: now, Valid: true}
		}
		if row.Status == "PENDING" {
			nextOrderStatus = "PAID"
		}
	case "REFUNDED":
		if input.CancelledAt != nil {
			cancelledAt = sql.NullTime{Time: *input.CancelledAt, Valid: true}
		} else if !cancelledAt.Valid {
			cancelledAt = sql.NullTime{Time: now, Valid: true}
		}
		if row.Status == "PENDING" || row.Status == "PAID" {
			nextOrderStatus = "CANCELLED"
		}
	case "UNPAID":
		paidAt = sql.NullTime{}
		if row.Status == "PAID" {
			nextOrderStatus = "PENDING"
		}
	default:
		return OrderResponse{}, ErrInvalidPaymentStatus
	}

	note := sql.NullString{}
	if input.Note != nil {
		trimmedNote := strings.TrimSpace(*input.Note)
		if trimmedNote != "" {
			note = sql.NullString{String: trimmedNote, Valid: true}
		}
	}

	var noteStr string
	if note.Valid {
		noteStr = note.String
	}

	var methodStr string
	if method.Valid {
		methodStr = method.String
	}

	_, err = qtx.UpdateOrderPaymentStatus(ctx, dbgen.UpdateOrderPaymentStatusParams{
		ID:            row.ID,
		PaymentStatus: nextStatus,
		PaymentMethod: methodStr,
		PaidAt:        paidAt,
		CancelledAt:   cancelledAt,
		Status:        nextOrderStatus,
		Note:          noteStr,
	})
	if err != nil {
		return OrderResponse{}, ErrOrderFailed
	}

	if err := tx.Commit(); err != nil {
		return OrderResponse{}, ErrOrderFailed
	}

	return s.Detail(ctx, row.ID.String())
}

func (s *service) getOrderSummaryByOrderNumber(ctx context.Context, orderNumber string) (dbgen.GetOrderSummaryByOrderNumberRow, error) {
	orderNumber = strings.TrimSpace(orderNumber)
	if orderNumber == "" {
		return dbgen.GetOrderSummaryByOrderNumberRow{}, ErrInvalidOrderNumber
	}

	res, err := s.repo.GetOrderSummaryByOrderNumber(ctx, orderNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dbgen.GetOrderSummaryByOrderNumberRow{}, ErrOrderNotFound
		}
		return dbgen.GetOrderSummaryByOrderNumberRow{}, ErrOrderFailed
	}

	return res, nil
}

func validateMidtransNotification(payload MidtransNotificationRequest) error {
	if strings.TrimSpace(payload.OrderID) == "" ||
		strings.TrimSpace(payload.StatusCode) == "" ||
		strings.TrimSpace(payload.GrossAmount) == "" ||
		strings.TrimSpace(payload.SignatureKey) == "" ||
		strings.TrimSpace(payload.TransactionStatus) == "" {
		return ErrInvalidMidtransPayload
	}
	return nil
}

func verifyMidtransSignature(payload MidtransNotificationRequest) error {
	serverKey := strings.TrimSpace(os.Getenv("MIDTRANS_SERVER_KEY"))
	if serverKey == "" {
		return ErrMidtransServerKeyNotConfigured
	}

	raw := payload.OrderID + payload.StatusCode + payload.GrossAmount + serverKey
	hash := sha512.Sum512([]byte(raw))
	expectedSignature := hex.EncodeToString(hash[:])
	incomingSignature := strings.ToLower(strings.TrimSpace(payload.SignatureKey))
	if expectedSignature != incomingSignature {
		return ErrInvalidMidtransSignature
	}

	return nil
}

func parseCurrencyToCents(amount string) (int64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	if err != nil {
		return 0, err
	}
	return int64(math.Round(parsed * 100)), nil
}

func calculateExpectedGrossCents(subtotal, discount, shipping string) (int64, error) {
	subtotalCents, err := parseCurrencyToCents(subtotal)
	if err != nil {
		return 0, ErrInvalidGrossAmount
	}
	discountCents, err := parseCurrencyToCents(discount)
	if err != nil {
		return 0, ErrInvalidGrossAmount
	}
	shippingCents, err := parseCurrencyToCents(shipping)
	if err != nil {
		return 0, ErrInvalidGrossAmount
	}

	total := subtotalCents - discountCents + shippingCents
	if total < 0 {
		return 0, nil
	}
	return total, nil
}

func parseMidtransTransactionTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now(), nil
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05-0700",
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	for _, layout := range layouts {
		var parsed time.Time
		var err error
		if layout == "2006-01-02 15:04:05" {
			parsed, err = time.ParseInLocation(layout, raw, loc)
		} else {
			parsed, err = time.Parse(layout, raw)
		}
		if err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, ErrInvalidTransactionTime
}

func timePtr(v time.Time) *time.Time {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

// // Implementasi Complete
// func (s *service) Complete(ctx context.Context, orderID string, userID, nextStatus string) (OrderResponse, error) {
// 	oid, err := uuid.Parse(orderID)
// 	if err != nil {
// 		return OrderResponse{}, ErrInvalidOrderID
// 	}

// 	tx, err := s.db.BeginTx(ctx, nil)
// 	if err != nil {
// 		return OrderResponse{}, err
// 	}
// 	defer tx.Rollback()

// 	qtx := s.repo.WithTx(tx)
// 	order, err := qtx.GetByID(ctx, oid)
// 	if err != nil {
// 		return OrderResponse{}, err
// 	}

// 	if order.UserID != uuid.MustParse(userID) {
// 		return OrderResponse{}, auth.ErrUnauthorized
// 	}

// 	o, err := qtx.UpdateStatus(ctx, oid, nextStatus)
// 	if err != nil {
// 		return OrderResponse{}, err
// 	}

// 	if err := tx.Commit(); err != nil {
// 		return OrderResponse{}, err
// 	}

// 	return s.mapOrderToResponse(o, nil), nil
// }

// Helper Mapper
func (s *service) mapOrderToResponse(o dbgen.Order, items []dbgen.OrderItem) OrderResponse {
	total, _ := strconv.ParseFloat(o.TotalPrice, 64)
	subtotal, _ := strconv.ParseFloat(o.SubtotalPrice, 64)
	shipping, _ := strconv.ParseFloat(o.ShippingPrice, 64)

	res := OrderResponse{
		ID:            o.ID.String(),
		OrderNumber:   o.OrderNumber,
		Status:        o.Status,
		PaymentStatus: o.PaymentStatus,
		SubtotalPrice: subtotal,
		ShippingPrice: shipping,
		TotalPrice:    total,
		PlacedAt:      o.PlacedAt,
	}

	if o.SnapToken.Valid {
		res.SnapToken = &o.SnapToken.String
	}
	if o.SnapRedirectUrl.Valid {
		res.SnapRedirectUrl = &o.SnapRedirectUrl.String
	}

	for _, item := range items {
		uPrice, _ := strconv.ParseFloat(item.UnitPrice, 64)
		sTotal, _ := strconv.ParseFloat(item.TotalPrice, 64)
		res.Items = append(res.Items, OrderItemResponse{
			ID:           item.ID.String(),
			ProductID:    item.ProductID.String(),
			NameSnapshot: item.NameSnapshot,
			UnitPrice:    uPrice,
			Quantity:     item.Quantity,
			Subtotal:     sTotal,
		})
	}
	return res
}
