package webhooks

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/sqlc/generated"
)

type Repository interface {
	ListByGrupo(ctx context.Context, grupoID string) ([]*Webhook, error)
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) ListByGrupo(ctx context.Context, grupoID string) ([]*Webhook, error) {
	q := sqlcgen.New(r.pool)
	var gid pgtype.UUID
	if err := gid.Scan(grupoID); err != nil {
		return nil, fmt.Errorf("webhooks.repository.ListByGrupo scan uuid: %w", err)
	}
	rows, err := q.ListWebhooksByGrupo(ctx, gid)
	if err != nil {
		return nil, fmt.Errorf("webhooks.repository.ListByGrupo: %w", err)
	}
	result := make([]*Webhook, len(rows))
	for i, row := range rows {
		result[i] = &Webhook{
			ID:      uuidToStr(row.ID),
			GrupoID: uuidToStr(row.GrupoID),
			URL:     row.Url,
			Secret:  row.Secret,
			Eventos: row.Eventos,
			Ativo:   row.Ativo,
		}
	}
	return result, nil
}

func uuidToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}
