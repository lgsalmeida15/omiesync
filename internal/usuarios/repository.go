package usuarios

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/sqlc/generated"
)

// ErrMigrationPendente é retornado quando usuario_grupos ainda não existe.
var ErrMigrationPendente = errors.New("migration 000023 pendente")

type Repository interface {
	Insert(ctx context.Context, grupoID, nome, email, passwordHash, role string) (*Usuario, error)
	InsertGrupoVinculo(ctx context.Context, usuarioID, grupoID string) error
	GetByID(ctx context.Context, id string) (*Usuario, error)
	GetByEmail(ctx context.Context, email string) (*Usuario, error)
	HasGrupoVinculo(ctx context.Context, usuarioID, grupoID string) (bool, error)
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

func (r *repository) GetByEmail(ctx context.Context, email string) (*Usuario, error) {
	q := sqlcgen.New(r.pool)
	row, err := q.GetUsuarioByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("usuarios.repository.GetByEmail: %w", err)
	}
	return toUsuario(row.ID, row.GrupoID, row.Nome, row.Email, row.Role, row.Ativo, row.CreatedAt, row.UpdatedAt), nil
}

func (r *repository) HasGrupoVinculo(ctx context.Context, usuarioID, grupoID string) (bool, error) {
	const q = `SELECT COUNT(*) > 0 FROM _etl.usuario_grupos WHERE usuario_id = $1::uuid AND grupo_id = $2::uuid`
	var exists bool
	if err := r.pool.QueryRow(ctx, q, usuarioID, grupoID).Scan(&exists); err != nil {
		if isUndefinedTable(err) {
			return false, nil
		}
		return false, fmt.Errorf("usuarios.repository.HasGrupoVinculo: %w", err)
	}
	return exists, nil
}

func (r *repository) InsertGrupoVinculo(ctx context.Context, usuarioID, grupoID string) error {
	const q = `INSERT INTO _etl.usuario_grupos (usuario_id, grupo_id) VALUES ($1::uuid, $2::uuid) ON CONFLICT DO NOTHING`
	if _, err := r.pool.Exec(ctx, q, usuarioID, grupoID); err != nil {
		if isUndefinedTable(err) {
			return ErrMigrationPendente
		}
		return fmt.Errorf("usuarios.repository.InsertGrupoVinculo: %w", err)
	}
	return nil
}

// isUndefinedTable detecta erro PostgreSQL 42P01 (tabela não existe — migration pendente).
func isUndefinedTable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
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
	// Tenta junction table primeiro; cai no legado se migration ainda não foi aplicada
	const qJunction = `
		SELECT u.id, u.grupo_id, u.nome, u.email, u.role, u.ativo, u.created_at, u.updated_at
		FROM _etl.usuarios u
		JOIN _etl.usuario_grupos ug ON ug.usuario_id = u.id
		WHERE ug.grupo_id = $1::uuid AND u.deleted_at IS NULL
		ORDER BY u.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, qJunction, grupoID, limit, offset)
	if err != nil {
		if !isUndefinedTable(err) {
			return nil, fmt.Errorf("usuarios.repository.List: %w", err)
		}
		// Fallback legado
		return r.listLegacy(ctx, grupoID, limit, offset)
	}
	defer rows.Close()

	var result []*Usuario
	for rows.Next() {
		var u Usuario
		var id, gid pgtype.UUID
		var ca, ua pgtype.Timestamptz
		if err := rows.Scan(&id, &gid, &u.Nome, &u.Email, &u.Role, &u.Ativo, &ca, &ua); err != nil {
			return nil, fmt.Errorf("usuarios.repository.List scan: %w", err)
		}
		u.ID = uuidToStr(id)
		u.GrupoID = uuidToStr(gid)
		u.CreatedAt = ca.Time
		u.UpdatedAt = ua.Time
		result = append(result, &u)
	}
	return result, rows.Err()
}

func (r *repository) listLegacy(ctx context.Context, grupoID string, limit, offset int32) ([]*Usuario, error) {
	q := sqlcgen.New(r.pool)
	var gid pgtype.UUID
	if err := gid.Scan(grupoID); err != nil {
		return nil, fmt.Errorf("usuarios.repository.listLegacy scan: %w", err)
	}
	rows, err := q.ListUsuariosByGrupo(ctx, sqlcgen.ListUsuariosByGrupoParams{GrupoID: gid, Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("usuarios.repository.listLegacy: %w", err)
	}
	result := make([]*Usuario, len(rows))
	for i, row := range rows {
		result[i] = toUsuario(row.ID, row.GrupoID, row.Nome, row.Email, row.Role, row.Ativo, row.CreatedAt, row.UpdatedAt)
	}
	return result, nil
}

func (r *repository) Count(ctx context.Context, grupoID string) (int64, error) {
	const qJunction = `
		SELECT COUNT(*)
		FROM _etl.usuarios u
		JOIN _etl.usuario_grupos ug ON ug.usuario_id = u.id
		WHERE ug.grupo_id = $1::uuid AND u.deleted_at IS NULL`

	var n int64
	err := r.pool.QueryRow(ctx, qJunction, grupoID).Scan(&n)
	if err != nil {
		if !isUndefinedTable(err) {
			return 0, fmt.Errorf("usuarios.repository.Count: %w", err)
		}
		// Fallback legado
		return r.countLegacy(ctx, grupoID)
	}
	return n, nil
}

func (r *repository) countLegacy(ctx context.Context, grupoID string) (int64, error) {
	q := sqlcgen.New(r.pool)
	var gid pgtype.UUID
	if err := gid.Scan(grupoID); err != nil {
		return 0, fmt.Errorf("usuarios.repository.countLegacy scan: %w", err)
	}
	n, err := q.CountUsuariosByGrupo(ctx, gid)
	if err != nil {
		return 0, fmt.Errorf("usuarios.repository.countLegacy: %w", err)
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
