package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	autherrors "go-gadget-api/internal/auth/errors"
	"go-gadget-api/internal/email"
	platform "go-gadget-api/internal/pkg/request"
	"go-gadget-api/internal/shared/database/dbgen"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo     Repository
	emailSvc email.Service
}

func NewService(repo Repository, emailSvc email.Service) *Service {
	return &Service{
		repo:     repo,
		emailSvc: emailSvc,
	}
}

func (s *Service) Login(ctx context.Context, email, password string) (string, string, AuthResponse, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", AuthResponse{}, autherrors.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", "", AuthResponse{}, autherrors.ErrInvalidCredentials
	}

	accessToken, err := s.generateToken(user.ID.String(), user.Role, time.Minute*15)
	if err != nil {
		return "", "", AuthResponse{}, autherrors.ErrTokenGenerationFailed
	}

	refreshToken, err := s.generateToken(user.ID.String(), user.Role, time.Hour*24*7)
	if err != nil {
		return "", "", AuthResponse{}, autherrors.ErrTokenGenerationFailed
	}

	return accessToken, refreshToken, AuthResponse{
		ID:    user.ID.String(),
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, string, AuthResponse, error) {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, autherrors.ErrInvalidToken
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return "", "", AuthResponse{}, autherrors.ErrInvalidRefreshToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", AuthResponse{}, autherrors.ErrInvalidToken
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return "", "", AuthResponse{}, autherrors.ErrInvalidToken
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return "", "", AuthResponse{}, autherrors.ErrInvalidUserID
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return "", "", AuthResponse{}, autherrors.ErrUserNotFound
	}

	newAccessToken, err := s.generateToken(user.ID.String(), user.Role, time.Minute*15)
	if err != nil {
		return "", "", AuthResponse{}, autherrors.ErrTokenGenerationFailed
	}

	newRefreshToken, err := s.generateToken(user.ID.String(), user.Role, time.Hour*24*7)
	if err != nil {
		return "", "", AuthResponse{}, autherrors.ErrTokenGenerationFailed
	}

	return newAccessToken, newRefreshToken, AuthResponse{
		ID:    user.ID.String(),
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}, nil
}

func (s *Service) GetMe(ctx context.Context, userID string) (*AuthResponse, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, autherrors.ErrInvalidUserID
	}

	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, autherrors.ErrUserNotFound
	}

	return &AuthResponse{
		ID:    u.ID.String(),
		Email: u.Email,
		Name:  u.Name,
		Role:  u.Role,
	}, nil
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (AuthResponse, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResponse{}, autherrors.ErrTokenGenerationFailed
	}

	fullName := strings.TrimSpace(req.FirstName + " " + req.LastName)

	user, err := s.repo.Create(ctx, dbgen.CreateUserParams{
		Email:    req.Email,
		Name:     fullName,
		Password: string(hashed),
		Role:     "CUSTOMER",
	})
	if err != nil {
		return AuthResponse{}, autherrors.ErrEmailAlreadyRegistered
	}

	return AuthResponse{
		ID:    user.ID.String(),
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}, nil
}

func (s *Service) GetCustomerByEmail(ctx context.Context, email string) (map[string]any, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, autherrors.ErrUserNotFound
	}

	return map[string]any{
		"userId": user.ID.String(),
		"role":   user.Role,
		"user": map[string]any{
			"name":  user.Name,
			"email": user.Email,
		},
	}, nil
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) (ActionStatusResponse, error) {
	user, err := s.repo.GetUserProfileByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return ActionStatusResponse{Success: true, EmailSent: false}, nil
		}
		return ActionStatusResponse{}, err
	}

	existingToken, err := s.repo.GetLatestPasswordResetTokenByUserID(ctx, user.ID)
	now := time.Now()
	if err == nil {
		diffMinutes := now.Sub(existingToken.CreatedAt).Minutes()
		if diffMinutes < 10 && existingToken.ExpiresAt.After(now) {
			return ActionStatusResponse{
				Success:   true,
				EmailSent: false,
				Message:   "A password reset link was recently sent. Please check your email or try again later.",
			}, nil
		}
	} else if err != sql.ErrNoRows {
		return ActionStatusResponse{}, err
	}

	resetToken := uuid.NewString()
	expiresAt := now.Add(30 * time.Minute)

	if err := s.repo.UpsertPasswordResetToken(ctx, user.ID, resetToken, expiresAt, now); err != nil {
		return ActionStatusResponse{}, err
	}

	baseURL := os.Getenv("WEBSTORE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", baseURL, resetToken)

	if err := s.emailSvc.SendResetPasswordEmail(ctx, email, user.Name, resetURL); err != nil {
		return ActionStatusResponse{}, err
	}

	return ActionStatusResponse{Success: true, EmailSent: true}, nil
}

func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) (ActionStatusResponse, error) {
	resetRecord, err := s.repo.GetPasswordResetToken(ctx, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return ActionStatusResponse{}, autherrors.ErrResetTokenInvalid
		}
		return ActionStatusResponse{}, err
	}

	if time.Now().After(resetRecord.ExpiresAt) {
		_ = s.repo.DeletePasswordResetTokenByToken(ctx, token)
		return ActionStatusResponse{}, autherrors.ErrResetTokenExpired
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return ActionStatusResponse{}, autherrors.ErrTokenGenerationFailed
	}

	if err := s.repo.UpdateUserPassword(ctx, resetRecord.UserID, string(hashed)); err != nil {
		return ActionStatusResponse{}, err
	}

	if err := s.repo.DeletePasswordResetTokenByToken(ctx, token); err != nil {
		return ActionStatusResponse{}, err
	}

	return ActionStatusResponse{
		Success: true,
		Message: "Password has been reset successfully.",
	}, nil
}

func (s *Service) RequestEmailConfirmation(ctx context.Context, email string, clientType platform.ClientType) (ActionStatusResponse, error) {
	user, err := s.repo.GetUserProfileByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return ActionStatusResponse{Success: true, EmailSent: false}, nil
		}
		return ActionStatusResponse{}, err
	}

	if user.EmailConfirmed {
		return ActionStatusResponse{
			Success:   true,
			EmailSent: false,
			Message:   "Email is already confirmed.",
		}, nil
	}

	now := time.Now()
	existingToken, err := s.repo.GetLatestEmailConfirmationTokenByUserID(ctx, user.ID)
	if err == nil {
		diffMinutes := now.Sub(existingToken.CreatedAt).Minutes()
		if diffMinutes < 10 && existingToken.ExpiresAt.After(now) {
			return ActionStatusResponse{
				Success:   true,
				EmailSent: false,
				Message:   "A confirmation email was recently sent. Please check your inbox or try again later.",
			}, nil
		}
	} else if err != sql.ErrNoRows {
		return ActionStatusResponse{}, err
	}

	token := uuid.NewString()
	pin, err := generatePIN()
	if err != nil {
		return ActionStatusResponse{}, err
	}
	expiresAt := now.Add(60 * time.Minute)

	if err := s.repo.UpsertEmailConfirmationToken(ctx, user.ID, token, pin, expiresAt, now); err != nil {
		return ActionStatusResponse{}, err
	}

	baseURL := os.Getenv("WEBSTORE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	confirmURL := fmt.Sprintf("%s/verify-email?token=%s", baseURL, token)

	if clientType == platform.Mobile {
		if err := s.emailSvc.SendConfirmationPin(ctx, email, user.Name, pin); err != nil {
			return ActionStatusResponse{}, err
		}
	} else {
		if err := s.emailSvc.SendConfirmationLink(ctx, email, user.Name, confirmURL); err != nil {
			return ActionStatusResponse{}, err
		}
	}

	return ActionStatusResponse{Success: true, EmailSent: true}, nil
}

func (s *Service) ResendEmailConfirmation(ctx context.Context, email string, clientType platform.ClientType) (ActionStatusResponse, error) {
	user, err := s.repo.GetUserProfileByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return ActionStatusResponse{Success: true, EmailSent: false}, nil
		}
		return ActionStatusResponse{}, err
	}

	if user.EmailConfirmed {
		return ActionStatusResponse{
			Success:   true,
			EmailSent: false,
			Message:   "Email is already confirmed.",
		}, nil
	}

	now := time.Now()
	existingToken, err := s.repo.GetLatestEmailConfirmationTokenByUserID(ctx, user.ID)
	if err == nil {
		diffMinutes := now.Sub(existingToken.CreatedAt).Minutes()
		if diffMinutes < 10 && existingToken.ExpiresAt.After(now) {
			return ActionStatusResponse{
				Success:   true,
				EmailSent: false,
				Message:   "A confirmation email was recently sent. Please check your inbox or try again later.",
			}, nil
		}
	} else if err != sql.ErrNoRows {
		return ActionStatusResponse{}, err
	}

	if err := s.repo.DeleteEmailConfirmationTokensByUserID(ctx, user.ID); err != nil {
		return ActionStatusResponse{}, err
	}

	token := uuid.NewString()
	pin, err := generatePIN()
	if err != nil {
		return ActionStatusResponse{}, err
	}
	expiresAt := now.Add(60 * time.Minute)

	if err := s.repo.UpsertEmailConfirmationToken(ctx, user.ID, token, pin, expiresAt, now); err != nil {
		return ActionStatusResponse{}, err
	}

	baseURL := os.Getenv("WEBSTORE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	confirmURL := fmt.Sprintf("%s/verify-email?token=%s", baseURL, token)

	if clientType == platform.Mobile {
		if err := s.emailSvc.SendConfirmationPin(ctx, email, user.Name, pin); err != nil {
			return ActionStatusResponse{}, err
		}
	} else {
		if err := s.emailSvc.SendConfirmationLink(ctx, email, user.Name, confirmURL); err != nil {
			return ActionStatusResponse{}, err
		}
	}

	return ActionStatusResponse{Success: true, EmailSent: true}, nil
}

func (s *Service) ConfirmEmailByToken(ctx context.Context, token string) (ActionStatusResponse, error) {
	record, err := s.repo.GetEmailConfirmationTokenByToken(ctx, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return ActionStatusResponse{}, autherrors.ErrConfirmationTokenInvalid
		}
		return ActionStatusResponse{}, err
	}

	if time.Now().After(record.ExpiresAt) {
		_ = s.repo.DeleteEmailConfirmationTokenByToken(ctx, token)
		return ActionStatusResponse{}, autherrors.ErrConfirmationTokenExpired
	}

	if err := s.repo.SetUserEmailConfirmed(ctx, record.UserID); err != nil {
		return ActionStatusResponse{}, err
	}

	if err := s.repo.DeleteEmailConfirmationTokenByToken(ctx, token); err != nil {
		return ActionStatusResponse{}, err
	}

	return ActionStatusResponse{
		Success: true,
		Message: "Email has been successfully confirmed.",
	}, nil
}

func (s *Service) ConfirmEmailByPin(ctx context.Context, email, pin string) (ActionStatusResponse, error) {
	user, err := s.repo.GetUserProfileByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return ActionStatusResponse{}, autherrors.ErrUserNotFound
		}
		return ActionStatusResponse{}, err
	}

	if user.EmailConfirmed {
		return ActionStatusResponse{
			Success: true,
			Message: "Email is already confirmed.",
		}, nil
	}

	record, err := s.repo.GetLatestEmailConfirmationTokenByUserID(ctx, user.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ActionStatusResponse{}, autherrors.ErrConfirmationPinInvalid
		}
		return ActionStatusResponse{}, err
	}

	if record.Pin != pin {
		return ActionStatusResponse{}, autherrors.ErrConfirmationPinInvalid
	}

	if time.Now().After(record.ExpiresAt) {
		_ = s.repo.DeleteEmailConfirmationTokenByPin(ctx, pin)
		return ActionStatusResponse{}, autherrors.ErrConfirmationTokenExpired
	}

	if err := s.repo.SetUserEmailConfirmed(ctx, user.ID); err != nil {
		return ActionStatusResponse{}, err
	}

	if err := s.repo.DeleteEmailConfirmationTokenByPin(ctx, pin); err != nil {
		return ActionStatusResponse{}, err
	}

	return ActionStatusResponse{
		Success: true,
		Message: "Email has been successfully confirmed.",
	}, nil
}

func generatePIN() (string, error) {
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

// reusable token generator
func (s *Service) generateToken(userID, role string, expiry time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(expiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
