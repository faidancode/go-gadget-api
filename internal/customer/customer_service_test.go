package customer_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"go-gadget-api/internal/customer"
	mockAddress "go-gadget-api/internal/mock/address"
	mockCustomer "go-gadget-api/internal/mock/customer"
	mockOrder "go-gadget-api/internal/mock/order"
	"go-gadget-api/internal/shared/database/dbgen"
	"go-gadget-api/internal/shared/database/helper"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

type updateProfileMatcher struct {
	UserID uuid.UUID
	Name   string
}

func (m updateProfileMatcher) Matches(x interface{}) bool {
	arg, ok := x.(dbgen.UpdateCustomerProfileParams)
	if !ok {
		return false
	}

	return arg.ID == m.UserID &&
		arg.Name == m.Name
}

func (m updateProfileMatcher) String() string {
	return "matches UpdateCustomerProfileParams"
}

func TestCustomerService_ListCustomers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	repo := mockCustomer.NewMockRepository(ctrl)
	addrRepo := mockAddress.NewMockRepository(ctrl)
	orderRepo := mockOrder.NewMockRepository(ctrl)
	svc := customer.NewService(db, repo, addrRepo, orderRepo)
	ctx := context.Background()

	t.Run("success_list_customers", func(t *testing.T) {
		userID := uuid.New()
		repo.EXPECT().
			ListCustomers(ctx, dbgen.ListCustomersParams{
				Limit:  10,
				Offset: 0,
				Search: sql.NullString{},
			}).
			Return([]dbgen.ListCustomersRow{
				{
					ID:         userID,
					Name:       "John Doe",
					Email:      "john@example.com",
					IsActive:   true,
					CreatedAt:  time.Now(),
					TotalCount: 1,
				},
			}, nil)

		resp, total, err := svc.ListCustomers(ctx, 1, 10, "")

		assert.NoError(t, err)
		assert.Len(t, resp, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "John Doe", resp[0].Name)
	})

	t.Run("error_repository_failure", func(t *testing.T) {
		repo.EXPECT().
			ListCustomers(ctx, gomock.Any()).
			Return(nil, errors.New("db error"))

		resp, total, err := svc.ListCustomers(ctx, 1, 10, "")

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, int64(0), total)
	})
}

func TestCustomerService_ToggleCustomerStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	repo := mockCustomer.NewMockRepository(ctrl)
	svc := customer.NewService(db, repo, nil, nil)
	ctx := context.Background()

	t.Run("success_toggle_status", func(t *testing.T) {
		userID := uuid.New()
		repo.EXPECT().
			UpdateStatus(ctx, dbgen.UpdateCustomerStatusParams{
				ID:       userID,
				IsActive: false,
			}).
			Return(dbgen.UpdateCustomerStatusRow{
				ID:        userID,
				Name:      "John Doe",
				IsActive:  false,
				UpdatedAt: time.Now(),
			}, nil)

		resp, err := svc.ToggleCustomerStatus(ctx, userID.String(), false)

		assert.NoError(t, err)
		assert.False(t, resp.IsActive)
	})

	t.Run("error_invalid_uuid", func(t *testing.T) {
		resp, err := svc.ToggleCustomerStatus(ctx, "invalid-uuid", true)

		assert.Error(t, err)
		assert.Empty(t, resp.ID)
	})
}

func TestCustomerService_GetCustomerByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	repo := mockCustomer.NewMockRepository(ctrl)
	svc := customer.NewService(db, repo, nil, nil)
	ctx := context.Background()

	t.Run("success_get_by_id", func(t *testing.T) {
		userID := uuid.New()

		// Mock User
		repo.EXPECT().GetByID(ctx, userID).Return(dbgen.GetUserByIDRow{
			ID:        userID,
			Name:      "John Detail",
			CreatedAt: time.Now(),
		}, nil)

		resp, err := svc.GetCustomerByID(ctx, customer.CustomerDetailsRequest{
			CustomerID: userID.String(),
		})

		assert.NoError(t, err)
		assert.Equal(t, "John Detail", resp.Name)
	})

	t.Run("error_user_not_found", func(t *testing.T) {
		userID := uuid.New()
		repo.EXPECT().GetByID(ctx, userID).Return(dbgen.GetUserByIDRow{}, errors.New("not found"))

		resp, err := svc.GetCustomerByID(ctx, customer.CustomerDetailsRequest{
			CustomerID: userID.String(),
		})

		assert.Error(t, err)
		assert.Empty(t, resp.ID)
	})
}

func TestCustomerService_ListCustomerAddresses(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	addrRepo := mockAddress.NewMockRepository(ctrl)
	svc := customer.NewService(db, nil, addrRepo, nil)
	ctx := context.Background()

	t.Run("success_list_addresses", func(t *testing.T) {
		userID := uuid.New()

		// Mock Addresses
		addrRepo.EXPECT().ListByUser(ctx, dbgen.ListAddressesByUserParams{
			UserID: userID,
			Limit:  10,
			Offset: 0,
			Search: sql.NullString{},
		}).Return([]dbgen.ListAddressesByUserRow{
			{
				ID:         uuid.New(),
				Label:      "Rumah",
				Street:     "Jl. Merdeka",
				IsPrimary:  true,
				TotalCount: 1,
			},
		}, nil)

		resp, err := svc.ListCustomerAddresses(ctx, customer.CustomerAddressesRequest{
			CustomerID: userID.String(),
			Page:       1,
			Limit:      10,
		})

		assert.NoError(t, err)
		assert.Len(t, resp.Data, 1)
		assert.Equal(t, int64(1), resp.Meta.Total)
		assert.Equal(t, "Rumah", resp.Data[0].AddressName)
	})
}

func TestCustomerService_ListCustomerOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, _, _ := sqlmock.New()
	orderRepo := mockOrder.NewMockRepository(ctrl)
	svc := customer.NewService(db, nil, nil, orderRepo)
	ctx := context.Background()

	t.Run("success_list_orders", func(t *testing.T) {
		userID := uuid.New()

		// Mock Orders
		orderRepo.EXPECT().List(ctx, dbgen.ListOrdersParams{
			UserID: userID,
			Limit:  10,
			Offset: 0,
			Search: sql.NullString{},
		}).Return([]dbgen.ListOrdersRow{
			{
				ID:          uuid.New(),
				OrderNumber: "ORD-001",
				TotalPrice:  "150000",
				PlacedAt:    time.Now(),
				TotalCount:  1,
			},
		}, nil)

		resp, err := svc.ListCustomerOrders(ctx, customer.CustomerOrdersRequest{
			CustomerID: userID.String(),
			Page:       1,
			Limit:      10,
		})

		assert.NoError(t, err)
		assert.Len(t, resp.Data, 1)
		assert.Equal(t, int64(1), resp.Meta.Total)
		assert.Equal(t, 150000, resp.Data[0].TotalPrice)
	})
}

func TestCustomerService_UpdateProfile_Complex(t *testing.T) {
	// Skenario Tambahan untuk UpdateProfile (Negative Case Password)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, sqlmock, _ := sqlmock.New()
	repo := mockCustomer.NewMockRepository(ctrl)
	svc := customer.NewService(db, repo, nil, nil)
	ctx := context.Background()

	t.Run("error_wrong_current_password", func(t *testing.T) {
		userID := uuid.New()
		hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("old-password"), 10)

		repo.EXPECT().GetByID(ctx, userID).Return(dbgen.GetUserByIDRow{
			ID:       userID,
			Password: string(hashedPwd),
		}, nil)

		// Mock Transaction
		sqlmock.ExpectBegin()
		repo.EXPECT().WithTx(gomock.Any()).Return(repo)
		sqlmock.ExpectRollback()

		req := customer.UpdateProfileRequest{
			CurrentPassword: helper.StringPtr("wrong-password"),
			Password:        helper.StringPtr("new-password"),
		}

		resp, err := svc.UpdateProfile(ctx, userID.String(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid current password")
		assert.Empty(t, resp.ID)
	})
}

func TestCustomerService_UpdateProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, sqlmock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := mockCustomer.NewMockRepository(ctrl)
	addrRepo := mockAddress.NewMockRepository(ctrl)
	orderRepo := mockOrder.NewMockRepository(ctrl)
	svc := customer.NewService(db, repo, addrRepo, orderRepo)
	ctx := context.Background()

	t.Run("success_update_name_only", func(t *testing.T) {
		userID := uuid.New()
		sqlmock.ExpectBegin()
		sqlmock.ExpectCommit()
		repo.EXPECT().
			GetByID(ctx, userID).
			Return(dbgen.GetUserByIDRow{
				ID:   userID,
				Role: "CUSTOMER",
			}, nil)
		repo.EXPECT().
			WithTx(gomock.Any()).
			Return(repo).
			AnyTimes()

		repo.EXPECT().
			UpdateProfile(
				ctx,
				updateProfileMatcher{
					UserID: userID,
					Name:   "New Name",
				},
			).
			Return(dbgen.UpdateCustomerProfileRow{
				ID:        userID,
				Name:      "New Name",
				UpdatedAt: time.Now(),
			}, nil)

		resp, err := svc.UpdateProfile(ctx, userID.String(), customer.UpdateProfileRequest{
			Name: helper.StringPtr("New Name"),
		})

		assert.NoError(t, err)
		assert.Equal(t, "New Name", resp.Name)
	})
}
