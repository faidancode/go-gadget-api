package order

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"gadget-api/internal/cart"
	"gadget-api/internal/dbgen"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

//go:generate mockgen -source=order_service.go -destination=../mocks/order/order_service_mock.go -package=mock
type Service interface {
	// Customer Actions
	Checkout(ctx context.Context, req CheckoutRequest) (OrderResponse, error)
	List(ctx context.Context, userID string, page, limit int) ([]OrderResponse, int64, error)
	Detail(ctx context.Context, orderID string) (OrderResponse, error)
	Cancel(ctx context.Context, orderID string) error

	// Shared/Admin Actions
	ListAdmin(ctx context.Context, status string, search string, page, limit int) ([]OrderResponse, int64, error)
	UpdateStatus(ctx context.Context, orderID string, status string) (OrderResponse, error)
}

type service struct {
	repo    Repository
	cartSvc cart.Service
	db      *sql.DB        // Dibutuhkan untuk s.db.BeginTx()
	queries *dbgen.Queries // Untuk query standar non-transaksi
}

func NewService(db *sql.DB, r Repository, c cart.Service) Service {
	return &service{
		db:      db,
		repo:    r,
		cartSvc: c,
	}
}

// CUSTOMER: Checkout
func (s *service) Checkout(ctx context.Context, req CheckoutRequest) (OrderResponse, error) {
	// 1. Ambil detail cart (Lakukan di luar transaksi untuk performa)
	cartData, err := s.cartSvc.Detail(ctx, req.UserID)
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
	defer tx.Rollback()

	// 3. Gunakan WithTx untuk mendapatkan instance queries dalam mode transaksi
	qtx := s.repo.WithTx(tx)

	// --- LOGIKA BISNIS ---

	// Hitung total harga
	var total float64
	for _, item := range cartData.Items {
		total += float64(item.Price) * float64(item.Qty)
	}

	uid, _ := uuid.Parse(req.UserID)
	orderNumber := fmt.Sprintf("ORD-%d%s", time.Now().Unix(), strings.ToUpper(uuid.New().String()[:4]))

	// 4. Simpan ke Database (Master Order)
	o, err := qtx.CreateOrder(ctx, dbgen.CreateOrderParams{
		OrderNumber:     orderNumber,
		UserID:          uid,
		Status:          "PENDING",
		AddressSnapshot: json.RawMessage(`{"address_id":"` + req.AddressID + `"}`),
		TotalPrice:      fmt.Sprintf("%.2f", total),
		Note:            dbgen.ToText(req.Note),
	})
	if err != nil {
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

	// 6. Kosongkan Cart
	// Jika cart service menggunakan database yang sama, gunakan qtx
	// Jika cart service adalah service terpisah (microservice), pastikan s.cartSvc.Delete mendukung context
	err = s.cartSvc.Delete(ctx, req.UserID)
	if err != nil {
		return OrderResponse{}, fmt.Errorf("failed to clear cart: %w", err)
	}

	// 7. COMMIT: Simpan semua perubahan secara permanen
	if err := tx.Commit(); err != nil {
		return OrderResponse{}, ErrOrderFailed
	}

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
		Status: dbgen.ToText(status),
		Search: dbgen.ToText(search),
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

// CUSTOMER: Update (DELIVERED -> COMPLETED)
func (s *service) UpdateStatus(ctx context.Context, orderID string, status string) (OrderResponse, error) {
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
