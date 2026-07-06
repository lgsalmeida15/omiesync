package omie_config

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/sqlc/generated"
)

type Repository interface {
	GetAll(ctx context.Context) ([]*EndpointConfig, error)
	GetByModulo(ctx context.Context, modulo string) (*EndpointConfig, error)
	Update(ctx context.Context, modulo string, req UpdateRequest, userID string) (*EndpointConfig, error)
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) GetAll(ctx context.Context) ([]*EndpointConfig, error) {
	q := sqlcgen.New(r.pool)
	rows, err := q.ListOmieEndpointConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("omie_config.repository.GetAll: %w", err)
	}

	result := make([]*EndpointConfig, len(rows))
	for i, row := range rows {
		result[i] = toConfig(row)
	}
	return result, nil
}

func (r *repository) GetByModulo(ctx context.Context, modulo string) (*EndpointConfig, error) {
	q := sqlcgen.New(r.pool)
	row, err := q.GetOmieEndpointConfigByModulo(ctx, modulo)
	if err != nil {
		return nil, fmt.Errorf("omie_config.repository.GetByModulo: %w", err)
	}
	return toConfig(sqlcgen.ListOmieEndpointConfigsRow(row)), nil
}

func (r *repository) Update(ctx context.Context, modulo string, req UpdateRequest, userID string) (*EndpointConfig, error) {
	q := sqlcgen.New(r.pool)
	
	var uid pgtype.UUID
	if err := uid.Scan(userID); err != nil {
		return nil, fmt.Errorf("omie_config.repository.Update scan user uuid: %w", err)
	}

	row, err := q.UpdateOmieEndpointConfig(ctx, sqlcgen.UpdateOmieEndpointConfigParams{
		Modulo:       modulo,
		EndpointPath: req.EndpointPath,
		Action:       req.Action,
		ArrayField:   req.ArrayField,
		PageSize:     int32(req.PageSize),
		Ativo:        req.Ativo,
		Notas:        pgtype.Text{String: req.Notas, Valid: req.Notas != ""},
		UpdatedBy:    uid,
	})
	if err != nil {
		return nil, fmt.Errorf("omie_config.repository.Update: %w", err)
	}

	// Recarrega para pegar o email do updated_by
	return r.GetByModulo(ctx, row.Modulo)
}

func toConfig(row sqlcgen.ListOmieEndpointConfigsRow) *EndpointConfig {
	c := &EndpointConfig{
		ID:             uuidToStr(row.ID),
		Modulo:         row.Modulo,
		EndpointPath:   row.EndpointPath,
		Action:         row.Action,
		ArrayField:     row.ArrayField,
		PageSize:       int(row.PageSize),
		Ativo:          row.Ativo,
		IgnorarDelta:   row.IgnorarDelta,
		Notas:          row.Notas.String,
		UpdatedAt:      row.UpdatedAt.Time,
		UpdatedByEmail: row.UpdatedByEmail.String,
	}
	if row.UpdatedBy.Valid {
		c.UpdatedBy = uuidToStr(row.UpdatedBy)
	}
	return c
}

func uuidToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		u.Bytes[0], u.Bytes[1], u.Bytes[2], u.Bytes[3],
		u.Bytes[4], u.Bytes[5],
		u.Bytes[6], u.Bytes[7],
		u.Bytes[8], u.Bytes[9],
		u.Bytes[10], u.Bytes[11], u.Bytes[12], u.Bytes[13], u.Bytes[14], u.Bytes[15])
}
