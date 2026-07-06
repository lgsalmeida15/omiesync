package permissoes

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/sqlc/generated"
)

type Repository interface {
	Grant(ctx context.Context, usuarioID, empresaID, recurso, acao string) (*Permissao, error)
	Revoke(ctx context.Context, usuarioID, empresaID, recurso, acao string) error
	ListByUsuario(ctx context.Context, usuarioID string) ([]*Permissao, error)
	ListByEmpresa(ctx context.Context, empresaID string) ([]*Permissao, error)
	Has(ctx context.Context, usuarioID, empresaID, recurso, acao string) (bool, error)
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) Grant(ctx context.Context, usuarioID, empresaID, recurso, acao string) (*Permissao, error) {
	q := sqlcgen.New(r.pool)
	var uid, eid pgtype.UUID
	if err := uid.Scan(usuarioID); err != nil {
		return nil, fmt.Errorf("permissoes.repository.Grant scan usuario_id: %w", err)
	}
	if err := eid.Scan(empresaID); err != nil {
		return nil, fmt.Errorf("permissoes.repository.Grant scan empresa_id: %w", err)
	}
	row, err := q.InsertPermissao(ctx, sqlcgen.InsertPermissaoParams{
		UsuarioID: uid, EmpresaID: eid, Recurso: recurso, Acao: acao,
	})
	if err != nil {
		return nil, fmt.Errorf("permissoes.repository.Grant: %w", err)
	}
	return toPermissao(row), nil
}

func (r *repository) Revoke(ctx context.Context, usuarioID, empresaID, recurso, acao string) error {
	q := sqlcgen.New(r.pool)
	var uid, eid pgtype.UUID
	if err := uid.Scan(usuarioID); err != nil {
		return fmt.Errorf("permissoes.repository.Revoke scan usuario_id: %w", err)
	}
	if err := eid.Scan(empresaID); err != nil {
		return fmt.Errorf("permissoes.repository.Revoke scan empresa_id: %w", err)
	}
	if err := q.DeletePermissao(ctx, sqlcgen.DeletePermissaoParams{
		UsuarioID: uid, EmpresaID: eid, Recurso: recurso, Acao: acao,
	}); err != nil {
		return fmt.Errorf("permissoes.repository.Revoke: %w", err)
	}
	return nil
}

func (r *repository) ListByUsuario(ctx context.Context, usuarioID string) ([]*Permissao, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(usuarioID); err != nil {
		return nil, fmt.Errorf("permissoes.repository.ListByUsuario scan uuid: %w", err)
	}
	rows, err := q.ListPermissoesByUsuario(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("permissoes.repository.ListByUsuario: %w", err)
	}
	return toSlice(rows), nil
}

func (r *repository) ListByEmpresa(ctx context.Context, empresaID string) ([]*Permissao, error) {
	q := sqlcgen.New(r.pool)
	var eid pgtype.UUID
	if err := eid.Scan(empresaID); err != nil {
		return nil, fmt.Errorf("permissoes.repository.ListByEmpresa scan uuid: %w", err)
	}
	rows, err := q.ListPermissoesByEmpresa(ctx, eid)
	if err != nil {
		return nil, fmt.Errorf("permissoes.repository.ListByEmpresa: %w", err)
	}
	return toSlice(rows), nil
}

func (r *repository) Has(ctx context.Context, usuarioID, empresaID, recurso, acao string) (bool, error) {
	q := sqlcgen.New(r.pool)
	var uid, eid pgtype.UUID
	if err := uid.Scan(usuarioID); err != nil {
		return false, fmt.Errorf("permissoes.repository.Has scan usuario_id: %w", err)
	}
	if err := eid.Scan(empresaID); err != nil {
		return false, fmt.Errorf("permissoes.repository.Has scan empresa_id: %w", err)
	}
	n, err := q.HasPermissao(ctx, sqlcgen.HasPermissaoParams{
		UsuarioID: uid, EmpresaID: eid, Recurso: recurso, Acao: acao,
	})
	if err != nil {
		return false, fmt.Errorf("permissoes.repository.Has: %w", err)
	}
	return n > 0, nil
}

func toPermissao(row sqlcgen.EtlPermisso) *Permissao {
	return &Permissao{
		ID:        uuidToStr(row.ID),
		UsuarioID: uuidToStr(row.UsuarioID),
		EmpresaID: uuidToStr(row.EmpresaID),
		Recurso:   row.Recurso,
		Acao:      row.Acao,
		CreatedAt: row.CreatedAt.Time,
	}
}

func toSlice(rows []sqlcgen.EtlPermisso) []*Permissao {
	result := make([]*Permissao, len(rows))
	for i, row := range rows {
		result[i] = toPermissao(row)
	}
	return result
}

func uuidToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}
