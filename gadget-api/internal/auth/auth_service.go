package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Login(ctx context.Context, username, password string) (string, AuthResponse, error) {
	// 1. Cari user di database
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return "", AuthResponse{}, fmt.Errorf("invalid username or password")
	}

	// 2. Verifikasi Password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", AuthResponse{}, fmt.Errorf("invalid username or password")
	}

	// 3. Generate JWT Token
	tokenString, err := s.generateToken(user.ID.String(), user.Role.String)
	if err != nil {
		return "", AuthResponse{}, fmt.Errorf("failed to generate token")
	}

	return tokenString, AuthResponse{
		Username: user.Username,
		Role:     user.Role.String,
	}, nil
}

func (s *Service) generateToken(userID, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
