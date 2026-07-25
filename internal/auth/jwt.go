package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	accessTokenDuration  = 15 * time.Minute
	preAuthTokenDuration = 5 * time.Minute
)

type JWTClaims struct {
	UserID  string `json:"user_id"`
	GrupoID string `json:"grupo_id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	jwt.RegisteredClaims
}

// PreAuthClaims é emitido no login quando o usuário tem múltiplos grupos.
// Não concede acesso a nenhum recurso — só é aceito em /auth/select-grupo.
type PreAuthClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Type   string `json:"type"` // sempre "pre_auth"
	jwt.RegisteredClaims
}

type JWTService interface {
	Generate(userID, grupoID, email, role string) (string, error)
	Validate(tokenStr string) (*JWTClaims, error)
	GeneratePreAuth(userID, email string) (string, error)
	ValidatePreAuth(tokenStr string) (*PreAuthClaims, error)
}

type jwtService struct {
	secret []byte
}

func NewJWTService(secret string) JWTService {
	return &jwtService{secret: []byte(secret)}
}

func (s *jwtService) Generate(userID, grupoID, email, role string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:  userID,
		GrupoID: grupoID,
		Email:   email,
		Role:    role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenDuration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("auth.jwt.Generate: %w", err)
	}
	return signed, nil
}

func (s *jwtService) Validate(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth.jwt.Validate: algoritmo inesperado %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth.jwt.Validate: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("auth.jwt.Validate: token inválido")
	}
	return claims, nil
}

func (s *jwtService) GeneratePreAuth(userID, email string) (string, error) {
	now := time.Now()
	claims := PreAuthClaims{
		UserID: userID,
		Email:  email,
		Type:   "pre_auth",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(preAuthTokenDuration)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("auth.jwt.GeneratePreAuth: %w", err)
	}
	return signed, nil
}

func (s *jwtService) ValidatePreAuth(tokenStr string) (*PreAuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &PreAuthClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth.jwt.ValidatePreAuth: algoritmo inesperado %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth.jwt.ValidatePreAuth: %w", err)
	}

	claims, ok := token.Claims.(*PreAuthClaims)
	if !ok || !token.Valid || claims.Type != "pre_auth" {
		return nil, fmt.Errorf("auth.jwt.ValidatePreAuth: token inválido")
	}
	return claims, nil
}
