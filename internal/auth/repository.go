package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/sqlc/generated"
)

type Repository interface {
	GetUsuarioByEmail(ctx context.Context, email string) (*Usuario, error)
	GetUsuarioByID(ctx context.Context, id string) (*Usuario, error)
	InsertRefreshToken(ctx context.Context, usuarioID, token string, expiresAt time.Time) (*RefreshToken, error)
	GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	RevokeAllUserTokens(ctx context.Context, usuarioID string) error
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) GetUsuarioByEmail(ctx context.Context, email string) (*Usuario, error) {
	q := sqlcgen.New(r.pool)
	row, err := q.GetUsuarioByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("auth.repository.GetUsuarioByEmail: %w", err)
	}
	return rowToUsuario(row.ID, row.GrupoID, row.Nome, row.Email, row.Password, row.Role, row.Ativo, row.CreatedAt, row.UpdatedAt), nil
}

func (r *repository) GetUsuarioByID(ctx context.Context, id string) (*Usuario, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return nil, fmt.Errorf("auth.repository.GetUsuarioByID scan uuid: %w", err)
	}
	row, err := q.GetUsuarioByID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("auth.repository.GetUsuarioByID: %w", err)
	}
	return rowToUsuario(row.ID, row.GrupoID, row.Nome, row.Email, row.Password, row.Role, row.Ativo, row.CreatedAt, row.UpdatedAt), nil
}

func (r *repository) InsertRefreshToken(ctx context.Context, usuarioID, token string, expiresAt time.Time) (*RefreshToken, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(usuarioID); err != nil {
		return nil, fmt.Errorf("auth.repository.InsertRefreshToken scan uuid: %w", err)
	}
	var exp pgtype.Timestamptz
	if err := exp.Scan(expiresAt); err != nil {
		return nil, fmt.Errorf("auth.repository.InsertRefreshToken scan expires_at: %w", err)
	}
	row, err := q.InsertRefreshToken(ctx, sqlcgen.InsertRefreshTokenParams{
		UsuarioID: uid,
		Token:     token,
		ExpiresAt: exp,
	})
	if err != nil {
		return nil, fmt.Errorf("auth.repository.InsertRefreshToken: %w", err)
	}
	return &RefreshToken{
		ID:        uuidToStr(row.ID),
		UsuarioID: uuidToStr(row.UsuarioID),
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt.Time,
		Revoked:   row.Revoked,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *repository) GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	q := sqlcgen.New(r.pool)
	row, err := q.GetRefreshToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("auth.repository.GetRefreshToken: %w", err)
	}
	return &RefreshToken{
		ID:        uuidToStr(row.ID),
		UsuarioID: uuidToStr(row.UsuarioID),
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt.Time,
		Revoked:   row.Revoked,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *repository) RevokeRefreshToken(ctx context.Context, token string) error {
	q := sqlcgen.New(r.pool)
	if err := q.RevokeRefreshToken(ctx, token); err != nil {
		return fmt.Errorf("auth.repository.RevokeRefreshToken: %w", err)
	}
	return nil
}

func (r *repository) RevokeAllUserTokens(ctx context.Context, usuarioID string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(usuarioID); err != nil {
		return fmt.Errorf("auth.repository.RevokeAllUserTokens scan uuid: %w", err)
	}
	if err := q.RevokeAllUserTokens(ctx, uid); err != nil {
		return fmt.Errorf("auth.repository.RevokeAllUserTokens: %w", err)
	}
	return nil
}

// --- helpers ---

func rowToUsuario(id, grupoID pgtype.UUID, nome, email, password, role string, ativo bool, createdAt, updatedAt pgtype.Timestamptz) *Usuario {
	return &Usuario{
		ID:        uuidToStr(id),
		GrupoID:   uuidToStr(grupoID),
		Nome:      nome,
		Email:     email,
		Password:  password,
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
