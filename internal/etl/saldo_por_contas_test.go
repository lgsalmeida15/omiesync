package etl

import (
	"testing"
	"time"
)

func data(ano int, mes time.Month) time.Time {
	return time.Date(ano, mes, 1, 0, 0, 0, 0, time.UTC)
}

func chave(ano int, mes time.Month) string {
	return data(ano, mes).Format("2006-01")
}

// Sem nada sincronizado ainda, a janela cobre ano anterior completo + ano
// corrente até o mês presente — nunca um mês futuro do ano corrente.
func TestPeriodosParaSincronizar_BackfillCompleto(t *testing.T) {
	agora := data(2026, time.March)
	out := periodosParaSincronizar(agora, map[string]bool{})

	esperado := 12 + 3 // ano anterior inteiro + jan/fev/mar do ano corrente
	if len(out) != esperado {
		t.Fatalf("devolveu %d períodos, esperava %d", len(out), esperado)
	}

	for _, p := range out {
		if p.After(agora) {
			t.Errorf("incluiu mês futuro: %s", p.Format("01/2006"))
		}
	}

	maisAntigo := data(2025, time.January)
	if out[0] != maisAntigo {
		t.Fatalf("primeiro período = %s, esperava %s", out[0].Format("01/2006"), maisAntigo.Format("01/2006"))
	}
}

// Mês futuro do ano corrente nunca entra, mesmo que o ano inteiro esteja
// marcado como "corrente" na janela de backfill.
func TestPeriodosParaSincronizar_NuncaInclueMesFuturo(t *testing.T) {
	agora := data(2026, time.February)
	out := periodosParaSincronizar(agora, map[string]bool{})

	for _, p := range out {
		if p.Year() == 2026 && p.Month() > time.February {
			t.Fatalf("incluiu mês futuro do ano corrente: %s", p.Format("01/2006"))
		}
	}
}

// Os últimos mesesResync meses (mês corrente + 2 anteriores) são SEMPRE
// repedidos, mesmo já tendo registro — por causa da conciliação bancária
// tardia do Omie.
func TestPeriodosParaSincronizar_UltimosTresMesesSempreRepetidos(t *testing.T) {
	agora := data(2026, time.June)
	jaSincronizados := map[string]bool{
		chave(2026, time.April): true,
		chave(2026, time.May):   true,
		chave(2026, time.June):  true,
	}

	out := periodosParaSincronizar(agora, jaSincronizados)

	for _, mes := range []time.Month{time.April, time.May, time.June} {
		achou := false
		for _, p := range out {
			if p.Year() == 2026 && p.Month() == mes {
				achou = true
			}
		}
		if !achou {
			t.Errorf("mês %s já sincronizado, mas está dentro da janela de re-sync e devia aparecer", mes)
		}
	}
}

// Meses mais antigos que a janela de re-sync, uma vez sincronizados, não
// geram nova chamada — são tratados como imutáveis.
func TestPeriodosParaSincronizar_MesesAntigosJaSincronizadosFicamDeFora(t *testing.T) {
	agora := data(2026, time.June)
	jaSincronizados := map[string]bool{
		chave(2026, time.January):   true,
		chave(2025, time.December): true,
	}

	out := periodosParaSincronizar(agora, jaSincronizados)

	for _, p := range out {
		if p.Year() == 2026 && p.Month() == time.January {
			t.Error("janeiro de 2026 já sincronizado e fora da janela de re-sync não devia ter sido repedido")
		}
		if p.Year() == 2025 && p.Month() == time.December {
			t.Error("dezembro de 2025 já sincronizado e fora da janela de re-sync não devia ter sido repedido")
		}
	}
}

// Meses antigos AINDA sem registro continuam entrando na lista, mesmo fora da
// janela de re-sync — é o backfill incompleto sendo completado aos poucos.
func TestPeriodosParaSincronizar_MesesAntigosSemRegistroContinuamPendentes(t *testing.T) {
	agora := data(2026, time.June)
	out := periodosParaSincronizar(agora, map[string]bool{})

	achouJaneiro := false
	for _, p := range out {
		if p.Year() == 2026 && p.Month() == time.January {
			achouJaneiro = true
		}
	}
	if !achouJaneiro {
		t.Error("janeiro de 2026 sem registro devia continuar pendente")
	}
}

// Virada de ano: em janeiro, a janela de re-sync (3 meses) cruza para o ano
// anterior — novembro e dezembro do ano passado continuam sendo repedidos.
func TestPeriodosParaSincronizar_ViradaDeAno(t *testing.T) {
	agora := data(2026, time.January)
	jaSincronizados := map[string]bool{
		chave(2025, time.November): true,
		chave(2025, time.December): true,
		chave(2026, time.January):  true,
	}

	out := periodosParaSincronizar(agora, jaSincronizados)

	for _, mes := range []struct {
		ano int
		m   time.Month
	}{{2025, time.November}, {2025, time.December}, {2026, time.January}} {
		achou := false
		for _, p := range out {
			if p.Year() == mes.ano && p.Month() == mes.m {
				achou = true
			}
		}
		if !achou {
			t.Errorf("%d/%d está dentro da janela de re-sync na virada do ano e devia aparecer", mes.m, mes.ano)
		}
	}
}
