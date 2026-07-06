package permissoes

import (
	"context"
	"fmt"

	"omie-sync-api/internal/apperror"
)

type Service interface {
	Grant(ctx context.Context, req GrantRequest) (*Permissao, error)
	Revoke(ctx context.Context, req RevokeRequest) error
	ListByUsuario(ctx context.Context, usuarioID string) ([]*Permissao, error)
	ListByEmpresa(ctx context.Context, empresaID string) ([]*Permissao, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Grant(ctx context.Context, req GrantRequest) (*Permissao, error) {
	if err := validate(req.Recurso, req.Acao); err != nil {
		return nil, err
	}
	p, err := s.repo.Grant(ctx, req.UsuarioID, req.EmpresaID, req.Recurso, req.Acao)
	if err != nil {
		return nil, fmt.Errorf("permissoes.service.Grant: %w", err)
	}
	return p, nil
}

func (s *service) Revoke(ctx context.Context, req RevokeRequest) error {
	if err := validate(req.Recurso, req.Acao); err != nil {
		return err
	}
	if err := s.repo.Revoke(ctx, req.UsuarioID, req.EmpresaID, req.Recurso, req.Acao); err != nil {
		return fmt.Errorf("permissoes.service.Revoke: %w", err)
	}
	return nil
}

func (s *service) ListByUsuario(ctx context.Context, usuarioID string) ([]*Permissao, error) {
	ps, err := s.repo.ListByUsuario(ctx, usuarioID)
	if err != nil {
		return nil, fmt.Errorf("permissoes.service.ListByUsuario: %w", err)
	}
	return ps, nil
}

func (s *service) ListByEmpresa(ctx context.Context, empresaID string) ([]*Permissao, error) {
	ps, err := s.repo.ListByEmpresa(ctx, empresaID)
	if err != nil {
		return nil, fmt.Errorf("permissoes.service.ListByEmpresa: %w", err)
	}
	return ps, nil
}

func validate(recurso, acao string) error {
	if !recursosValidos[recurso] {
		return apperror.Unprocessable("recurso inválido: use dashboard, sync ou admin")
	}
	if !acoesValidas[acao] {
		return apperror.Unprocessable("ação inválida: use ver, editar ou forcar_sync")
	}
	return nil
}
