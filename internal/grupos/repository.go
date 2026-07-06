package grupos

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/sqlc/generated"
)

type Repository interface {
	Insert(ctx context.Context, nome, slug, schemaName string) (*Grupo, error)
	GetByID(ctx context.Context, id string) (*Grupo, error)
	GetBySlug(ctx context.Context, slug string) (*Grupo, error)
	List(ctx context.Context, limit, offset int32) ([]*Grupo, error)
	Count(ctx context.Context) (int64, error)
	Update(ctx context.Context, id, nome string) (*Grupo, error)
	SoftDelete(ctx context.Context, id string) error
	CountEmpresasAtivas(ctx context.Context, grupoID string) (int64, error)
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) Insert(ctx context.Context, nome, slug, schemaName string) (*Grupo, error) {
	q := sqlcgen.New(r.pool)
	row, err := q.InsertGrupo(ctx, sqlcgen.InsertGrupoParams{
		Nome:       nome,
		Slug:       slug,
		SchemaName: schemaName,
	})
	if err != nil {
		return nil, fmt.Errorf("grupos.repository.Insert: %w", err)
	}
	return toGrupo(row), nil
}

func (r *repository) GetByID(ctx context.Context, id string) (*Grupo, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return nil, fmt.Errorf("grupos.repository.GetByID scan uuid: %w", err)
	}
	row, err := q.GetGrupoByID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("grupos.repository.GetByID: %w", err)
	}
	return toGrupo(row), nil
}

func (r *repository) GetBySlug(ctx context.Context, slug string) (*Grupo, error) {
	q := sqlcgen.New(r.pool)
	row, err := q.GetGrupoBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("grupos.repository.GetBySlug: %w", err)
	}
	return toGrupo(row), nil
}

func (r *repository) List(ctx context.Context, limit, offset int32) ([]*Grupo, error) {
	q := sqlcgen.New(r.pool)
	rows, err := q.ListGrupos(ctx, sqlcgen.ListGruposParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("grupos.repository.List: %w", err)
	}
	result := make([]*Grupo, len(rows))
	for i, row := range rows {
		result[i] = toGrupo(row)
	}
	return result, nil
}

func (r *repository) Count(ctx context.Context) (int64, error) {
	q := sqlcgen.New(r.pool)
	n, err := q.CountGrupos(ctx)
	if err != nil {
		return 0, fmt.Errorf("grupos.repository.Count: %w", err)
	}
	return n, nil
}

func (r *repository) Update(ctx context.Context, id, nome string) (*Grupo, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return nil, fmt.Errorf("grupos.repository.Update scan uuid: %w", err)
	}
	row, err := q.UpdateGrupo(ctx, sqlcgen.UpdateGrupoParams{ID: uid, Nome: nome})
	if err != nil {
		return nil, fmt.Errorf("grupos.repository.Update: %w", err)
	}
	return toGrupo(row), nil
}

func (r *repository) SoftDelete(ctx context.Context, id string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return fmt.Errorf("grupos.repository.SoftDelete scan uuid: %w", err)
	}
	if err := q.SoftDeleteGrupo(ctx, uid); err != nil {
		return fmt.Errorf("grupos.repository.SoftDelete: %w", err)
	}
	return nil
}

func (r *repository) CountEmpresasAtivas(ctx context.Context, grupoID string) (int64, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(grupoID); err != nil {
		return 0, fmt.Errorf("grupos.repository.CountEmpresasAtivas scan uuid: %w", err)
	}
	n, err := q.CountEmpresasAtivasByGrupo(ctx, uid)
	if err != nil {
		return 0, fmt.Errorf("grupos.repository.CountEmpresasAtivas: %w", err)
	}
	return n, nil
}

func toGrupo(row sqlcgen.EtlGrupo) *Grupo {
	return &Grupo{
		ID:         uuidToStr(row.ID),
		Nome:       row.Nome,
		Slug:       row.Slug,
		SchemaName: row.SchemaName,
		Status:     row.Status,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}

func uuidToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}
