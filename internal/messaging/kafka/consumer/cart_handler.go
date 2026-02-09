package consumer

import (
	"context"
	"encoding/json"
	"go-gadget-api/internal/cart"
	"go-gadget-api/internal/order"
	"log"
)

func handleDeleteCart(ctx context.Context, payload []byte, cartService cart.Service) error {
	var data order.DeleteCartPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		return err
	}

	log.Printf("[CONSUMER] Deleting cart for user: %s", data.UserID)

	if err := cartService.ClearCart(ctx, data.UserID); err != nil {
		return err
	}

	log.Printf("[CONSUMER] Cart deleted successfully for user: %s", data.UserID)
	return nil
}
