package empresas

import (
	"context"
	"fmt"
	"strings"
	"time"

	"omie-sync-api/internal/apperror"
)

const gracePeriod = 30 * 24 * time.Hour

type Service interface {
	Create(ctx context.Context, grupoID string, req CreateRequest) (*EmpresaResponse, error)
	GetByID(ctx context.Context, id string) (*EmpresaResponse, error)
	List(ctx context.Context, params ListParams) ([]*EmpresaResponse, int64, error)
	Update(ctx context.Context, id string, req UpdateRequest) (*EmpresaResponse, error)
	Delete(ctx context.Context, id string) error
	Reativar(ctx context.Context, id string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, grupoID string, req CreateRequest) (*EmpresaResponse, error) {
	if err := validateCreate(req); err != nil {
		return nil, err
	}

	empresa, err := s.repo.Insert(ctx, grupoID,
		strings.TrimSpace(req.Nome),
		strings.TrimSpace(req.CNPJ),
		strings.TrimSpace(req.AppKey),
		strings.TrimSpace(req.AppSecret),
	)
	if err != nil {
		return nil, fmt.Errorf("empresas.service.Create: %w", err)
	}

	return toResponse(empresa), nil
}

func (s *service) GetByID(ctx context.Context, id string) (*EmpresaResponse, error) {
	empresa, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("empresa não encontrada")
	}
	return toResponse(empresa), nil
}

func (s *service) List(ctx context.Context, params ListParams) ([]*EmpresaResponse, int64, error) {
	if params.PerPage <= 0 {
		params.PerPage = 50
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	offset := int32((params.Page - 1) * params.PerPage)
	limit := int32(params.PerPage)

	empresas, err := s.repo.List(ctx, params.GrupoID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("empresas.service.List: %w", err)
	}

	total, err := s.repo.Count(ctx, params.GrupoID)
	if err != nil {
		return nil, 0, fmt.Errorf("empresas.service.List count: %w", err)
	}

	result := make([]*EmpresaResponse, len(empresas))
	for i, e := range empresas {
		result[i] = toResponse(e)
	}
	return result, total, nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateRequest) (*EmpresaResponse, error) {
	if strings.TrimSpace(req.Nome) == "" {
		return nil, apperror.Unprocessable("nome é obrigatório")
	}
	if strings.TrimSpace(req.AppKey) == "" || strings.TrimSpace(req.AppSecret) == "" {
		return nil, apperror.Unprocessable("app_key e app_secret são obrigatórios")
	}

	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, apperror.NotFound("empresa não encontrada")
	}

	empresa, err := s.repo.Update(ctx, id,
		strings.TrimSpace(req.Nome),
		strings.TrimSpace(req.CNPJ),
		strings.TrimSpace(req.AppKey),
		strings.TrimSpace(req.AppSecret),
	)
	if err != nil {
		return nil, fmt.Errorf("empresas.service.Update: %w", err)
	}

	return toResponse(empresa), nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	empresa, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperror.NotFound("empresa não encontrada")
	}

	if empresa.Status == "deletando" {
		return apperror.Conflict("empresa já está em processo de exclusão")
	}

	// 1. Marca deleted_at + status=deletando
	if err := s.repo.MarkDeletando(ctx, id); err != nil {
		return fmt.Errorf("empresas.service.Delete marcar deletando: %w", err)
	}

	// 2. Insere na fila com carência de 30 dias
	executeAt := time.Now().Add(gracePeriod)
	if err := s.repo.InsertDeletionQueue(ctx, id, executeAt); err != nil {
		return fmt.Errorf("empresas.service.Delete insertion_queue: %w", err)
	}

	return nil
}

func (s *service) Reativar(ctx context.Context, id string) error {
	empresa, err := s.repo.GetByID(ctx, id)
	if err != nil {
		// GetByID não retorna empresa com deleted_at — usa scan direto
		return apperror.NotFound("empresa não encontrada")
	}

	if empresa.Status == "ativa" {
		return apperror.Conflict("empresa já está ativa")
	}

	// Cancela a fila de exclusão pendente
	if err := s.repo.CancelDeletionQueue(ctx, id); err != nil {
		return fmt.Errorf("empresas.service.Reativar cancelar fila: %w", err)
	}

	if err := s.repo.Reativar(ctx, id); err != nil {
		return fmt.Errorf("empresas.service.Reativar: %w", err)
	}

	return nil
}

func validateCreate(req CreateRequest) error {
	if strings.TrimSpace(req.Nome) == "" {
		return apperror.Unprocessable("nome é obrigatório")
	}
	if strings.TrimSpace(req.AppKey) == "" {
		return apperror.Unprocessable("app_key é obrigatório")
	}
	if strings.TrimSpace(req.AppSecret) == "" {
		return apperror.Unprocessable("app_secret é obrigatório")
	}
	return nil
}
