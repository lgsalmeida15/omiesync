package etl

import (
	"testing"

	"omie-sync-api/internal/worker"
)

func TestBuildPaginacao_SemDelta(t *testing.T) {
	opts := worker.SyncOptions{}
	p := buildPaginacao(1, 50, opts)

	if p.Pagina != 1 {
		t.Errorf("pagina: got %d want 1", p.Pagina)
	}
	if p.RegistrosPorPagina != 50 {
		t.Errorf("registros_por_pagina: got %d want 50", p.RegistrosPorPagina)
	}
	if p.FiltrarPorDataDe != "" {
		t.Errorf("filtrar_por_data_de deveria ser vazio, got %q", p.FiltrarPorDataDe)
	}
}

func TestBuildPaginacao_ComDelta(t *testing.T) {
	opts := worker.SyncOptions{UltimoSyncAt: "01/06/2026"}
	p := buildPaginacao(3, 50, opts)

	if p.Pagina != 3 {
		t.Errorf("pagina: got %d want 3", p.Pagina)
	}
	if p.FiltrarPorDataDe != "01/06/2026" {
		t.Errorf("filtrar_por_data_de: got %q want 01/06/2026", p.FiltrarPorDataDe)
	}
}

func TestBuildPaginacao_FullIgnoraDelta(t *testing.T) {
	opts := worker.SyncOptions{UltimoSyncAt: "01/06/2026", Full: true}
	p := buildPaginacao(1, 50, opts)

	if p.FiltrarPorDataDe != "" {
		t.Errorf("full=true deve ignorar delta, got %q", p.FiltrarPorDataDe)
	}
}

func TestSyncOptions_DeltaFormatoBrasileiro(t *testing.T) {
	opts := worker.SyncOptions{UltimoSyncAt: "04/06/2026"}
	p := buildPaginacao(1, 50, opts)

	if len(p.FiltrarPorDataDe) != 10 {
		t.Errorf("formato inválido: %q (esperado DD/MM/YYYY)", p.FiltrarPorDataDe)
	}
}

func TestAllExecutors_AssinaturaCorreta(t *testing.T) {
	execs := NewAllExecutors(nil, nopLogger())
	for _, e := range execs {
		// Verificação em compile-time — se compilou, implementa worker.Executor
		var _ worker.Executor = e
		if e.Nome() == "" {
			t.Errorf("executor sem nome")
		}
		t.Logf("✅ %s", e.Nome())
	}
}
