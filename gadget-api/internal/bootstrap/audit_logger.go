package bootstrap

import "context"

type AuditLog struct {
	Action  string
	Message string
	Meta    map[string]any
}

type AuditLogger interface {
	Log(ctx context.Context, log AuditLog)
}
