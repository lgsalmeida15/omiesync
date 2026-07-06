package etl

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestTruncDay(t *testing.T) {
	ts := time.Date(2026, 6, 4, 15, 30, 59, 999, time.UTC)
	got := truncDay(ts)
	if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
		t.Errorf("truncDay deveria zerar hora: %v", got)
	}
	if got.Year() != 2026 || got.Month() != 6 || got.Day() != 4 {
		t.Errorf("truncDay alterou a data: %v", got)
	}
}

func TestIsTimeoutOrRetryable(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{context.DeadlineExceeded, true},
		{fmt.Errorf("omie.client.call: context deadline exceeded"), true},
		{fmt.Errorf("i/o timeout"), true},
		{fmt.Errorf("connection reset by peer"), true},
		{fmt.Errorf("omie error [SOAP-ENV:Client-107]: credencial inválida"), false},
		{fmt.Errorf("omie error [SOAP-ENV:Client-5001]: tag inválida"), false},
		{nil, false},
	}
	for _, tc := range cases {
		got := isTimeoutOrRetryable(tc.err)
		if got != tc.want {
			t.Errorf("isTimeoutOrRetryable(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestExtratoJanelaFutura(t *testing.T) {
	hoje := truncDay(time.Now())
	fim := truncDay(time.Now().AddDate(1, 0, 0))

	if !fim.After(hoje) {
		t.Error("fim deveria ser após hoje")
	}
	dias := int(fim.Sub(hoje).Hours() / 24)
	if dias < 364 || dias > 366 {
		t.Errorf("janela esperada ~365 dias, got %d", dias)
	}
}

func TestFetchAdaptive_MinWindowNaoSubdivide(t *testing.T) {
	inicio := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	fim := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	windowDays := int(fim.Sub(inicio).Hours()/24) + 1
	if windowDays != 1 {
		t.Errorf("windowDays: got %d want 1", windowDays)
	}
	if windowDays > minWindowDays {
		t.Error("janela de 1 dia não deveria subdividir")
	}
}

func TestFetchAdaptive_MeioCorreto(t *testing.T) {
	inicio := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	fim := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	windowDays := int(fim.Sub(inicio).Hours()/24) + 1 // 30

	meio := truncDay(inicio.Add(time.Duration(windowDays/2) * 24 * time.Hour))

	// 30/2 = 15 dias a partir de 01/06 = dia 16
	if meio.Day() != 16 {
		t.Errorf("meio: esperado dia 16, got dia %d", meio.Day())
	}

	// Primeira metade termina em 15/06
	fim1 := meio.AddDate(0, 0, -1)
	if fim1.Day() != 15 {
		t.Errorf("fim1: esperado dia 15, got dia %d", fim1.Day())
	}
}

func TestFetchAdaptive_SubdivisaoBinariaJanelas(t *testing.T) {
	// Verifica que a subdivisão produz janelas corretas em vários cenários
	casos := []struct {
		inicio     time.Time
		fim        time.Time
		wantMetade int // dia esperado do ponto médio
	}{
		{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
			16, // 30 dias → meio = dia 16
		},
		{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC),
			4, // 7 dias → meio = dia 4
		},
		{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
			2, // 2 dias → meio = dia 2
		},
	}

	for _, tc := range casos {
		windowDays := int(tc.fim.Sub(tc.inicio).Hours()/24) + 1
		meio := truncDay(tc.inicio.Add(time.Duration(windowDays/2) * 24 * time.Hour))
		if meio.Day() != tc.wantMetade {
			t.Errorf("janela %d-%d: meio esperado dia %d, got dia %d",
				tc.inicio.Day(), tc.fim.Day(), tc.wantMetade, meio.Day())
		}
	}
}

func TestIsTimeoutOrRetryable_NaoRetriaErroOmie(t *testing.T) {
	// Erros de negócio do Omie NÃO devem ser retentados com subdivisão
	errosNegocio := []error{
		fmt.Errorf("omie error [SOAP-ENV:Client-107]: credencial inválida"),
		fmt.Errorf("omie error [SOAP-ENV:Client-500]: registro não encontrado"),
		fmt.Errorf("omie error [SOAP-ENV:Client-1003]: campo obrigatório"),
	}
	for _, err := range errosNegocio {
		if isTimeoutOrRetryable(err) {
			t.Errorf("erro de negócio não deve ser retryable: %v", err)
		}
	}
}
