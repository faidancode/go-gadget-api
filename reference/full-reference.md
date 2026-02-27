## Query
-- name: CreateOrder :one
INSERT INTO orders (
    order_number, user_id, status, address_snapshot, 
    subtotal_price, shipping_price, total_price, note, placed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
RETURNING *;

-- name: CreateOrderItem :exec
INSERT INTO order_items (
    order_id, product_id, name_snapshot, unit_price, quantity, total_price
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListOrders :many
SELECT o.*, count(*) OVER() AS total_count
FROM orders o
WHERE o.user_id = sqlc.arg('user_id')
  AND (sqlc.narg('status')::text IS NULL OR o.status = sqlc.narg('status')::text)
ORDER BY o.placed_at DESC
LIMIT $1 OFFSET $2;

-- name: ListOrdersAdmin :many
SELECT o.*, count(*) OVER() AS total_count
FROM orders o
WHERE (sqlc.narg('status')::text IS NULL OR o.status = sqlc.narg('status')::text)
  AND (sqlc.narg('search')::text IS NULL OR o.order_number ILIKE '%' || sqlc.narg('search')::text || '%')
ORDER BY o.placed_at DESC
LIMIT $1 OFFSET $2;

-- name: GetOrderByID :one
SELECT * FROM orders WHERE id = $1 LIMIT 1;

-- name: GetOrderItems :many
SELECT * FROM order_items WHERE order_id = $1;

-- name: UpdateOrderStatus :one
UPDATE orders 
SET status = $2, 
    updated_at = NOW(),
    completed_at = CASE WHEN $2 = 'COMPLETED' THEN NOW() ELSE completed_at END,
    cancelled_at = CASE WHEN $2 = 'CANCELLED' THEN NOW() ELSE cancelled_at END
WHERE id = $1
RETURNING *;


## Migration
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number VARCHAR(32) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id),
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING', -- PENDING, PAID, SHIPPING, DELIVERED, COMPLETED, CANCELLED
    payment_method VARCHAR(32),
    payment_status VARCHAR(16) NOT NULL DEFAULT 'UNPAID',
    address_snapshot JSONB NOT NULL,
    subtotal_price DECIMAL(12,2) NOT NULL DEFAULT 0,
    discount_price DECIMAL(12,2) NOT NULL DEFAULT 0,
    shipping_price DECIMAL(12,2) NOT NULL DEFAULT 0,
    total_price DECIMAL(12,2) NOT NULL DEFAULT 0,
    note VARCHAR(255),
    placed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMP,
    cancelled_at TIMESTAMP,
    cancel_reason VARCHAR(100),
    completed_at TIMESTAMP,
    receipt_no VARCHAR(50) UNIQUE,
    snap_token VARCHAR(255),
    snap_redirect_url VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    name_snapshot VARCHAR(200) NOT NULL,
    unit_price DECIMAL(12,2) NOT NULL,
    quantity INTEGER NOT NULL,
    total_price DECIMAL(12,2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_user_status ON orders (user_id, status);
CREATE INDEX idx_order_items_order ON order_items (order_id);


## Response
package response

import (
	"github.com/gin-gonic/gin"
)

type PaginationMeta struct {
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"totalPages,omitempty"`
	Page       int   `json:"page,omitempty"`
	PageSize   int   `json:"pageSize,omitempty"`
}

type ApiEnvelope struct {
	Success bool                   `json:"success"`
	Data    interface{}            `json:"data"`
	Meta    *PaginationMeta        `json:"meta"`
	Error   map[string]interface{} `json:"error"`
}

func Success(c *gin.Context, status int, data interface{}, meta *PaginationMeta) {
	c.JSON(status, ApiEnvelope{
		Success: true,
		Data:    data,
		Meta:    meta,
		Error:   nil,
	})
}

func Error(c *gin.Context, status int, errorCode string, message string, details interface{}) {
	c.JSON(status, ApiEnvelope{
		Success: false,
		Data:    nil,
		Meta:    nil,
		Error: map[string]interface{}{
			"code":    errorCode,
			"message": message,
			"details": details,
		},
	})
}

## Repo
package order

import (
	"context"
	"database/sql"
	"gadget-api/internal/dbgen"

	"github.com/google/uuid"
)

//go:generate mockgen -source=order_repo.go -destination=../mock/order/order_repo_mock.go -package=mock
type Repository interface {
	WithTx(tx dbgen.DBTX) Repository
	CreateOrder(ctx context.Context, arg dbgen.CreateOrderParams) (dbgen.Order, error)
	CreateOrderItem(ctx context.Context, arg dbgen.CreateOrderItemParams) error
	GetByID(ctx context.Context, id uuid.UUID) (dbgen.Order, error)
	GetItems(ctx context.Context, orderID uuid.UUID) ([]dbgen.OrderItem, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (dbgen.Order, error)
	List(ctx context.Context, arg dbgen.ListOrdersParams) ([]dbgen.ListOrdersRow, error)
	ListAdmin(ctx context.Context, arg dbgen.ListOrdersAdminParams) ([]dbgen.ListOrdersAdminRow, error)
}

type repository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{queries: q}
}

func (r *repository) WithTx(tx dbgen.DBTX) Repository {
	// Lakukan type assertion dari interface dbgen.DBTX ke *sql.Tx
	// Karena s.db.BeginTx(ctx, nil) di service menghasilkan *sql.Tx
	if sqlTx, ok := tx.(*sql.Tx); ok {
		return &repository{
			queries: r.queries.WithTx(sqlTx),
		}
	}

	// Jika gagal (misal yang dipassing adalah *sql.DB),
	// Anda bisa mengembalikan repository standar atau menangani error-nya
	return r
}

func (r *repository) CreateOrder(ctx context.Context, arg dbgen.CreateOrderParams) (dbgen.Order, error) {
	return r.queries.CreateOrder(ctx, arg)
}

func (r *repository) CreateOrderItem(ctx context.Context, arg dbgen.CreateOrderItemParams) error {
	return r.queries.CreateOrderItem(ctx, arg)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (dbgen.Order, error) {
	return r.queries.GetOrderByID(ctx, id)
}

func (r *repository) GetItems(ctx context.Context, orderID uuid.UUID) ([]dbgen.OrderItem, error) {
	return r.queries.GetOrderItems(ctx, orderID)
}

func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (dbgen.Order, error) {
	return r.queries.UpdateOrderStatus(ctx, dbgen.UpdateOrderStatusParams{
		ID:     id,
		Status: status,
	})
}

func (r *repository) List(ctx context.Context, arg dbgen.ListOrdersParams) ([]dbgen.ListOrdersRow, error) {
	return r.queries.ListOrders(ctx, arg)
}

func (r *repository) ListAdmin(ctx context.Context, arg dbgen.ListOrdersAdminParams) ([]dbgen.ListOrdersAdminRow, error) {
	return r.queries.ListOrdersAdmin(ctx, arg)
}


## DTO
package order

import "time"

// ==================== REQUEST STRUCTS ====================

type CheckoutRequest struct {
	UserID    string `json:"-"`
	AddressID string `json:"address_id" binding:"required"`
	Note      string `json:"note"`
}

type ListOrderRequest struct {
	UserID string `json:"-"`
	Page   int32  `json:"page"`
	Limit  int32  `json:"limit"`
	Status string `json:"status"` // filter by status
}

type ListOrderAdminRequest struct {
	Page   int32  `json:"page"`
	Limit  int32  `json:"limit"`
	Status string `json:"status"`  // filter by status
	UserID string `json:"user_id"` // filter by user
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"` // PAID, SHIPPED, DELIVERED, CANCELLED
}

// ==================== RESPONSE STRUCTS ====================

type CheckoutResponse struct {
	ID          string    `json:"id"`
	OrderNumber string    `json:"order_number"`
	Status      string    `json:"status"`
	TotalPrice  float64   `json:"total_price"`
	PlacedAt    time.Time `json:"placed_at"`
}

type OrderResponse struct {
	ID          string              `json:"id"`
	OrderNumber string              `json:"order_number"`
	Status      string              `json:"status"`
	TotalPrice  float64             `json:"total_price"`
	PlacedAt    time.Time           `json:"placed_at"`
	Items       []OrderItemResponse `json:"items,omitempty"`
}

type OrderItemResponse struct {
	ProductID    string  `json:"product_id"`
	NameSnapshot string  `json:"name"`
	UnitPrice    float64 `json:"unit_price"`
	Quantity     int32   `json:"quantity"`
	Subtotal     float64 `json:"subtotal"` // unit_price * quantity
}

type OrderDetailResponse struct {
	ID          string              `json:"id"`
	OrderNumber string              `json:"order_number"`
	UserID      string              `json:"user_id"`
	Status      string              `json:"status"`
	TotalPrice  float64             `json:"total_price"`
	Note        string              `json:"note"`
	PlacedAt    time.Time           `json:"placed_at"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Items       []OrderItemResponse `json:"items"`
}

type ListOrderResponse struct {
	Orders []OrderResponse `json:"orders"`
	Total  int64           `json:"total"`
	Page   int32           `json:"page"`
	Limit  int32           `json:"limit"`
}

type ListOrderAdminResponse struct {
	Orders []OrderAdminResponse `json:"orders"`
	Total  int64                `json:"total"`
	Page   int32                `json:"page"`
	Limit  int32                `json:"limit"`
}

type OrderAdminResponse struct {
	ID          string    `json:"id"`
	OrderNumber string    `json:"order_number"`
	UserID      string    `json:"user_id"`
	UserEmail   string    `json:"user_email,omitempty"` // jika perlu join dengan user table
	Status      string    `json:"status"`
	TotalPrice  float64   `json:"total_price"`
	PlacedAt    time.Time `json:"placed_at"`
}


## Service
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



## Service Test
package order_test

import (
	"context"
	"database/sql"
	"gadget-api/internal/cart"
	cartMock "gadget-api/internal/cart/mock"
	"gadget-api/internal/dbgen"
	orderMock "gadget-api/internal/mock/order"
	"gadget-api/internal/order"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestOrderService_Checkout(t *testing.T) {
	ctrl := gomock.NewHandler(t)
	defer ctrl.Finish()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)

	// Sekarang menyertakan DB untuk keperluan transaksi
	svc := order.NewService(db, orderRepo, cartSvc)
	ctx := context.Background()

	t.Run("success_checkout", func(t *testing.T) {
		userID := uuid.New()
		productID := uuid.New()
		orderID := uuid.New()

		// --- SQL Mock Expectations ---
		mock.ExpectBegin()
		mock.ExpectCommit()

		// --- Repo Mock Expectations ---
		// PENTING: Mock WithTx agar tidak mengembalikan nil
		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo).AnyTimes()

		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{
				Items: []cart.CartItemDetailResponse{
					{
						ProductID: productID.String(),
						Qty:       2,
						Price:     5000,
					},
				},
			}, nil)

		orderRepo.EXPECT().
			CreateOrder(gomock.Any(), gomock.Any()).
			Return(dbgen.Order{
				ID:          orderID,
				OrderNumber: "ORD-123",
				UserID:      userID,
				Status:      "PENDING",
				TotalPrice:  "10000.00",
			}, nil)

		orderRepo.EXPECT().
			CreateOrderItem(gomock.Any(), gomock.Any()).
			Return(nil)

		cartSvc.EXPECT().
			Delete(gomock.Any(), userID.String()).
			Return(nil)

		// Execute
		res, err := svc.Checkout(ctx, order.CheckoutRequest{
			UserID:    userID.String(),
			AddressID: "addr-1",
		})

		assert.NoError(t, err)
		assert.Equal(t, "ORD-123", res.OrderNumber)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_create_order_failed_should_rollback", func(t *testing.T) {
		userID := uuid.New()

		// --- SQL Mock: Expect Begin and then Rollback because of error ---
		mock.ExpectBegin()
		mock.ExpectRollback()

		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo).AnyTimes()

		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{
				Items: []cart.CartItemDetailResponse{{ProductID: uuid.New().String(), Qty: 1, Price: 1000}},
			}, nil)

		// Simulate error in DB
		orderRepo.EXPECT().
			CreateOrder(gomock.Any(), gomock.Any()).
			Return(dbgen.Order{}, assert.AnError)

		_, err := svc.Checkout(ctx, order.CheckoutRequest{
			UserID: userID.String(),
		})

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_cart_empty", func(t *testing.T) {
		userID := uuid.New()

		// Tidak ada mock.ExpectBegin karena fungsi return sebelum transaksi mulai
		cartSvc.EXPECT().
			Detail(gomock.Any(), userID.String()).
			Return(cart.CartDetailResponse{Items: []cart.CartItemDetailResponse{}}, nil)

		_, err := svc.Checkout(ctx, order.CheckoutRequest{UserID: userID.String()})

		assert.ErrorIs(t, err, order.ErrCartEmpty)
	})
}

func TestOrderService_List(t *testing.T) {
	ctrl := gomock.NewHandler(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	svc := order.NewService(db, orderRepo, cartSvc)
	ctx := context.Background()

	t.Run("success_list_orders", func(t *testing.T) {
		userID := uuid.New()
		orderID1 := uuid.New()
		orderID2 := uuid.New()

		mockRows := []dbgen.ListOrdersRow{
			{ID: orderID1, OrderNumber: "ORD-001", UserID: userID, Status: "PENDING", TotalPrice: "10000.00", TotalCount: 2},
			{ID: orderID2, OrderNumber: "ORD-002", UserID: userID, Status: "COMPLETED", TotalPrice: "20000.00", TotalCount: 2},
		}

		orderRepo.EXPECT().
			List(gomock.Any(), gomock.Any()).
			Return(mockRows, nil)

		res, total, err := svc.List(ctx, userID.String(), 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, res, 2)
	})

	t.Run("error_list_orders", func(t *testing.T) {
		userID := uuid.New()
		orderRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

		_, _, err := svc.List(ctx, userID.String(), 1, 10)
		assert.Error(t, err)
	})
}

func TestOrderService_ListAdmin(t *testing.T) {
	ctrl := gomock.NewHandler(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	svc := order.NewService(db, orderRepo, cartSvc)
	ctx := context.Background()

	t.Run("success_list_all_orders", func(t *testing.T) {
		orderRepo.EXPECT().
			ListAdmin(gomock.Any(), gomock.Any()).
			Return([]dbgen.ListOrdersAdminRow{
				{ID: uuid.New(), OrderNumber: "ORD-001", TotalCount: 1},
			}, nil)

		res, total, err := svc.ListAdmin(ctx, "", "", 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, res, 1)
	})
}

func TestOrderService_Detail(t *testing.T) {
	ctrl := gomock.NewHandler(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	svc := order.NewService(db, orderRepo, cartSvc)
	ctx := context.Background()

	t.Run("success_get_detail", func(t *testing.T) {
		orderID := uuid.New()
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(dbgen.Order{ID: orderID, OrderNumber: "ORD-123"}, nil)
		orderRepo.EXPECT().GetItems(gomock.Any(), orderID).Return([]dbgen.OrderItem{}, nil)

		res, err := svc.Detail(ctx, orderID.String())
		assert.NoError(t, err)
		assert.Equal(t, "ORD-123", res.OrderNumber)
	})

	t.Run("error_order_not_found", func(t *testing.T) {
		orderID := uuid.New()
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(dbgen.Order{}, sql.ErrNoRows)

		_, err := svc.Detail(ctx, orderID.String())
		assert.Error(t, err)
	})
}

func TestOrderService_Cancel(t *testing.T) {
	ctrl := gomock.NewHandler(t)
	defer ctrl.Finish()

	db, mock, _ := sqlmock.New()
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	svc := order.NewService(db, orderRepo, cartSvc)
	ctx := context.Background()

	t.Run("success_cancel_order", func(t *testing.T) {
		orderID := uuid.New()

		// 1. Mock GetByID (DILUAR/SEBELUM transaksi)
		orderRepo.EXPECT().
			GetByID(gomock.Any(), orderID).
			Return(dbgen.Order{
				ID: orderID, Status: "PENDING",
			}, nil)

		// 2. Setup Transaction Mock (Setelah GetByID)
		mock.ExpectBegin()

		// 3. Mock WithTx dan UpdateStatus (DIDALAM transaksi)
		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo).AnyTimes()
		orderRepo.EXPECT().
			UpdateStatus(gomock.Any(), orderID, "CANCELLED").
			Return(dbgen.Order{}, nil)

		mock.ExpectCommit()

		// Execute
		err := svc.Cancel(ctx, orderID.String())

		// Assert
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error_order_not_pending", func(t *testing.T) {
		orderID := uuid.New()
		// Tidak ada BeginTx karena divalidasi sebelum transaksi (opsional, tergantung logic service Anda)
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(dbgen.Order{
			ID: orderID, Status: "COMPLETED",
		}, nil)

		err := svc.Cancel(ctx, orderID.String())
		assert.ErrorIs(t, err, order.ErrCannotCancel)
	})
}

func TestOrderService_UpdateStatus(t *testing.T) {
	ctrl := gomock.NewHandler(t)
	defer ctrl.Finish()

	db, mock, _ := sqlmock.New()
	defer db.Close()

	orderRepo := orderMock.NewMockRepository(ctrl)
	cartSvc := cartMock.NewMockService(ctrl)
	svc := order.NewService(db, orderRepo, cartSvc)
	ctx := context.Background()

	t.Run("success_update_status", func(t *testing.T) {
		orderID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectCommit()
		orderRepo.EXPECT().WithTx(gomock.Any()).Return(orderRepo).AnyTimes()

		orderRepo.EXPECT().
			UpdateStatus(gomock.Any(), orderID, "COMPLETED").
			Return(dbgen.Order{ID: orderID, Status: "COMPLETED", OrderNumber: "ORD-123"}, nil)

		res, err := svc.UpdateStatus(ctx, orderID.String(), "COMPLETED")

		assert.NoError(t, err)
		assert.Equal(t, "COMPLETED", res.Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}


## Controller
package order

import (
	"gadget-api/internal/pkg/apperror"
	"gadget-api/internal/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service Service
}

func NewHandler(svc Service) *Controller {
	return &Controller{service: svc}
}

// ==================== CUSTOMER ENDPOINTS ====================

// Checkout creates a new order from user's cart
// POST /orders
func (ctrl *Controller) Checkout(c *gin.Context) {
	var req CheckoutRequest

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		)
		return
	}
	req.UserID = userID.(string)

	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperror.Wrap(
			err,
			apperror.CodeInvalidInput,
			"Invalid request body",
			http.StatusBadRequest,
		)
		httpErr := apperror.ToHTTP(appErr)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, err.Error())
		return
	}

	res, err := h.service.Checkout(c.Request.Context(), req)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusCreated, res, nil)
}

// List retrieves all orders for the authenticated user
// GET /orders?page=1&limit=10
func (ctrl *Controller) List(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	orders, total, err := h.service.List(
		c.Request.Context(),
		userID.(string),
		page,
		limit,
	)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"orders": orders,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	}, nil)
}

// Detail retrieves a single order by ID
// GET /orders/:id
func (ctrl *Controller) Detail(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		httpErr := apperror.ToHTTP(ErrInvalidOrderID)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	res, err := h.service.Detail(c.Request.Context(), orderID)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

// Cancel cancels an order (only for PENDING status)
// PATCH /orders/:id/cancel
func (ctrl *Controller) Cancel(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		httpErr := apperror.ToHTTP(ErrInvalidOrderID)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	if err := h.service.Cancel(c.Request.Context(), orderID); err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Order cancelled successfully",
	}, nil)
}

// ==================== ADMIN ENDPOINTS ====================

// ListAdmin retrieves all orders with filters (admin only)
// GET /admin/orders
func (ctrl *Controller) ListAdmin(c *gin.Context) {
	status := c.Query("status")
	search := c.Query("search")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	orders, total, err := h.service.ListAdmin(
		c.Request.Context(),
		status,
		search,
		page,
		limit,
	)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"orders": orders,
		"pagination": gin.H{
			"page":   page,
			"limit":  limit,
			"total":  total,
			"status": status,
			"search": search,
		},
	}, nil)
}

// UpdateStatus updates order status (admin only)
// PATCH /admin/orders/:id/status
func (ctrl *Controller) UpdateStatus(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		httpErr := apperror.ToHTTP(ErrInvalidOrderID)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperror.Wrap(
			err,
			apperror.CodeInvalidInput,
			"Invalid request body",
			http.StatusBadRequest,
		)
		httpErr := apperror.ToHTTP(appErr)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, err.Error())
		return
	}

	res, err := h.service.UpdateStatus(
		c.Request.Context(),
		orderID,
		req.Status,
	)
	if err != nil {
		httpErr := apperror.ToHTTP(err)
		response.Error(c, httpErr.Status, httpErr.Code, httpErr.Message, nil)
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}


## Controller Test
package order_test

import (
	"context"
	"errors"
	"gadget-api/internal/order"
	"gadget-api/internal/pkg/apperror"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ==================== FAKE SERVICE ====================

type fakeOrderService struct {
	checkoutFunc     func(ctx context.Context, req order.CheckoutRequest) (order.OrderResponse, error)
	listFunc         func(ctx context.Context, userID string, page, limit int) ([]order.OrderResponse, int64, error)
	detailFunc       func(ctx context.Context, orderID string) (order.OrderResponse, error)
	cancelFunc       func(ctx context.Context, orderID string) error
	listAdminFunc    func(ctx context.Context, status string, search string, page, limit int) ([]order.OrderResponse, int64, error)
	updateStatusFunc func(ctx context.Context, orderID string, status string) (order.OrderResponse, error)
}

func (f *fakeOrderService) Checkout(ctx context.Context, req order.CheckoutRequest) (order.OrderResponse, error) {
	if f.checkoutFunc != nil {
		return f.checkoutFunc(ctx, req)
	}
	return order.OrderResponse{}, nil
}

func (f *fakeOrderService) List(ctx context.Context, userID string, page, limit int) ([]order.OrderResponse, int64, error) {
	if f.listFunc != nil {
		return f.listFunc(ctx, userID, page, limit)
	}
	return []order.OrderResponse{}, 0, nil
}

func (f *fakeOrderService) Detail(ctx context.Context, orderID string) (order.OrderResponse, error) {
	if f.detailFunc != nil {
		return f.detailFunc(ctx, orderID)
	}
	return order.OrderResponse{}, nil
}

func (f *fakeOrderService) Cancel(ctx context.Context, orderID string) error {
	if f.cancelFunc != nil {
		return f.cancelFunc(ctx, orderID)
	}
	return nil
}

func (f *fakeOrderService) ListAdmin(ctx context.Context, status string, search string, page, limit int) ([]order.OrderResponse, int64, error) {
	if f.listAdminFunc != nil {
		return f.listAdminFunc(ctx, status, search, page, limit)
	}
	return []order.OrderResponse{}, 0, nil
}

func (f *fakeOrderService) UpdateStatus(ctx context.Context, orderID string, status string) (order.OrderResponse, error) {
	if f.updateStatusFunc != nil {
		return f.updateStatusFunc(ctx, orderID, status)
	}
	return order.OrderResponse{}, nil
}

// ==================== HELPER FUNCTIONS ====================

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func newTestController(svc order.Service) *order.Handler {
	return order.NewHandler(svc)
}

// ==================== CHECKOUT TESTS ====================

func TestOrderController_Checkout(t *testing.T) {
	t.Run("success_checkout", func(t *testing.T) {
		orderID := uuid.New().String()
		userID := uuid.New().String()

		svc := &fakeOrderService{
			checkoutFunc: func(ctx context.Context, req order.CheckoutRequest) (order.OrderResponse, error) {
				assert.Equal(t, userID, req.UserID)
				assert.Equal(t, "addr-123", req.AddressID)

				return order.OrderResponse{
					ID:          orderID,
					OrderNumber: "ORD-999",
					Status:      "PENDING",
					TotalPrice:  150000.00,
					PlacedAt:    time.Now(),
				}, nil
			},
		}

		ctrl := newTestController(svc)
		r := setupTestRouter()
		r.POST("/orders", ctrl.Checkout)

		body := `{"address_id": "addr-123", "note": "Please deliver in the morning"}`
		req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// Simulate middleware setting user_id
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", userID)

		ctrl.Checkout(c)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-999")
		assert.Contains(t, w.Body.String(), "PENDING")
	})

	t.Run("invalid_json_payload", func(t *testing.T) {
		ctrl := newTestController(&fakeOrderService{})
		r := setupTestRouter()
		r.POST("/orders", func(c *gin.Context) {
			c.Set("user_id", "some-user-id") // Set user_id supaya lolos cek auth
			ctrl.Checkout(c)
		})

		req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{invalid-json}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing_required_fields", func(t *testing.T) {
		ctrl := newTestController(&fakeOrderService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"note": "test"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Checkout(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("cart_is_empty", func(t *testing.T) {
		svc := &fakeOrderService{
			checkoutFunc: func(ctx context.Context, req order.CheckoutRequest) (order.OrderResponse, error) {
				return order.OrderResponse{}, order.ErrCartEmpty
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"address_id": "addr-123"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Checkout(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service_internal_error", func(t *testing.T) {
		svc := &fakeOrderService{
			checkoutFunc: func(ctx context.Context, req order.CheckoutRequest) (order.OrderResponse, error) {
				return order.OrderResponse{}, errors.New("database connection failed")
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"address_id": "addr-123"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("user_id", uuid.New().String())

		ctrl.Checkout(c)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ==================== LIST ORDERS TESTS ====================

func TestOrderController_List(t *testing.T) {
	t.Run("success_list_user_orders", func(t *testing.T) {
		userID := uuid.New().String()

		svc := &fakeOrderService{
			listFunc: func(ctx context.Context, uid string, page, limit int) ([]order.OrderResponse, int64, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, 1, page)
				assert.Equal(t, 10, limit)

				orders := []order.OrderResponse{
					{
						ID:          uuid.New().String(),
						OrderNumber: "ORD-001",
						Status:      "PENDING",
						TotalPrice:  100000.00,
						PlacedAt:    time.Now(),
					},
					{
						ID:          uuid.New().String(),
						OrderNumber: "ORD-002",
						Status:      "PAID",
						TotalPrice:  200000.00,
						PlacedAt:    time.Now(),
					},
				}
				return orders, 2, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/orders?page=1&limit=10", nil)
		c.Set("user_id", userID)

		ctrl.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-001")
		assert.Contains(t, w.Body.String(), "ORD-002")
	})

	t.Run("success_with_status_filter", func(t *testing.T) {
		userID := uuid.New().String()

		svc := &fakeOrderService{
			listFunc: func(ctx context.Context, uid string, page, limit int) ([]order.OrderResponse, int64, error) {
				// Note: status filter logic should be in controller layer
				orders := []order.OrderResponse{
					{OrderNumber: "ORD-003", Status: "PAID", TotalPrice: 150000.00},
				}
				return orders, 1, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/orders?status=PAID", nil)
		c.Set("user_id", userID)

		ctrl.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "PAID")
	})

	t.Run("empty_orders", func(t *testing.T) {
		svc := &fakeOrderService{
			listFunc: func(ctx context.Context, uid string, page, limit int) ([]order.OrderResponse, int64, error) {
				return []order.OrderResponse{}, 0, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/orders", nil)
		c.Set("user_id", uuid.New().String())

		ctrl.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service_error", func(t *testing.T) {
		svc := &fakeOrderService{
			listFunc: func(ctx context.Context, uid string, page, limit int) ([]order.OrderResponse, int64, error) {
				return nil, 0, errors.New("database error")
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/orders", nil)
		c.Set("user_id", uuid.New().String())

		ctrl.List(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ==================== DETAIL ORDER TESTS ====================

func TestOrderController_Detail(t *testing.T) {
	t.Run("success_get_order_detail", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			detailFunc: func(ctx context.Context, id string) (order.OrderResponse, error) {
				assert.Equal(t, orderID, id)

				return order.OrderResponse{
					ID:          orderID,
					OrderNumber: "ORD-123",
					Status:      "PAID",
					TotalPrice:  250000.00,
					PlacedAt:    time.Now(),
					Items: []order.OrderItemResponse{
						{
							ProductID:    uuid.New().String(),
							NameSnapshot: "Product A",
							UnitPrice:    50000.00,
							Quantity:     2,
							Subtotal:     100000.00,
						},
					},
				}, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/orders/"+orderID, nil)
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.Detail(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-123")
		assert.Contains(t, w.Body.String(), "Product A")
	})

	t.Run("order_not_found", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			detailFunc: func(ctx context.Context, id string) (order.OrderResponse, error) {
				return order.OrderResponse{}, errors.New("order not found")
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/orders/"+orderID, nil)
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.Detail(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// ==================== CANCEL ORDER TESTS ====================

func TestOrderController_Cancel(t *testing.T) {
	t.Run("success_cancel_order", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			cancelFunc: func(ctx context.Context, id string) error {
				assert.Equal(t, orderID, id)
				return nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodPatch, "/orders/"+orderID+"/cancel", nil)
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.Cancel(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("order_already_cancelled", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			cancelFunc: func(ctx context.Context, id string) error {
				return errors.New("order already cancelled")
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodPatch, "/orders/"+orderID+"/cancel", nil)
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.Cancel(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("order_not_found", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			cancelFunc: func(ctx context.Context, id string) error {
				return errors.New("order not found")
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodPatch, "/orders/"+orderID+"/cancel", nil)
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.Cancel(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// ==================== ADMIN LIST ORDERS TESTS ====================

func TestOrderController_ListAdmin(t *testing.T) {
	t.Run("success_list_all_orders", func(t *testing.T) {
		svc := &fakeOrderService{
			listAdminFunc: func(ctx context.Context, status, search string, page, limit int) ([]order.OrderResponse, int64, error) {
				assert.Equal(t, 1, page)
				assert.Equal(t, 20, limit)

				orders := []order.OrderResponse{
					{
						ID:          uuid.New().String(),
						OrderNumber: "ORD-ADM-001",
						Status:      "PAID",
						TotalPrice:  300000.00,
						PlacedAt:    time.Now(),
					},
				}
				return orders, 1, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/admin/orders?page=1&limit=20", nil)

		ctrl.ListAdmin(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-ADM-001")
	})

	t.Run("success_filter_by_user", func(t *testing.T) {
		svc := &fakeOrderService{
			listAdminFunc: func(ctx context.Context, status, search string, page, limit int) ([]order.OrderResponse, int64, error) {
				assert.Equal(t, "user-123", search) // search could be userID or email

				orders := []order.OrderResponse{
					{OrderNumber: "ORD-USR-001", Status: "PAID"},
				}
				return orders, 1, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/admin/orders?search=user-123", nil)

		ctrl.ListAdmin(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success_filter_by_status", func(t *testing.T) {
		svc := &fakeOrderService{
			listAdminFunc: func(ctx context.Context, status, search string, page, limit int) ([]order.OrderResponse, int64, error) {
				assert.Equal(t, "SHIPPED", status)

				orders := []order.OrderResponse{
					{OrderNumber: "ORD-SHP-001", Status: "SHIPPED"},
				}
				return orders, 1, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/admin/orders?status=SHIPPED", nil)

		ctrl.ListAdmin(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ==================== UPDATE STATUS TESTS ====================

func TestOrderController_UpdateStatus(t *testing.T) {
	t.Run("success_update_status", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			updateStatusFunc: func(ctx context.Context, id, status string) (order.OrderResponse, error) {
				assert.Equal(t, orderID, id)
				assert.Equal(t, "SHIPPED", status)

				return order.OrderResponse{
					ID:          id,
					OrderNumber: "ORD-123",
					Status:      status,
					TotalPrice:  100000.00,
					PlacedAt:    time.Now(),
				}, nil
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"status": "SHIPPED"}`
		c.Request = httptest.NewRequest(http.MethodPatch, "/admin/orders/"+orderID+"/status", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.UpdateStatus(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid_status", func(t *testing.T) {
		orderID := uuid.New().String()

		ctrl := newTestController(&fakeOrderService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"status": ""}`
		c.Request = httptest.NewRequest(http.MethodPatch, "/admin/orders/"+orderID+"/status", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.UpdateStatus(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid_json", func(t *testing.T) {
		orderID := uuid.New().String()

		ctrl := newTestController(&fakeOrderService{})
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodPatch, "/admin/orders/"+orderID+"/status", strings.NewReader(`{invalid}`))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.UpdateStatus(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("order_not_found", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			updateStatusFunc: func(ctx context.Context, id, status string) (order.OrderResponse, error) {
				return order.OrderResponse{}, order.ErrOrderNotFound
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"status": "DELIVERED"}`
		c.Request = httptest.NewRequest(http.MethodPatch, "/admin/orders/"+orderID+"/status", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.UpdateStatus(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid_status_transition", func(t *testing.T) {
		orderID := uuid.New().String()

		svc := &fakeOrderService{
			updateStatusFunc: func(ctx context.Context, id, status string) (order.OrderResponse, error) {
				return order.OrderResponse{}, order.ErrInvalidStatusTransition
			},
		}

		ctrl := newTestController(svc)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{"status":"PENDING"}`
		c.Request = httptest.NewRequest(
			http.MethodPatch,
			"/admin/orders/"+orderID+"/status",
			strings.NewReader(body),
		)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: orderID}}

		ctrl.UpdateStatus(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), apperror.CodeInvalidState)
	})

}

