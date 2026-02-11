package address_test

import (
	"context"
	"errors"
	"testing"

	"go-gadget-api/internal/address"
	mockAddress "go-gadget-api/internal/mock/address"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAddressService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockAddress.NewMockRepository(ctrl)
	svc := address.NewService(nil, repo)
	userID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		repo.EXPECT().
			ListByUser(gomock.Any(), userID).
			Return([]dbgen.ListAddressesByUserRow{
				{
					ID:        uuid.New(),
					Label:     "Home",
					IsPrimary: true,
				},
			}, nil)

		res, err := svc.List(context.Background(), userID.String())

		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "Home", res[0].Label)
	})

	t.Run("Failed", func(t *testing.T) {
		repo.EXPECT().
			ListByUser(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db error"))

		_, err := svc.List(context.Background(), uuid.New().String())

		assert.Error(t, err)
	})
}

func TestAddressService_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockAddress.NewMockRepository(ctrl)
	svc := address.NewService(nil, repo)
	addrID := uuid.New()
	userID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		repo.EXPECT().
			GetByID(gomock.Any(), addrID, userID).
			Return(dbgen.GetAddressByIDRow{
				ID:    addrID,
				Label: "Home",
			}, nil)

		res, err := svc.GetByID(context.Background(), addrID.String(), userID.String())

		assert.NoError(t, err)
		assert.Equal(t, "Home", res.Label)
		assert.Equal(t, addrID.String(), res.ID)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		repo.EXPECT().
			GetByID(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(dbgen.GetAddressByIDRow{}, errors.New("not found"))

		_, err := svc.GetByID(context.Background(), uuid.New().String(), uuid.New().String())

		assert.Error(t, err)
	})

	t.Run("Failed_InvalidUUID", func(t *testing.T) {
		_, err := svc.GetByID(context.Background(), "invalid-uuid", userID.String())
		assert.Error(t, err)
	})
}

func TestAddressService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockAddress.NewMockRepository(ctrl)
	db, mock, _ := sqlmock.New()
	defer db.Close()

	svc := address.NewService(db, repo)
	userID := uuid.New()

	t.Run("Success_Primary", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectCommit()

		repo.EXPECT().WithTx(gomock.Any()).Return(repo)
		repo.EXPECT().UnsetPrimaryByUser(gomock.Any(), userID).Return(nil)
		repo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(dbgen.Address{
				ID:        uuid.New(),
				Label:     "Home",
				IsPrimary: true,
			}, nil)

		res, err := svc.Create(context.Background(), address.CreateAddressRequest{
			UserID:         userID.String(),
			Label:          "Home",
			RecipientName:  "John",
			RecipientPhone: "08123",
			Street:         "Jl Test",
			IsPrimary:      true,
		})

		assert.NoError(t, err)
		assert.True(t, res.IsPrimary)
	})

	t.Run("Failed", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectRollback()

		repo.EXPECT().WithTx(gomock.Any()).Return(repo)
		repo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(dbgen.Address{}, errors.New("insert failed"))

		_, err := svc.Create(context.Background(), address.CreateAddressRequest{
			UserID: uuid.New().String(),
			Label:  "Home",
		})

		assert.Error(t, err)
	})
}

func TestAddressService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockAddress.NewMockRepository(ctrl)
	db, mock, _ := sqlmock.New()
	defer db.Close()

	svc := address.NewService(db, repo)
	addrID := uuid.New()
	userID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectCommit()

		repo.EXPECT().WithTx(gomock.Any()).Return(repo)
		repo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			Return(dbgen.Address{
				ID:        addrID,
				Label:     "Office",
				IsPrimary: false,
			}, nil)

		res, err := svc.Update(
			context.Background(),
			addrID.String(),
			userID.String(),
			address.UpdateAddressRequest{
				Label: "Office",
			},
		)

		assert.NoError(t, err)
		assert.Equal(t, "Office", res.Label)
	})

	t.Run("Failed", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectRollback()

		repo.EXPECT().WithTx(gomock.Any()).Return(repo)
		repo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			Return(dbgen.Address{}, errors.New("update failed"))

		_, err := svc.Update(
			context.Background(),
			uuid.New().String(),
			uuid.New().String(),
			address.UpdateAddressRequest{},
		)

		assert.Error(t, err)
	})
}

func TestAddressService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockAddress.NewMockRepository(ctrl)
	svc := address.NewService(nil, repo)

	t.Run("Success", func(t *testing.T) {
		repo.EXPECT().
			Delete(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)

		err := svc.Delete(
			context.Background(),
			uuid.New().String(),
			uuid.New().String(),
		)

		assert.NoError(t, err)
	})

	t.Run("Failed", func(t *testing.T) {
		repo.EXPECT().
			Delete(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("delete failed"))

		err := svc.Delete(
			context.Background(),
			uuid.New().String(),
			uuid.New().String(),
		)

		assert.Error(t, err)
	})
}
