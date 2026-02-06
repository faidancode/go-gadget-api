package bootstrap

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

type StdoutAuditLogger struct{}

func NewStdoutAuditLogger() *StdoutAuditLogger {
	return &StdoutAuditLogger{}
}

func (l *StdoutAuditLogger) Log(ctx context.Context, entry AuditLog) {
	payload := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"action":    entry.Action,
		"message":   entry.Message,
		"meta":      entry.Meta,
	}

	b, _ := json.Marshal(payload)
	log.Println("[AUDIT]", string(b))
}
