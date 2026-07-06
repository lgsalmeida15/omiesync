package usuarios

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/sqlc/generated"
)

type Repository interface {
	Insert(ctx context.Context, grupoID, nome, email, passwordHash, role string) (*Usuario, error)
	GetByID(ctx context.Context, id string) (*Usuario, error)
	List(ctx context.Context, grupoID string, limit, offset int32) ([]*Usuario, error)
	Count(ctx context.Context, grupoID string) (int64, error)
	Update(ctx context.Context, id, nome, role string, ativo bool) (*Usuario, error)
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	SoftDelete(ctx context.Context, id string) error
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) Insert(ctx context.Context, grupoID, nome, email, passwordHash, role string) (*Usuario, error) {
	q := sqlcgen.New(r.pool)
	var gid pgtype.UUID
	if err := gid.Scan(grupoID); err != nil {
		return nil, fmt.Errorf("usuarios.repository.Insert scan grupo_id: %w", err)
	}
	row, err := q.InsertUsuario(ctx, sqlcgen.InsertUsuarioParams{
		GrupoID:  gid,
		Nome:     nome,
		Email:    email,
		Password: passwordHash,
		Role:     role,
	})
	if err != nil {
		return nil, fmt.Errorf("usuarios.repository.Insert: %w", err)
	}
	return toUsuario(row.ID, row.GrupoID, row.Nome, row.Email, row.Role, row.Ativo, row.CreatedAt, row.UpdatedAt), nil
}

func (r *repository) GetByID(ctx context.Context, id string) (*Usuario, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return nil, fmt.Errorf("usuarios.repository.GetByID scan uuid: %w", err)
	}
	row, err := q.GetUsuarioByIDFull(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("usuarios.repository.GetByID: %w", err)
	}
	return toUsuario(row.ID, row.GrupoID, row.Nome, row.Email, row.Role, row.Ativo, row.CreatedAt, row.UpdatedAt), nil
}

func (r *repository) List(ctx context.Context, grupoID string, limit, offset int32) ([]*Usuario, error) {
	q := sqlcgen.New(r.pool)
	var gid pgtype.UUID
	if err := gid.Scan(grupoID); err != nil {
		return nil, fmt.Errorf("usuarios.repository.List scan grupo_id: %w", err)
	}
	rows, err := q.ListUsuariosByGrupo(ctx, sqlcgen.ListUsuariosByGrupoParams{GrupoID: gid, Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("usuarios.repository.List: %w", err)
	}
	result := make([]*Usuario, len(rows))
	for i, row := range rows {
		result[i] = toUsuario(row.ID, row.GrupoID, row.Nome, row.Email, row.Role, row.Ativo, row.CreatedAt, row.UpdatedAt)
	}
	return result, nil
}

func (r *repository) Count(ctx context.Context, grupoID string) (int64, error) {
	q := sqlcgen.New(r.pool)
	var gid pgtype.UUID
	if err := gid.Scan(grupoID); err != nil {
		return 0, fmt.Errorf("usuarios.repository.Count scan grupo_id: %w", err)
	}
	n, err := q.CountUsuariosByGrupo(ctx, gid)
	if err != nil {
		return 0, fmt.Errorf("usuarios.repository.Count: %w", err)
	}
	return n, nil
}

func (r *repository) Update(ctx context.Context, id, nome, role string, ativo bool) (*Usuario, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return nil, fmt.Errorf("usuarios.repository.Update scan uuid: %w", err)
	}
	row, err := q.UpdateUsuario(ctx, sqlcgen.UpdateUsuarioParams{ID: uid, Nome: nome, Role: role, Ativo: ativo})
	if err != nil {
		return nil, fmt.Errorf("usuarios.repository.Update: %w", err)
	}
	return toUsuario(row.ID, row.GrupoID, row.Nome, row.Email, row.Role, row.Ativo, row.CreatedAt, row.UpdatedAt), nil
}

func (r *repository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return fmt.Errorf("usuarios.repository.UpdatePassword scan uuid: %w", err)
	}
	if err := q.UpdateUsuarioPassword(ctx, sqlcgen.UpdateUsuarioPasswordParams{ID: uid, Password: passwordHash}); err != nil {
		return fmt.Errorf("usuarios.repository.UpdatePassword: %w", err)
	}
	return nil
}

func (r *repository) SoftDelete(ctx context.Context, id string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return fmt.Errorf("usuarios.repository.SoftDelete scan uuid: %w", err)
	}
	if err := q.SoftDeleteUsuario(ctx, uid); err != nil {
		return fmt.Errorf("usuarios.repository.SoftDelete: %w", err)
	}
	return nil
}

func toUsuario(id, grupoID pgtype.UUID, nome, email, role string, ativo bool, createdAt, updatedAt pgtype.Timestamptz) *Usuario {
	return &Usuario{
		ID:        uuidToStr(id),
		GrupoID:   uuidToStr(grupoID),
		Nome:      nome,
		Email:     email,
		Role:      role,
		Ativo:     ativo,
		CreatedAt: createdAt.Time,
		UpdatedAt: updatedAt.Time,
	}
}

func uuidToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}
