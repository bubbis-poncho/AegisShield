package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"aegisshield/services/api-gateway/internal/config"
)

type Service struct {
	config config.AuthConfig
}

type Claims struct {
	UserID   string   `json:"user_id"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

type User struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

func NewService(cfg config.AuthConfig) *Service {
	return &Service{
		config: cfg,
	}
}

func (s *Service) GenerateToken(user *User) (string, error) {
	now := time.Now()
	expirationTime := now.Add(time.Duration(s.config.TokenDuration) * time.Minute)

	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Roles:  user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.config.Issuer,
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (s *Service) GetUserFromContext(ctx context.Context) (*User, error) {
	user, ok := ctx.Value("user").(*User)
	if !ok {
		return nil, fmt.Errorf("user not found in context")
	}
	return user, nil
}

func (s *Service) HasRole(user *User, role string) bool {
	for _, userRole := range user.Roles {
		if userRole == role {
			return true
		}
	}
	return false
}

func (s *Service) HasAnyRole(user *User, roles []string) bool {
	for _, role := range roles {
		if s.HasRole(user, role) {
			return true
		}
	}
	return false
}

// Predefined roles
const (
	RoleAnalyst      = "analyst"
	RoleInvestigator = "investigator"
	RoleAdmin        = "admin"
	RoleCompliance   = "compliance"
	RoleViewOnly     = "view_only"
)