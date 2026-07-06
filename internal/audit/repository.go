package audit

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/sqlc/generated"
)

type Repository interface {
	Insert(ctx context.Context, entry LogEntry) error
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) Insert(ctx context.Context, entry LogEntry) error {
	q := sqlcgen.New(r.pool)

	statusCode := int32(entry.StatusCode)
	durationMs := entry.DurationMs

	params := sqlcgen.InsertAuditLogParams{
		RequestID:    pgtype.Text{String: entry.RequestID, Valid: entry.RequestID != ""},
		UserEmail:    pgtype.Text{String: entry.UserEmail, Valid: entry.UserEmail != ""},
		Role:         pgtype.Text{String: entry.Role, Valid: entry.Role != ""},
		Method:       entry.Method,
		Path:         entry.Path,
		QueryParams:  pgtype.Text{String: entry.QueryParams, Valid: entry.QueryParams != ""},
		StatusCode:   &statusCode,
		RequestBody:  pgtype.Text{String: entry.RequestBody, Valid: entry.RequestBody != ""},
		ResponseBody: pgtype.Text{String: entry.ResponseBody, Valid: entry.ResponseBody != ""},
		IpAddress:    pgtype.Text{String: entry.IPAddress, Valid: entry.IPAddress != ""},
		UserAgent:    pgtype.Text{String: entry.UserAgent, Valid: entry.UserAgent != ""},
		DurationMs:   &durationMs,
	}

	if entry.UserID != "" {
		if err := params.UserID.Scan(entry.UserID); err != nil {
			return fmt.Errorf("audit.repository.Insert scan user_id: %w", err)
		}
	}

	if _, err := q.InsertAuditLog(ctx, params); err != nil {
		return fmt.Errorf("audit.repository.Insert: %w", err)
	}

	return nil
}
