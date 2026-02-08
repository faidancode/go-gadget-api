package order

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-gadget-api/internal/auth"
	"go-gadget-api/internal/cart"
	"go-gadget-api/internal/outbox"
	"go-gadget-api/internal/shared/database/dbgen"
	"go-gadget-api/internal/shared/database/helper"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

//go:generate mockgen -source=order_service.go -destination=../mocks/order/order_service_mock.go -package=mock
type Service interface {
	// Customer Actions
	Checkout(ctx context.Context, userID string, req CheckoutRequest) (OrderResponse, error)
	List(ctx context.Context, userID string, page, limit int) ([]OrderResponse, int64, error)
	Detail(ctx context.Context, orderID string) (OrderResponse, error)
	Cancel(ctx context.Context, orderID string) error
	UpdateStatusByCustomer(ctx context.Context, orderID string, userID uuid.UUID, nextStatus string) (OrderResponse, error)

	// Shared/Admin Actions
	ListAdmin(ctx context.Context, status string, search string, page, limit int) ([]OrderResponse, int64, error)
	UpdateStatusByAdmin(ctx context.Context, orderID string, nextStatus string, receiptNo *string) (OrderResponse, error)
}

type service struct {
	db         *sql.DB
	repo       Repository
	outboxRepo outbox.Repository
	cartSvc    cart.Service
}

type Deps struct {
	DB         *sql.DB
	Repo       Repository
	OutboxRepo outbox.Repository
	CartSvc    cart.Service
}

func NewService(deps Deps) Service {
	if deps.DB == nil {
		panic("db cannot be nil")
	}
	if deps.Repo == nil {
		panic("order repository cannot be nil")
	}
	if deps.OutboxRepo == nil {
		panic("outbox repository cannot be nil") // ← Check ini
	}
	if deps.CartSvc == nil {
		panic("cart service cannot be nil")
	}
	return &service{
		db:         deps.DB,
		repo:       deps.Repo,
		outboxRepo: deps.OutboxRepo,
		cartSvc:    deps.CartSvc,
	}
}

// CUSTOMER: Checkout
func (s *service) Checkout(ctx context.Context, userID string, req CheckoutRequest) (OrderResponse, error) {
	// 1. Ambil detail cart (Lakukan di luar transaksi untuk performa)
	cartData, err := s.cartSvc.Detail(ctx, userID)
	if err != nil {
		return OrderResponse{}, err
	}
	if len(cartData.Items) == 0 {
		return OrderResponse{}, ErrCartEmpty
	}

	// 2. Mulai Transaksi Database
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return OrderResponse{}, ErrOrderFailed
	}

	// Safety: Jika fungsi exit sebelum Commit, maka akan Rollback.
	// Jika sudah Commit, Rollback ini tidak akan melakukan apa-apa.
	committed := false
	defer func() {
		if !committed {
			err = tx.Rollback()
			fmt.Printf("Rollback called, error: %v\n", err)
		}
	}()

	// 3. Gunakan WithTx untuk mendapatkan instance queries dalam mode transaksi
	qtx := s.repo.WithTx(tx)

	// --- LOGIKA BISNIS ---

	// Hitung subtotal harga
	var subtotal float64
	for _, item := range cartData.Items {
		subtotal += float64(item.Price) * float64(item.Qty)
	}
	var shippingPrice float64 = 0.0 // Untuk sekarang, gratis ongkir
	var total float64 = subtotal + shippingPrice

	uid, _ := uuid.Parse(userID)
	var addId uuid.NullUUID
	if req.AddressID != "" {
		parsedID, err := uuid.Parse(req.AddressID)
		if err == nil {
			addId.UUID = parsedID
			addId.Valid = true
		}
	} else {
		addId.Valid = false
	}
	orderNumber := fmt.Sprintf("ORD-%d%s", time.Now().Unix(), strings.ToUpper(uuid.New().String()[:4]))

	// 4. Simpan ke Database (Master Order)
	o, err := qtx.CreateOrder(ctx, dbgen.CreateOrderParams{
		OrderNumber:     orderNumber,
		UserID:          uid,
		Status:          "PENDING",
		AddressID:       addId,
		AddressSnapshot: json.RawMessage(`{"address_id":"` + req.AddressID + `"}`),
		SubtotalPrice:   fmt.Sprintf("%.2f", subtotal),
		ShippingPrice:   fmt.Sprintf("%.2f", shippingPrice),
		TotalPrice:      fmt.Sprintf("%.2f", total),
		Note:            helper.StringToNull(&req.Note),
	})
	if err != nil {
		fmt.Printf("Error creating order: %v\n", err)
		return OrderResponse{}, ErrOrderFailed
	}

	// 5. Simpan Order Items secara loop
	for _, item := range cartData.Items {
		pID, _ := uuid.Parse(item.ProductID)
		err := qtx.CreateOrderItem(ctx, dbgen.CreateOrderItemParams{
			OrderID:      o.ID,
			ProductID:    pID,
			NameSnapshot: "Product Name Placeholder",
			UnitPrice:    fmt.Sprintf("%.2f", float64(item.Price)),
			Quantity:     item.Qty,
			TotalPrice:   fmt.Sprintf("%.2f", float64(item.Price)*float64(item.Qty)),
		})
		if err != nil {
			// Mengembalikan error di sini akan memicu defer tx.Rollback()
			return OrderResponse{}, ErrOrderFailed
		}
	}

	if s.outboxRepo == nil {
		log.Println("CRITICAL: outboxRepo is nil")
		return OrderResponse{}, ErrOrderFailed
	}

	payload, err := json.Marshal(map[string]string{
		"user_id": userID,
	})
	if err != nil {
		return OrderResponse{}, ErrOrderFailed
	}

	err = s.outboxRepo.
		WithTx(tx).
		CreateOutboxEvent(ctx, dbgen.CreateOutboxEventParams{
			ID:            uuid.New(),
			AggregateType: "ORDER",
			AggregateID:   o.ID,
			EventType:     "DELETE_CART",
			Payload:       payload,
		})
	if err != nil {
		return OrderResponse{}, ErrOrderFailed
	}

	// 7. COMMIT: Simpan semua perubahan secara permanen
	if err := tx.Commit(); err != nil {
		return OrderResponse{}, ErrOrderFailed
	}
	committed = true

	return s.mapOrderToResponse(o, nil), nil
}

// CUSTOMER & ADMIN: List
func (s *service) List(ctx context.Context, userID string, page, limit int) ([]OrderResponse, int64, error) {
	uid, _ := uuid.Parse(userID)
	rows, err := s.repo.List(ctx, dbgen.ListOrdersParams{
		UserID: uid,
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	})
	if err != nil {
		return nil, 0, err
	}

	var res []OrderResponse
	var total int64
	for _, r := range rows {
		total = r.TotalCount
		res = append(res, s.mapOrderToResponse(dbgen.Order{
			ID:          r.ID,
			OrderNumber: r.OrderNumber,
			UserID:      r.UserID,
			Status:      r.Status,
			TotalPrice:  r.TotalPrice,
			PlacedAt:    r.PlacedAt,
			CreatedAt:   r.CreatedAt,
		}, nil))
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
		return OrderResponse{}, ErrInvalidOrderID
	}

	o, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return OrderResponse{}, ErrOrderNotFound
	}

	// Disarankan menangani error GetItems juga
	items, err := s.repo.GetItems(ctx, oid)
	if err != nil {
		// Tergantung kebutuhan bisnis, bisa return error atau biarkan items kosong
		return OrderResponse{}, err
	}

	return s.mapOrderToResponse(o, items), nil
}

// CUSTOMER: Cancel
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
		return err
	}

	// Jika ke depannya ada logika kembalikan stok:
	// err = s.productSvc.RestoreStock(ctx, o.Items)
	// if err != nil { return err }

	return tx.Commit()
}

// // CUSTOMER: Update (DELIVERED -> COMPLETED)
// func (s *service) UpdateStatus(ctx context.Context, orderID string, status string) (OrderResponse, error) {
// 	oid, err := uuid.Parse(orderID)
// 	if err != nil {
// 		return OrderResponse{}, ErrInvalidOrderID
// 	}

// 	// 1. Mulai Transaksi
// 	tx, err := s.db.BeginTx(ctx, nil)
// 	if err != nil {
// 		return OrderResponse{}, err
// 	}
// 	defer tx.Rollback()

// 	// 2. Hubungkan Repository dengan Transaksi
// 	qtx := s.repo.WithTx(tx)

// 	// 3. Eksekusi Update Status
// 	o, err := qtx.UpdateStatus(ctx, oid, status)
// 	if err != nil {
// 		// Jika error (misal: order tidak ketemu atau DB error)
// 		return OrderResponse{}, err
// 	}

// 	// --- LOGIKA TAMBAHAN (Opsional di masa depan) ---
// 	// Jika status == "SHIPPED", mungkin Anda ingin otomatis kirim email/notifikasi
// 	// if status == "SHIPPED" {
// 	//    s.notificationSvc.Send(o.UserID, "Pesanan Anda sedang dikirim!")
// 	// }

// 	// 4. Commit Transaksi
// 	if err := tx.Commit(); err != nil {
// 		return OrderResponse{}, err
// 	}

// 	return s.mapOrderToResponse(o, nil), nil
// }

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

// Implementasi UpdateStatusByCustomer
func (s *service) UpdateStatusByCustomer(ctx context.Context, orderID string, userID uuid.UUID, nextStatus string) (OrderResponse, error) {
	oid, err := uuid.Parse(orderID)
	if err != nil {
		return OrderResponse{}, fmt.Errorf("invalid order id")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return OrderResponse{}, err
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)
	order, err := qtx.GetByID(ctx, oid)
	if err != nil {
		return OrderResponse{}, err
	}

	if order.UserID != userID {
		return OrderResponse{}, auth.ErrUnauthorized
	}

	o, err := qtx.UpdateStatus(ctx, oid, nextStatus)
	if err != nil {
		return OrderResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return OrderResponse{}, err
	}

	return s.mapOrderToResponse(o, nil), nil
}

// Helper Mapper
func (s *service) mapOrderToResponse(o dbgen.Order, items []dbgen.OrderItem) OrderResponse {
	total, _ := strconv.ParseFloat(o.TotalPrice, 64)
	res := OrderResponse{
		ID:          o.ID.String(),
		OrderNumber: o.OrderNumber,
		Status:      o.Status,
		TotalPrice:  total,
		PlacedAt:    o.PlacedAt,
	}

	for _, item := range items {
		uPrice, _ := strconv.ParseFloat(item.UnitPrice, 64)
		res.Items = append(res.Items, OrderItemResponse{
			ProductID:    item.ProductID.String(),
			NameSnapshot: item.NameSnapshot,
			UnitPrice:    uPrice,
			Quantity:     item.Quantity,
		})
	}
	return res
}
