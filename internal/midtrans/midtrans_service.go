package midtrans

import (
	"os"

	midtransgo "github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

//go:generate mockgen -source=midtrans_service.go -destination=../mock/midtrans/midtrans_service_mock.go -package=mock
type Service interface {
	CreateTransactionToken(req *CreateTransactionRequest) (*CreateTransactionResponse, error)
}

type service struct {
	client snap.Client
}

func NewService() Service {
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	isProduction := os.Getenv("MIDTRANS_IS_PRODUCTION") == "true"

	var env midtransgo.EnvironmentType
	if isProduction {
		env = midtransgo.Production
	} else {
		env = midtransgo.Sandbox
	}

	c := snap.Client{}
	c.New(serverKey, env)

	return &service{
		client: c,
	}
}

func (s *service) CreateTransactionToken(req *CreateTransactionRequest) (*CreateTransactionResponse, error) {
	snapReq := &snap.Request{
		TransactionDetails: midtransgo.TransactionDetails{
			OrderID:  req.OrderID,
			GrossAmt: req.GrossAmount,
		},
		CustomerDetail: &midtransgo.CustomerDetails{
			FName: req.Customer.FirstName,
			LName: req.Customer.LastName,
			Email: req.Customer.Email,
			Phone: req.Customer.Phone,
		},
	}

	var items []midtransgo.ItemDetails
	for _, item := range req.Items {
		items = append(items, midtransgo.ItemDetails{
			ID:    item.ID,
			Price: item.Price,
			Qty:   item.Qty,
			Name:  item.Name,
		})
	}
	snapReq.Items = &items

	snapResp, err := s.client.CreateTransaction(snapReq)
	if err != nil {
		return nil, err
	}

	return &CreateTransactionResponse{
		Token:       snapResp.Token,
		RedirectURL: snapResp.RedirectURL,
	}, nil
}
