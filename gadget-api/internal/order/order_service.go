package order

import (
	"context"
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
}

func NewService(r Repository, c cart.Service) Service {
	return &service{
		repo:    r,
		cartSvc: c,
	}
}

// CUSTOMER: Checkout
func (s *service) Checkout(ctx context.Context, req CheckoutRequest) (OrderResponse, error) {
	// 1. Dapatkan detail cart
	cartData, err := s.cartSvc.Detail(ctx, req.UserID)
	if err != nil || len(cartData.Items) == 0 {
		return OrderResponse{}, ErrCartEmpty
	}

	// 2. Hitung total (Contoh sederhana, idealnya ada pengecekan stok di sini)
	var total float64
	for _, item := range cartData.Items {
		total += float64(item.Price) * float64(item.Qty)
	}

	uid, _ := uuid.Parse(req.UserID)
	orderNumber := fmt.Sprintf("ORD-%d%s", time.Now().Unix(), strings.ToUpper(uuid.New().String()[:4]))

	// 3. Simpan ke Database
	// Catatan: Idealnya menggunakan DB Transaction jika membuat order + items
	o, err := s.repo.CreateOrder(ctx, dbgen.CreateOrderParams{
		OrderNumber:     orderNumber,
		UserID:          uid,
		Status:          "PENDING",
		AddressSnapshot: json.RawMessage(`{"address_id":"` + req.AddressID + `"}`), // Contoh snapshot sederhana
		TotalPrice:      fmt.Sprintf("%.2f", total),
		Note:            dbgen.ToText(req.Note),
	})
	if err != nil {
		return OrderResponse{}, err
	}

	// 4. Simpan Order Items & Kosongkan Cart
	for _, item := range cartData.Items {
		pID, _ := uuid.Parse(item.ProductID)
		_ = s.repo.CreateOrderItem(ctx, dbgen.CreateOrderItemParams{
			OrderID:      o.ID,
			ProductID:    pID,
			NameSnapshot: "Product Name Placeholder", // Ambil dari info produk asli
			UnitPrice:    fmt.Sprintf("%.2f", float64(item.Price)),
			Quantity:     item.Qty,
			TotalPrice:   fmt.Sprintf("%.2f", float64(item.Price)*float64(item.Qty)),
		})
	}
	_ = s.cartSvc.Delete(ctx, req.UserID)

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
func (s *service) Cancel(ctx context.Context, orderID string) error {
	oid, _ := uuid.Parse(orderID)
	o, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return err
	}
	if o.Status != "PENDING" {
		return ErrCannotCancel
	}
	_, err = s.repo.UpdateStatus(ctx, oid, "CANCELLED")
	return err
}

// CUSTOMER: Update (DELIVERED -> COMPLETED)
func (s *service) UpdateStatus(ctx context.Context, orderID string, status string) (OrderResponse, error) {
	oid, err := uuid.Parse(orderID)
	if err != nil {
		// Jika error, langsung return tanpa memanggil repo
		return OrderResponse{}, ErrInvalidOrderID
	}
	o, err := s.repo.UpdateStatus(ctx, oid, status)
	if err != nil {
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
