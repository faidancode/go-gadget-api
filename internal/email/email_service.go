package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Service interface {
	SendResetPasswordEmail(ctx context.Context, to, userName, resetLink string) error
	SendConfirmationLink(ctx context.Context, to, userName, confirmLink string) error
	SendConfirmationPin(ctx context.Context, to, userName, pin string) error
}

type resendService struct {
	apiKey    string
	fromEmail string
	baseURL   string
}

func NewResendServiceFromEnv() (Service, error) {
	apiKey := strings.Trim(os.Getenv("RESEND_API_KEY"), "\"")
	if apiKey == "" {
		return nil, fmt.Errorf("RESEND_API_KEY is not configured")
	}

	from := strings.TrimSpace(strings.Trim(os.Getenv("RESEND_FROM_EMAIL"), "\""))
	if from == "" {
		from = "onboarding@resend.dev"
	}

	return &resendService{
		apiKey:    apiKey,
		fromEmail: from,
		baseURL:   "https://api.resend.com",
	}, nil
}

func NewNoopService() Service {
	return &noopService{}
}

func (s *resendService) SendResetPasswordEmail(ctx context.Context, to, userName, resetLink string) error {
	html := fmt.Sprintf(
		"<p>Halo %s,</p><p>Gunakan link berikut untuk reset password Anda:</p><p><a href=\"%s\">Reset Password</a></p>",
		userName,
		resetLink,
	)
	return s.send(ctx, to, "Reset Password", html)
}

func (s *resendService) SendConfirmationLink(ctx context.Context, to, userName, confirmLink string) error {
	html := fmt.Sprintf(
		"<p>Halo %s,</p><p>Silakan verifikasi email Anda melalui link berikut:</p><p><a href=\"%s\">Verifikasi Email</a></p>",
		userName,
		confirmLink,
	)
	return s.send(ctx, to, "Konfirmasi Email", html)
}

func (s *resendService) SendConfirmationPin(ctx context.Context, to, userName, pin string) error {
	html := fmt.Sprintf(
		"<p>Halo %s,</p><p>Gunakan PIN berikut untuk konfirmasi email:</p><h2>%s</h2>",
		userName,
		pin,
	)
	return s.send(ctx, to, "PIN Konfirmasi Email", html)
}

func (s *resendService) send(ctx context.Context, to, subject, html string) error {
	payload := map[string]any{
		"from":    s.fromEmail,
		"to":      []string{to},
		"subject": subject,
		"html":    html,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(respBody))
		if len(msg) > 500 {
			msg = msg[:500]
		}
		if msg == "" {
			return fmt.Errorf("resend API returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("resend API returned status %d: %s", resp.StatusCode, msg)
	}

	return nil
}

type noopService struct{}

func (s *noopService) SendResetPasswordEmail(_ context.Context, _, _, _ string) error {
	return nil
}

func (s *noopService) SendConfirmationLink(_ context.Context, _, _, _ string) error {
	return nil
}

func (s *noopService) SendConfirmationPin(_ context.Context, _, _, _ string) error {
	return nil
}
