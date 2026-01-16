## DTO
package auth

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

## Repo
package auth

import (
	"context"
	"gadget-api/internal/dbgen"
)

//go:generate mockgen -source=auth_repo.go -destination=mock/auth_repo_mock.go -package=mock
type Repository interface {
	GetByEmail(ctx context.Context, email string) (dbgen.GetUserByEmailRow, error)
}

type repository struct {
	queries *dbgen.Queries
}

func NewRepository(q *dbgen.Queries) Repository {
	return &repository{queries: q}
}

func (r *repository) GetByEmail(ctx context.Context, email string) (dbgen.GetUserByEmailRow, error) {
	return r.queries.GetUserByEmail(ctx, email)
}

## Service
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


## Controller
package auth

import (
	"gadget-api/internal/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service *Service
}

func NewController(s *Service) *Controller {
	return &Controller{service: s}
}

func (ctrl *Controller) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Response Error Seragam
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", err.Error())
		return
	}

	token, userResp, err := ctrl.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		// Response Error Seragam
		response.Error(c, http.StatusUnauthorized, "AUTH_FAILED", "Email atau password salah", nil)
		return
	}

	// Set Cookie
	c.SetCookie(
		"access_token",
		token,
		86400,
		"/",
		"",
		false,
		true,
	)

	// Response Success Seragam
	// Data yang dikirim adalah struct AuthResponse (Email & Role)
	response.Success(c, http.StatusOK, userResp, nil)
}

func (ctrl *Controller) Logout(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", false, true)

	// Response Success Seragam
	response.Success(c, http.StatusOK, "Logout berhasil", nil)
}

