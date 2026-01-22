package auth

import (
	"context"
	"fmt"
	"gadget-api/internal/dbgen"
	"os"
	"time"

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
	// 1. Cari user di database
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", AuthResponse{}, fmt.Errorf("invalid email or password")
	}

	// 2. Verifikasi Password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", "", AuthResponse{}, fmt.Errorf("invalid email or password")
	}

	// 3. Generate Access Token (misal: 15 menit)
	accessToken, err := s.generateToken(user.ID.String(), user.Role, time.Minute*15)
	if err != nil {
		return "", "", AuthResponse{}, fmt.Errorf("failed to generate access token")
	}

	// 4. Generate Refresh Token (misal: 7 hari)
	refreshToken, err := s.generateToken(user.ID.String(), user.Role, time.Hour*24*7)
	if err != nil {
		return "", "", AuthResponse{}, fmt.Errorf("failed to generate refresh token")
	}

	return accessToken, refreshToken, AuthResponse{
		ID:    user.ID.String(),
		Email: user.Email,
		Role:  user.Role,
	}, nil
}

// Menambahkan parameter expiry agar reusable
func (s *Service) generateToken(userID, role string, expiry time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(expiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, string, AuthResponse, error) {
	// 1. Parse dan Validasi Refresh Token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return "", "", AuthResponse{}, ErrInvalidToken
	}

	// 2. Ambil Claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", AuthResponse{}, ErrInvalidToken
	}

	userIDStr, _ := claims["user_id"].(string)

	// 3. Cari User di Database
	// Ini penting agar kita mendapatkan data terbaru (Email, Name, dll)
	// dan memastikan akun belum di-ban/dihapus.
	user, err := s.repo.GetByID(ctx, uuid.MustParse(userIDStr))
	if err != nil {
		return "", "", AuthResponse{}, ErrUserNotFound
	}

	// 4. Generate Pasangan Token Baru (Rotation)
	newAccessToken, err := s.generateToken(user.ID.String(), user.Role, time.Minute*15)
	if err != nil {
		return "", "", AuthResponse{}, ErrInvalidRefreshToken
	}

	newRefreshToken, err := s.generateToken(user.ID.String(), user.Role, time.Hour*24*7)
	if err != nil {
		return "", "", AuthResponse{}, ErrInvalidToken
	}

	// 5. Kembalikan data lengkap (Tokens + User Info)
	return newAccessToken, newRefreshToken, AuthResponse{
		ID:    user.ID.String(),
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}, nil
}

func (s *Service) GetMe(ctx context.Context, userID string) (*AuthResponse, error) {
	u, err := s.repo.GetByID(ctx, uuid.MustParse(userID))
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		ID:    u.ID.String(),
		Email: u.Email,
		Name:  u.Name,
		Role:  u.Role,
	}, nil
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
		Role:     "CUSTOMER",
	})
	if err != nil {
		return AuthResponse{}, fmt.Errorf("email already registered")
	}

	return AuthResponse{
		Email: user.Email,
		Role:  user.Role,
	}, nil
}
