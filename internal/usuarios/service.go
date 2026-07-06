package usuarios

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"omie-sync-api/internal/apperror"
)

type Service interface {
	Create(ctx context.Context, grupoID string, req CreateRequest) (*Usuario, error)
	GetByID(ctx context.Context, id string) (*Usuario, error)
	List(ctx context.Context, params ListParams) ([]*Usuario, int64, error)
	Update(ctx context.Context, id string, req UpdateRequest) (*Usuario, error)
	UpdatePassword(ctx context.Context, id string, req UpdatePasswordRequest) error
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, grupoID string, req CreateRequest) (*Usuario, error) {
	if err := validateCreate(req); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("usuarios.service.Create hash password: %w", err)
	}

	role := req.Role
	if role == "" {
		role = "viewer"
	}

	u, err := s.repo.Insert(ctx, grupoID,
		strings.TrimSpace(req.Nome),
		strings.ToLower(strings.TrimSpace(req.Email)),
		string(hash),
		role,
	)
	if err != nil {
		return nil, fmt.Errorf("usuarios.service.Create: %w", err)
	}
	return u, nil
}

func (s *service) GetByID(ctx context.Context, id string) (*Usuario, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("usuário não encontrado")
	}
	return u, nil
}

func (s *service) List(ctx context.Context, params ListParams) ([]*Usuario, int64, error) {
	if params.PerPage <= 0 {
		params.PerPage = 50
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	offset := int32((params.Page - 1) * params.PerPage)

	us, err := s.repo.List(ctx, params.GrupoID, int32(params.PerPage), offset)
	if err != nil {
		return nil, 0, fmt.Errorf("usuarios.service.List: %w", err)
	}
	total, err := s.repo.Count(ctx, params.GrupoID)
	if err != nil {
		return nil, 0, fmt.Errorf("usuarios.service.List count: %w", err)
	}
	return us, total, nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateRequest) (*Usuario, error) {
	if strings.TrimSpace(req.Nome) == "" {
		return nil, apperror.Unprocessable("nome é obrigatório")
	}
	if req.Role != "" && !rolesValidas[req.Role] {
		return nil, apperror.Unprocessable("role inválida")
	}

	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, apperror.NotFound("usuário não encontrado")
	}

	role := req.Role
	if role == "" {
		role = "viewer"
	}

	u, err := s.repo.Update(ctx, id, strings.TrimSpace(req.Nome), role, req.Ativo)
	if err != nil {
		return nil, fmt.Errorf("usuarios.service.Update: %w", err)
	}
	return u, nil
}

func (s *service) UpdatePassword(ctx context.Context, id string, req UpdatePasswordRequest) error {
	if len(req.Password) < 8 {
		return apperror.Unprocessable("password deve ter no mínimo 8 caracteres")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("usuarios.service.UpdatePassword hash: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, id, string(hash)); err != nil {
		return fmt.Errorf("usuarios.service.UpdatePassword: %w", err)
	}
	return nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperror.NotFound("usuário não encontrado")
	}
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("usuarios.service.Delete: %w", err)
	}
	return nil
}

func validateCreate(req CreateRequest) error {
	if strings.TrimSpace(req.Nome) == "" {
		return apperror.Unprocessable("nome é obrigatório")
	}
	if strings.TrimSpace(req.Email) == "" {
		return apperror.Unprocessable("email é obrigatório")
	}
	if len(req.Password) < 8 {
		return apperror.Unprocessable("password deve ter no mínimo 8 caracteres")
	}
	if req.Role != "" && !rolesValidas[req.Role] {
		return apperror.Unprocessable("role inválida")
	}
	return nil
}
