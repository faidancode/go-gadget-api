package auth

import (
	"context"
	"os"
	"time"

	autherrors "go-gadget-api/internal/auth/errors"
	"go-gadget-api/internal/dbgen"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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

	user, err := s.repo.Create(ctx, dbgen.CreateUserParams{
		Email:    req.Email,
		Name:     req.Name,
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
