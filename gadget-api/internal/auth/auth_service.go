package auth

import (
	"context"
	"database/sql"
	"fmt"
	"gadget-api/internal/dbgen"
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

func (s *Service) Login(ctx context.Context, email, password string) (string, AuthResponse, error) {
	// 1. Cari user di database
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", AuthResponse{}, fmt.Errorf("invalid email or password")
	}

	// 2. Verifikasi Password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", AuthResponse{}, fmt.Errorf("invalid email or password")
	}

	// 3. Generate JWT Token
	tokenString, err := s.generateToken(user.ID.String(), user.Role.String)
	if err != nil {
		return "", AuthResponse{}, fmt.Errorf("failed to generate token")
	}

	return tokenString, AuthResponse{
		Email: user.Email,
		Role:  user.Role.String,
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

func (s *Service) Register(ctx context.Context, req RegisterRequest) (AuthResponse, error) {
	// 1. Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("failed to hash password")
	}

	// 2. Simpan user
	user, err := s.repo.Create(ctx, dbgen.CreateUserParams{
		Email:    req.Email,
		Name:     req.Name,
		Password: string(hashed),
		Role: sql.NullString{
			String: "customer",
			Valid:  true,
		},
	})
	if err != nil {
		return AuthResponse{}, fmt.Errorf("email already registered")
	}

	return AuthResponse{
		Email: user.Email,
		Role:  user.Role.String,
	}, nil
}
