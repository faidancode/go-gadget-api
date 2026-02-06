package address_test

import (
	"context"
	"errors"
	"testing"

	"go-gadget-api/internal/address"
	"go-gadget-api/internal/dbgen"
	mockAddress "go-gadget-api/internal/mock/address"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAddressService_ListByUser_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockAddress.NewMockRepository(ctrl)

	svc := address.NewService(nil, repo)

	userID := uuid.New()

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
}

func TestAddressService_ListByUser_Failed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockAddress.NewMockRepository(ctrl)
	svc := address.NewService(nil, repo)

	repo.EXPECT().
		ListByUser(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("db error"))

	_, err := svc.List(context.Background(), uuid.New().String())

	assert.Error(t, err)
}

func TestAddressService_Create_Primary_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockAddress.NewMockRepository(ctrl)

	db, mock, _ := sqlmock.New()
	defer db.Close()

	svc := address.NewService(db, repo)

	userID := uuid.New()

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
}

func TestAddressService_Create_Failed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockAddress.NewMockRepository(ctrl)

	db, mock, _ := sqlmock.New()
	defer db.Close()

	svc := address.NewService(db, repo)

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
}

func TestAddressService_Update_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockAddress.NewMockRepository(ctrl)

	db, mock, _ := sqlmock.New()
	defer db.Close()

	svc := address.NewService(db, repo)

	addrID := uuid.New()
	userID := uuid.New()

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
}

func TestAddressService_Update_Failed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockAddress.NewMockRepository(ctrl)

	db, mock, _ := sqlmock.New()
	defer db.Close()

	svc := address.NewService(db, repo)

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
}

func TestAddressService_Delete_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockAddress.NewMockRepository(ctrl)
	svc := address.NewService(nil, repo)

	repo.EXPECT().
		Delete(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	err := svc.Delete(
		context.Background(),
		uuid.New().String(),
		uuid.New().String(),
	)

	assert.NoError(t, err)
}
func TestAddressService_Delete_Failed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockAddress.NewMockRepository(ctrl)
	svc := address.NewService(nil, repo)

	repo.EXPECT().
		Delete(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("delete failed"))

	err := svc.Delete(
		context.Background(),
		uuid.New().String(),
		uuid.New().String(),
	)

	assert.Error(t, err)
}
