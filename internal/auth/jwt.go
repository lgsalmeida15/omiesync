package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const accessTokenDuration = 15 * time.Minute

type JWTClaims struct {
	UserID  string `json:"user_id"`
	GrupoID string `json:"grupo_id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	jwt.RegisteredClaims
}

type JWTService interface {
	Generate(userID, grupoID, email, role string) (string, error)
	Validate(tokenStr string) (*JWTClaims, error)
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
