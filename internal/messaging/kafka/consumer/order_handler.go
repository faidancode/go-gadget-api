package consumer

import (
	"context"
	"encoding/json"
	"go-gadget-api/internal/email"
	"go-gadget-api/internal/order"
	"go-gadget-api/internal/shared/database/dbgen"
	"log"

	"github.com/google/uuid"
)

func handleOrderStatusChanged(ctx context.Context, payload []byte, emailSvc email.Service, queries *dbgen.Queries) error {
	var data order.OrderStatusChangedPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		return err
	}

	log.Printf("[CONSUMER] Handling ORDER_STATUS_CHANGED for order: %s", data.OrderNumber)

	userID, err := uuid.Parse(data.UserID)
	if err != nil {
		return err
	}

	user, err := queries.GetUserByID(ctx, userID)
	if err != nil {
		log.Printf("[CONSUMER] Failed to get user for order %s: %v", data.OrderNumber, err)
		return err // Retry later or maybe ignore if user not found, but return err for now
	}

	log.Printf("[CONSUMER] Sending order status email for %s to %s", data.OrderNumber, user.Email)
	err = emailSvc.SendOrderStatusEmail(ctx, user.Email, user.Name, data.OrderNumber, data.NewStatus)
	if err != nil {
		log.Printf("[CONSUMER] Failed to send order status email for %s: %v", data.OrderNumber, err)
		return err
	}

	log.Printf("[CONSUMER] Email sent for ORDER_STATUS_CHANGED: %s", data.OrderNumber)
	return nil
}

func handleOrderPaymentUpdated(ctx context.Context, payload []byte, emailSvc email.Service, queries *dbgen.Queries) error {
	var data order.OrderPaymentUpdatedPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		return err
	}

	log.Printf("[CONSUMER] Handling ORDER_PAYMENT_UPDATED for order: %s", data.OrderNumber)

	userID, err := uuid.Parse(data.UserID)
	if err != nil {
		return err
	}

	user, err := queries.GetUserByID(ctx, userID)
	if err != nil {
		log.Printf("[CONSUMER] Failed to get user for order %s: %v", data.OrderNumber, err)
		return err
	}

	err = emailSvc.SendOrderPaymentEmail(ctx, user.Email, user.Name, data.OrderNumber, data.NewStatus)
	if err != nil {
		log.Printf("[CONSUMER] Failed to send payment update email for %s: %v", data.OrderNumber, err)
		return err
	}

	log.Printf("[CONSUMER] Email sent for ORDER_PAYMENT_UPDATED: %s", data.OrderNumber)
	return nil
}
