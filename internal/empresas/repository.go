package empresas

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/sqlc/generated"
)

type Repository interface {
	Insert(ctx context.Context, grupoID, nome, cnpj, appKey, appSecret string) (*Empresa, error)
	GetByID(ctx context.Context, id string) (*Empresa, error)
	List(ctx context.Context, grupoID string, limit, offset int32) ([]*Empresa, error)
	Count(ctx context.Context, grupoID string) (int64, error)
	Update(ctx context.Context, id, nome, cnpj, appKey, appSecret string) (*Empresa, error)
	MarkDeletando(ctx context.Context, id string) error
	InsertDeletionQueue(ctx context.Context, empresaID string, executeAt time.Time) error
	Reativar(ctx context.Context, id string) error
	CancelDeletionQueue(ctx context.Context, empresaID string) error
	ListPendingDeletions(ctx context.Context) ([]PendingDeletion, error)
	MarkDeletionExecuted(ctx context.Context, deletionID string) error
}

type PendingDeletion struct {
	ID        string
	EmpresaID string
	ExecuteAt time.Time
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) Insert(ctx context.Context, grupoID, nome, cnpj, appKey, appSecret string) (*Empresa, error) {
	q := sqlcgen.New(r.pool)
	var gid pgtype.UUID
	if err := gid.Scan(grupoID); err != nil {
		return nil, fmt.Errorf("empresas.repository.Insert scan grupo_id: %w", err)
	}
	row, err := q.InsertEmpresa(ctx, sqlcgen.InsertEmpresaParams{
		GrupoID:   gid,
		Nome:      nome,
		Cnpj:      pgtype.Text{String: cnpj, Valid: cnpj != ""},
		AppKey:    appKey,
		AppSecret: appSecret,
	})
	if err != nil {
		return nil, fmt.Errorf("empresas.repository.Insert: %w", err)
	}
	return rowToEmpresa(row.ID, row.GrupoID, row.Nome, row.Cnpj, row.AppKey, row.AppSecret, row.Status, row.StatusSync, row.UltimoSyncAt, row.CreatedAt, row.UpdatedAt), nil
}

func (r *repository) GetByID(ctx context.Context, id string) (*Empresa, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return nil, fmt.Errorf("empresas.repository.GetByID scan uuid: %w", err)
	}
	row, err := q.GetEmpresaByID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("empresas.repository.GetByID: %w", err)
	}
	return rowToEmpresa(row.ID, row.GrupoID, row.Nome, row.Cnpj, row.AppKey, row.AppSecret, row.Status, row.StatusSync, row.UltimoSyncAt, row.CreatedAt, row.UpdatedAt), nil
}

func (r *repository) List(ctx context.Context, grupoID string, limit, offset int32) ([]*Empresa, error) {
	q := sqlcgen.New(r.pool)
	var gid pgtype.UUID
	if err := gid.Scan(grupoID); err != nil {
		return nil, fmt.Errorf("empresas.repository.List scan grupo_id: %w", err)
	}
	rows, err := q.ListEmpresasByGrupo(ctx, sqlcgen.ListEmpresasByGrupoParams{GrupoID: gid, Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("empresas.repository.List: %w", err)
	}
	result := make([]*Empresa, len(rows))
	for i, row := range rows {
		result[i] = rowToEmpresa(row.ID, row.GrupoID, row.Nome, row.Cnpj, row.AppKey, row.AppSecret, row.Status, row.StatusSync, row.UltimoSyncAt, row.CreatedAt, row.UpdatedAt)
	}
	return result, nil
}

func (r *repository) Count(ctx context.Context, grupoID string) (int64, error) {
	q := sqlcgen.New(r.pool)
	var gid pgtype.UUID
	if err := gid.Scan(grupoID); err != nil {
		return 0, fmt.Errorf("empresas.repository.Count scan grupo_id: %w", err)
	}
	n, err := q.CountEmpresasByGrupo(ctx, gid)
	if err != nil {
		return 0, fmt.Errorf("empresas.repository.Count: %w", err)
	}
	return n, nil
}

func (r *repository) Update(ctx context.Context, id, nome, cnpj, appKey, appSecret string) (*Empresa, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return nil, fmt.Errorf("empresas.repository.Update scan uuid: %w", err)
	}
	row, err := q.UpdateEmpresa(ctx, sqlcgen.UpdateEmpresaParams{
		ID:        uid,
		Nome:      nome,
		Cnpj:      pgtype.Text{String: cnpj, Valid: cnpj != ""},
		AppKey:    appKey,
		AppSecret: appSecret,
	})
	if err != nil {
		return nil, fmt.Errorf("empresas.repository.Update: %w", err)
	}
	return rowToEmpresa(row.ID, row.GrupoID, row.Nome, row.Cnpj, row.AppKey, row.AppSecret, row.Status, row.StatusSync, row.UltimoSyncAt, row.CreatedAt, row.UpdatedAt), nil
}

func (r *repository) MarkDeletando(ctx context.Context, id string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return fmt.Errorf("empresas.repository.MarkDeletando scan uuid: %w", err)
	}
	if err := q.MarkEmpresaDeletando(ctx, uid); err != nil {
		return fmt.Errorf("empresas.repository.MarkDeletando: %w", err)
	}
	return nil
}

func (r *repository) InsertDeletionQueue(ctx context.Context, empresaID string, executeAt time.Time) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return fmt.Errorf("empresas.repository.InsertDeletionQueue scan uuid: %w", err)
	}
	var exp pgtype.Timestamptz
	if err := exp.Scan(executeAt); err != nil {
		return fmt.Errorf("empresas.repository.InsertDeletionQueue scan execute_at: %w", err)
	}
	if _, err := q.InsertDeletionQueue(ctx, sqlcgen.InsertDeletionQueueParams{EmpresaID: uid, ExecuteAt: exp}); err != nil {
		return fmt.Errorf("empresas.repository.InsertDeletionQueue: %w", err)
	}
	return nil
}

func (r *repository) Reativar(ctx context.Context, id string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return fmt.Errorf("empresas.repository.Reativar scan uuid: %w", err)
	}
	if err := q.ReativarEmpresa(ctx, uid); err != nil {
		return fmt.Errorf("empresas.repository.Reativar: %w", err)
	}
	return nil
}

func (r *repository) CancelDeletionQueue(ctx context.Context, empresaID string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return fmt.Errorf("empresas.repository.CancelDeletionQueue scan uuid: %w", err)
	}
	if err := q.CancelDeletionQueue(ctx, uid); err != nil {
		return fmt.Errorf("empresas.repository.CancelDeletionQueue: %w", err)
	}
	return nil
}

func (r *repository) ListPendingDeletions(ctx context.Context) ([]PendingDeletion, error) {
	q := sqlcgen.New(r.pool)
	rows, err := q.ListPendingDeletions(ctx)
	if err != nil {
		return nil, fmt.Errorf("empresas.repository.ListPendingDeletions: %w", err)
	}
	result := make([]PendingDeletion, len(rows))
	for i, row := range rows {
		result[i] = PendingDeletion{
			ID:        uuidToStr(row.ID),
			EmpresaID: uuidToStr(row.EmpresaID),
			ExecuteAt: row.ExecuteAt.Time,
		}
	}
	return result, nil
}

func (r *repository) MarkDeletionExecuted(ctx context.Context, deletionID string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(deletionID); err != nil {
		return fmt.Errorf("empresas.repository.MarkDeletionExecuted scan uuid: %w", err)
	}
	if err := q.MarkDeletionExecuted(ctx, uid); err != nil {
		return fmt.Errorf("empresas.repository.MarkDeletionExecuted: %w", err)
	}
	return nil
}

// --- helpers ---

func rowToEmpresa(id, grupoID pgtype.UUID, nome string, cnpj pgtype.Text, appKey, appSecret, status, statusSync string, ultimoSyncAt, createdAt, updatedAt pgtype.Timestamptz) *Empresa {
	e := &Empresa{
		ID:         uuidToStr(id),
		GrupoID:    uuidToStr(grupoID),
		Nome:       nome,
		CNPJ:       cnpj.String,
		AppKey:     appKey,
		AppSecret:  appSecret,
		Status:     status,
		StatusSync: statusSync,
		CreatedAt:  createdAt.Time,
		UpdatedAt:  updatedAt.Time,
	}
	if ultimoSyncAt.Valid {
		t := ultimoSyncAt.Time
		e.UltimoSyncAt = &t
	}
	return e
}

func uuidToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}
