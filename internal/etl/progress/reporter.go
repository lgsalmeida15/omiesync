package progress

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/internal/sync"
	"omie-sync-api/sqlc/generated"
)

type Reporter interface {
	Start(ctx context.Context, jobID, executor string) error
	UpdatePage(ctx context.Context, jobID, executor string, paginaAtual, totalPaginas, registrosProc, registrosTotal int, payload, response []byte) error
	Done(ctx context.Context, jobID, executor string, totalRegistros int) error
	Fail(ctx context.Context, jobID, executor string, err error, payload []byte, response string) error
	Skip(ctx context.Context, jobID, executor, motivo string) error
	Init(ctx context.Context, jobID string, executors []string) error
	SetHub(hub *sync.SSEHub, empresaID string)
	Heartbeat(ctx context.Context) error
}

type DBReporter struct {
	pool      *pgxpool.Pool
	syncRepo  sync.Repository
	queries   *sqlcgen.Queries
	hub       *sync.SSEHub
	empresaID string
	jobID     string
}

func NewDBReporter(pool *pgxpool.Pool, syncRepo sync.Repository) *DBReporter {
	return &DBReporter{
		pool:     pool,
		syncRepo: syncRepo,
		queries:  sqlcgen.New(pool),
	}
}

func (r *DBReporter) SetHub(hub *sync.SSEHub, empresaID string) {
	r.hub = hub
	r.empresaID = empresaID
}

func (r *DBReporter) publish(evtType string, data any) {
	if r.hub != nil && r.empresaID != "" {
		r.hub.Publish(r.empresaID, sync.SSEEvent{
			Type: evtType,
			Data: data,
		})
	}
}

func (r *DBReporter) Init(ctx context.Context, jobID string, executors []string) error {
	r.jobID = jobID
	var uid pgtype.UUID
	if err := uid.Scan(jobID); err != nil {
		return fmt.Errorf("DBReporter.Init scan uuid: %w", err)
	}
	for _, exec := range executors {
		err := r.queries.InitSyncJobProgress(ctx, sqlcgen.InitSyncJobProgressParams{
			JobID:    uid,
			Executor: exec,
		})
		if err != nil {
			return fmt.Errorf("DBReporter.Init insert %s: %w", exec, err)
		}
	}
	return nil
}

func (r *DBReporter) Start(ctx context.Context, jobID, executor string) error {
	var uid pgtype.UUID
	if err := uid.Scan(jobID); err != nil {
		return fmt.Errorf("DBReporter.Start scan uuid: %w", err)
	}
	err := r.queries.StartSyncJobProgress(ctx, sqlcgen.StartSyncJobProgressParams{
		JobID:    uid,
		Executor: executor,
	})
	if err != nil {
		return fmt.Errorf("DBReporter.Start: %w", err)
	}

	r.publish("modulo.iniciado", map[string]string{
		"job_id":   jobID,
		"executor": executor,
	})
	return nil
}

func (r *DBReporter) UpdatePage(ctx context.Context, jobID, executor string, paginaAtual, totalPaginas, registrosProc, registrosTotal int, payload, response []byte) error {
	var uid pgtype.UUID
	if err := uid.Scan(jobID); err != nil {
		return fmt.Errorf("DBReporter.UpdatePage scan uuid: %w", err)
	}

	pAtual := int32(paginaAtual)
	pTotal := int32(totalPaginas)
	rTotal := int32(registrosTotal)

	err := r.queries.UpdateSyncJobProgressPage(ctx, sqlcgen.UpdateSyncJobProgressPageParams{
		JobID:          uid,
		Executor:       executor,
		PaginaAtual:    &pAtual,
		TotalPaginas:   &pTotal,
		RegistrosProc:  int32(registrosProc),
		RegistrosTotal: &rTotal,
		UltimoPayload:  payload,
		UltimoResponse: response,
	})

	// Publica SSE independente do resultado do banco — o frontend não pode parar de receber atualizações
	// por causa de uma lentidão ou erro transitório de escrita.
	r.publish("modulo.progresso", map[string]any{
		"job_id":          jobID,
		"executor":        executor,
		"pagina_atual":    paginaAtual,
		"total_paginas":   totalPaginas,
		"registros_proc":  registrosProc,
		"registros_total": registrosTotal,
	})
	if err != nil {
		return fmt.Errorf("DBReporter.UpdatePage: %w", err)
	}
	return nil
}

func (r *DBReporter) Done(ctx context.Context, jobID, executor string, totalRegistros int) error {
	var uid pgtype.UUID
	if err := uid.Scan(jobID); err != nil {
		return fmt.Errorf("DBReporter.Done scan uuid: %w", err)
	}
	err := r.queries.DoneSyncJobProgress(ctx, sqlcgen.DoneSyncJobProgressParams{
		JobID:         uid,
		Executor:      executor,
		RegistrosProc: int32(totalRegistros),
	})
	if err != nil {
		return fmt.Errorf("DBReporter.Done: %w", err)
	}

	r.publish("modulo.concluido", map[string]any{
		"job_id":          jobID,
		"executor":        executor,
		"registros_total": totalRegistros,
	})
	return nil
}

func (r *DBReporter) Fail(ctx context.Context, jobID, executor string, err error, payload []byte, response string) error {
	var uid pgtype.UUID
	if scanErr := uid.Scan(jobID); scanErr != nil {
		return fmt.Errorf("DBReporter.Fail scan uuid: %w", scanErr)
	}
	dbErr := r.queries.FailSyncJobProgress(ctx, sqlcgen.FailSyncJobProgressParams{
		JobID:        uid,
		Executor:     executor,
		Erro:         pgtype.Text{String: err.Error(), Valid: true},
		ErroPayload:  payload,
		ErroResponse: pgtype.Text{String: response, Valid: true},
	})
	if dbErr != nil {
		return fmt.Errorf("DBReporter.Fail: %w", dbErr)
	}

	r.publish("modulo.erro", map[string]any{
		"job_id":   jobID,
		"executor": executor,
		"erro":     err.Error(),
	})
	return nil
}

func (r *DBReporter) Skip(ctx context.Context, jobID, executor, motivo string) error {
	var uid pgtype.UUID
	if err := uid.Scan(jobID); err != nil {
		return fmt.Errorf("DBReporter.Skip scan uuid: %w", err)
	}
	err := r.queries.SkipSyncJobProgress(ctx, sqlcgen.SkipSyncJobProgressParams{
		JobID:    uid,
		Executor: executor,
		Erro:     pgtype.Text{String: motivo, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("DBReporter.Skip: %w", err)
	}
	return nil
}

func (r *DBReporter) Heartbeat(ctx context.Context) error {
	if r.syncRepo == nil || r.jobID == "" {
		return nil
	}
	return r.syncRepo.UpdateJobHeartbeat(ctx, r.jobID)
}

type NoopReporter struct{}

func (r *NoopReporter) Init(ctx context.Context, jobID string, executors []string) error { return nil }
func (r *NoopReporter) Start(ctx context.Context, jobID, executor string) error          { return nil }
func (r *NoopReporter) UpdatePage(ctx context.Context, jobID, executor string, paginaAtual, totalPaginas, registrosProc, registrosTotal int, payload, response []byte) error {
	return nil
}
func (r *NoopReporter) Done(ctx context.Context, jobID, executor string, totalRegistros int) error {
	return nil
}
func (r *NoopReporter) Fail(ctx context.Context, jobID, executor string, err error, payload []byte, response string) error {
	return nil
}
func (r *NoopReporter) Skip(ctx context.Context, jobID, executor, motivo string) error {
	return nil
}
func (r *NoopReporter) SetHub(hub *sync.SSEHub, empresaID string) {}
func (r *NoopReporter) Heartbeat(ctx context.Context) error      { return nil }
