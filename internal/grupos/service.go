package grupos

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/db"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:[_-][a-z0-9]+)*$`)

type Service interface {
	Create(ctx context.Context, req CreateRequest) (*Grupo, error)
	GetByID(ctx context.Context, id string) (*Grupo, error)
	List(ctx context.Context, params ListParams) ([]*Grupo, int64, error)
	Update(ctx context.Context, id string, req UpdateRequest) (*Grupo, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo        Repository
	provisioner *db.Provisioner
}

func NewService(repo Repository, provisioner *db.Provisioner) Service {
	return &service{repo: repo, provisioner: provisioner}
}

func (s *service) Create(ctx context.Context, req CreateRequest) (*Grupo, error) {
	if err := validateCreate(req); err != nil {
		return nil, err
	}

	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	schemaName := db.SchemaName(slug)

	// Verifica duplicidade de slug
	if existing, _ := s.repo.GetBySlug(ctx, slug); existing != nil {
		return nil, apperror.Conflict("já existe um grupo com este slug")
	}

	grupo, err := s.repo.Insert(ctx, strings.TrimSpace(req.Nome), slug, schemaName)
	if err != nil {
		return nil, fmt.Errorf("grupos.service.Create: %w", err)
	}

	// Provisiona schema PostgreSQL com todas as tabelas Omie
	if s.provisioner != nil {
		if err := s.provisioner.ProvisionSchema(ctx, schemaName); err != nil {
			return nil, fmt.Errorf("grupos.service.Create provisionar schema: %w", err)
		}
	}

	return grupo, nil
}

func (s *service) GetByID(ctx context.Context, id string) (*Grupo, error) {
	grupo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("grupo não encontrado")
	}
	return grupo, nil
}

func (s *service) List(ctx context.Context, params ListParams) ([]*Grupo, int64, error) {
	if params.PerPage <= 0 {
		params.PerPage = 50
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	offset := int32((params.Page - 1) * params.PerPage)
	limit := int32(params.PerPage)

	grupos, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("grupos.service.List: %w", err)
	}

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("grupos.service.List count: %w", err)
	}

	return grupos, total, nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateRequest) (*Grupo, error) {
	if strings.TrimSpace(req.Nome) == "" {
		return nil, apperror.Unprocessable("nome é obrigatório")
	}

	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, apperror.NotFound("grupo não encontrado")
	}

	grupo, err := s.repo.Update(ctx, id, strings.TrimSpace(req.Nome))
	if err != nil {
		return nil, fmt.Errorf("grupos.service.Update: %w", err)
	}

	return grupo, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperror.NotFound("grupo não encontrado")
	}

	// Regra absoluta: grupo só pode ser deletado se todas as empresas estiverem inativas
	count, err := s.repo.CountEmpresasAtivas(ctx, id)
	if err != nil {
		return fmt.Errorf("grupos.service.Delete verificar empresas: %w", err)
	}
	if count > 0 {
		return apperror.Conflict("grupo possui empresas ativas — desative todas antes de excluir")
	}

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("grupos.service.Delete: %w", err)
	}

	return nil
}

func validateCreate(req CreateRequest) error {
	if strings.TrimSpace(req.Nome) == "" {
		return apperror.Unprocessable("nome é obrigatório")
	}
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if slug == "" {
		return apperror.Unprocessable("slug é obrigatório")
	}
	if !slugRegex.MatchString(slug) {
		return apperror.Unprocessable("slug deve conter apenas letras minúsculas, números, hífens e underscores")
	}
	return nil
}
