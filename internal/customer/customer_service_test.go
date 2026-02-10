package customer_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"go-gadget-api/internal/customer"
	mock "go-gadget-api/internal/mock/customer"
	"go-gadget-api/internal/shared/database/dbgen"
	"go-gadget-api/internal/shared/database/helper"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

func TestCustomerService_UpdateProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, sqlmock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := mock.NewMockRepository(ctrl)
	svc := customer.NewService(db, repo)
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
			Return(repo). // Kembalikan mock yang sama
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

	t.Run("error_invalid_user_id", func(t *testing.T) {
		_, err := svc.UpdateProfile(ctx, "invalid-uuid", customer.UpdateProfileRequest{
			Name: helper.StringPtr("Test"),
		})

		assert.Error(t, err)
	})

	t.Run("error_user_not_found", func(t *testing.T) {
		userID := uuid.New()

		repo.EXPECT().
			GetByID(ctx, userID).
			Return(dbgen.GetUserByIDRow{}, sql.ErrNoRows)

		_, err := svc.UpdateProfile(ctx, userID.String(), customer.UpdateProfileRequest{
			Name: helper.StringPtr("Test"),
		})

		assert.Error(t, err)
	})

	t.Run("error_not_customer_role", func(t *testing.T) {
		userID := uuid.New()

		repo.EXPECT().
			GetByID(ctx, userID).
			Return(dbgen.GetUserByIDRow{
				ID:   userID,
				Role: "ADMIN",
			}, nil)

		_, err := svc.UpdateProfile(ctx, userID.String(), customer.UpdateProfileRequest{
			Name: helper.StringPtr("Test"),
		})

		assert.Error(t, err)
	})

	t.Run("error_update_failed", func(t *testing.T) {
		userID := uuid.New()
		sqlmock.ExpectBegin()
		sqlmock.ExpectCommit()
		repo.EXPECT().
			GetByID(ctx, userID).
			Return(dbgen.GetUserByIDRow{
				ID:   userID,
				Role: "CUSTOMER",
			}, nil)

		repo.EXPECT().WithTx(gomock.Any()).Return(repo).AnyTimes()

		repo.EXPECT().
			UpdateProfile(ctx, dbgen.UpdateCustomerProfileParams{
				ID:   userID,
				Name: helper.StringPtrValue(helper.StringPtr("Test")),
			}).
			Return(dbgen.UpdateCustomerProfileRow{}, errors.New("db error"))

		_, err := svc.UpdateProfile(ctx, userID.String(), customer.UpdateProfileRequest{
			Name: helper.StringPtr("Test"),
		})

		assert.Error(t, err)
	})
}
