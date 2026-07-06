package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"omie-sync-api/internal/apperror"
)

const refreshTokenDuration = 7 * 24 * time.Hour

type Service interface {
	Login(ctx context.Context, email, password string) (*LoginResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	Refresh(ctx context.Context, refreshToken string) (*LoginResponse, error)
	Me(ctx context.Context, userID string) (*MeResponse, error)
}

type service struct {
	repo Repository
	jwt  JWTService
}

func NewService(repo Repository, jwt JWTService) Service {
	return &service{repo: repo, jwt: jwt}
}

func (s *service) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	usuario, err := s.repo.GetUsuarioByEmail(ctx, email)
	if err != nil {
		return nil, apperror.Unauthorized("credenciais inválidas")
	}

	if !usuario.Ativo {
		return nil, apperror.Unauthorized("usuário inativo")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usuario.Password), []byte(password)); err != nil {
		return nil, apperror.Unauthorized("credenciais inválidas")
	}

	accessToken, err := s.jwt.Generate(usuario.ID, usuario.GrupoID, usuario.Email, usuario.Role)
	if err != nil {
		return nil, fmt.Errorf("auth.service.Login gerar access token: %w", err)
	}

	refreshToken, err := generateOpaqueToken()
	if err != nil {
		return nil, fmt.Errorf("auth.service.Login gerar refresh token: %w", err)
	}

	if _, err := s.repo.InsertRefreshToken(ctx, usuario.ID, refreshToken, time.Now().Add(refreshTokenDuration)); err != nil {
		return nil, fmt.Errorf("auth.service.Login salvar refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(15 * time.Minute / time.Second),
	}, nil
}

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	if err := s.repo.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return fmt.Errorf("auth.service.Logout: %w", err)
	}
	return nil
}

func (s *service) Refresh(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	rt, err := s.repo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, apperror.Unauthorized("refresh token inválido ou expirado")
	}

	// Rotação obrigatória: revoga o token atual
	if err := s.repo.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("auth.service.Refresh revogar token: %w", err)
	}

	usuario, err := s.repo.GetUsuarioByID(ctx, rt.UsuarioID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.Refresh buscar usuário: %w", err)
	}

	if !usuario.Ativo {
		return nil, apperror.Unauthorized("usuário inativo")
	}

	accessToken, err := s.jwt.Generate(usuario.ID, usuario.GrupoID, usuario.Email, usuario.Role)
	if err != nil {
		return nil, fmt.Errorf("auth.service.Refresh gerar access token: %w", err)
	}

	newRefreshToken, err := generateOpaqueToken()
	if err != nil {
		return nil, fmt.Errorf("auth.service.Refresh gerar novo refresh token: %w", err)
	}

	if _, err := s.repo.InsertRefreshToken(ctx, usuario.ID, newRefreshToken, time.Now().Add(refreshTokenDuration)); err != nil {
		return nil, fmt.Errorf("auth.service.Refresh salvar novo refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(15 * time.Minute / time.Second),
	}, nil
}

func (s *service) Me(ctx context.Context, userID string) (*MeResponse, error) {
	usuario, err := s.repo.GetUsuarioByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.Me: %w", err)
	}
	return &MeResponse{
		ID:      usuario.ID,
		GrupoID: usuario.GrupoID,
		Nome:    usuario.Nome,
		Email:   usuario.Email,
		Role:    usuario.Role,
	}, nil
}

func generateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateOpaqueToken: %w", err)
	}
	return hex.EncodeToString(b), nil
}
