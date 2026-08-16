package etl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"omie-sync-api/internal/omie"
	"omie-sync-api/internal/omie_config"
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

// ── Fatiamento proativo ────────────────────────────────────────────────────

func TestFatiarPeriodo_CobreTudoSemBuraco(t *testing.T) {
	agora := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	fatias := fatiarPeriodo(agora, horizonteAnos, mesesPorFatia)

	if len(fatias) != 4 {
		t.Fatalf("1 ano em fatias de 3 meses = 4 fatias, got %d", len(fatias))
	}
	if !fatias[0].inicio.Equal(truncDay(agora)) {
		t.Errorf("primeira fatia deveria começar hoje: %v", fatias[0].inicio)
	}
	fimEsperado := truncDay(agora).AddDate(horizonteAnos, 0, 0)
	if !fatias[len(fatias)-1].fim.Equal(fimEsperado) {
		t.Errorf("última fatia deveria terminar em %v, got %v", fimEsperado, fatias[len(fatias)-1].fim)
	}

	// Contíguas: cada fatia começa exatamente no dia seguinte ao fim da anterior.
	// É o que garante que nenhum dia fique de fora nem seja contado duas vezes.
	for i := 1; i < len(fatias); i++ {
		esperado := fatias[i-1].fim.AddDate(0, 0, 1)
		if !fatias[i].inicio.Equal(esperado) {
			t.Errorf("fatia %d começa em %v, esperado %v (buraco ou sobreposição)",
				i, fatias[i].inicio, esperado)
		}
		if !fatias[i].fim.After(fatias[i].inicio) {
			t.Errorf("fatia %d invertida: %v..%v", i, fatias[i].inicio, fatias[i].fim)
		}
	}
}

func TestFatiarPeriodo_NaoDependeDaHora(t *testing.T) {
	manha := fatiarPeriodo(time.Date(2026, 8, 16, 0, 1, 0, 0, time.UTC), 1, 3)
	noite := fatiarPeriodo(time.Date(2026, 8, 16, 23, 59, 0, 0, time.UTC), 1, 3)
	if !manha[0].inicio.Equal(noite[0].inicio) || !manha[3].fim.Equal(noite[3].fim) {
		t.Error("o fatiamento não deveria variar com a hora do dia")
	}
}

// ── Linhas de saldo ────────────────────────────────────────────────────────

func TestEhLinhaDeSaldo(t *testing.T) {
	casos := []struct {
		desc string
		want bool
	}{
		{"SALDO", true},
		{"SALDO ANTERIOR", true},
		{" saldo anterior ", true},
		{"GOVERNO DO ESTADO DE SP", false},
		{"SALDOS E EXTRATOS LTDA", false}, // não é prefixo: precisa ser exato
		{"", false},
	}
	for _, c := range casos {
		if got := ehLinhaDeSaldo(OmieExtratoMovimento{Descricao: c.desc}); got != c.want {
			t.Errorf("ehLinhaDeSaldo(%q) = %v, want %v", c.desc, got, c.want)
		}
	}
}

// ── Servidor Omie falso ────────────────────────────────────────────────────

// omieFake responde ao ListarExtrato conforme a regra passada, e registra os
// períodos pedidos para que o teste verifique a cobertura.
type omieFake struct {
	srv       *httptest.Server
	pedidos   []string // "dd/mm/aaaa..dd/mm/aaaa"
	responder func(inicio, fim time.Time) (any, bool) // (corpo, ok); ok=false → erro de volume
}

func novoOmieFake(t *testing.T, responder func(inicio, fim time.Time) (any, bool)) *omieFake {
	t.Helper()
	f := &omieFake{responder: responder}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Param []listarExtratoParams `json:"param"`
		}
		_ = json.Unmarshal(body, &req)
		p := req.Param[0]
		ini, _ := time.Parse("02/01/2006", p.DataInicial)
		fim, _ := time.Parse("02/01/2006", p.DataFinal)
		f.pedidos = append(f.pedidos, p.DataInicial+".."+p.DataFinal)

		corpo, ok := f.responder(ini, fim)
		w.Header().Set("Content-Type", "application/json")
		if !ok {
			// Erro de NEGÓCIO por volume — é este que o código antigo descartava.
			// Client-105 e não Client-8020: este último significa requisição
			// concorrente e o cliente o retenta com esperas de até 180s, o que
			// travaria o teste.
			_ = json.NewEncoder(w).Encode(map[string]string{
				"faultcode":   "SOAP-ENV:Client-105",
				"faultstring": "ERROR: Foram encontrados mais registros do que o permitido",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(corpo)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *omieFake) client() *omie.Client {
	c := omie.NewClient("k", "segredo-nao-vaza")
	c.SetBaseURL(f.srv.URL)
	return c
}

func execExtrato() *ExtratoExecutor {
	return &ExtratoExecutor{log: zerolog.Nop()}
}

func cfgExtrato() *omie_config.EndpointConfig {
	return &omie_config.EndpointConfig{
		EndpointPath: "/financas/extrato/",
		Action:       "ListarExtrato",
		ArrayField:   "listaMovimentos",
	}
}

func movimentosEm(data time.Time, n int) map[string]any {
	lista := make([]map[string]any, 0, n+1)
	for i := 0; i < n; i++ {
		lista = append(lista, map[string]any{
			"cDesCliente":      fmt.Sprintf("FORNECEDOR %d", i),
			"dDataLancamento":  data.Format("02/01/2006"),
			"nValorDocumento":  -100.0,
			"cSituacao":        "Previsto",
		})
	}
	// O Omie sempre acrescenta a linha de saldo.
	lista = append(lista, map[string]any{
		"cDesCliente":     "SALDO",
		"dDataLancamento": data.Format("02/01/2006"),
		"nValorDocumento": 0.0,
	})
	return map[string]any{"listaMovimentos": lista, "cFluxoCaixa": "S"}
}

// ── fetchAdaptive ──────────────────────────────────────────────────────────

// TestFetchAdaptive_SubdivideEmErroDeNegocio reproduz o caso de produção: a conta
// grande recusa janelas largas com erro de negócio. Antes isso devolvia (0, nil) e
// o período sumia; agora tem de subdividir e coletar tudo.
func TestFetchAdaptive_SubdivideEmErroDeNegocio(t *testing.T) {
	fake := novoOmieFake(t, func(ini, fim time.Time) (any, bool) {
		dias := int(fim.Sub(ini).Hours()/24) + 1
		if dias > 45 {
			return nil, false // volume demais
		}
		return movimentosEm(ini, 2), true
	})

	ini := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	fim := ini.AddDate(0, 3, 0).AddDate(0, 0, -1) // um trimestre

	movs, err := execExtrato().fetchAdaptive(context.Background(), fake.client(), 123, ini, fim, cfgExtrato())
	if err != nil {
		t.Fatalf("deveria ter subdividido e concluído, got %v", err)
	}
	if len(movs) == 0 {
		t.Fatal("nenhum movimento coletado: o período foi descartado, é a regressão que estamos evitando")
	}
	for _, m := range movs {
		if ehLinhaDeSaldo(m.mv) {
			t.Error("linha de saldo não deveria ser coletada")
		}
	}
}

// TestFetchAdaptive_PropagaErroNoPiso garante que uma falha irredutível quebra o
// job. Antes o dia era engolido com (0, nil) e o sync terminava "com sucesso".
func TestFetchAdaptive_PropagaErroNoPiso(t *testing.T) {
	fake := novoOmieFake(t, func(ini, fim time.Time) (any, bool) {
		return nil, false // falha sempre, em qualquer janela
	})

	ini := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	movs, err := execExtrato().fetchAdaptive(context.Background(), fake.client(), 123, ini, ini.AddDate(0, 0, 3), cfgExtrato())
	if err == nil {
		t.Fatalf("falha irredutível deveria propagar erro, got %d movimentos e err=nil", len(movs))
	}
}

// TestFetchAdaptive_SemRegistrosNaoEhFalha: conta sem extrato é resposta legítima.
func TestFetchAdaptive_SemRegistrosNaoEhFalha(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"faultcode":   "SOAP-ENV:Client-500",
			"faultstring": "Não existem registros para a página informada",
		})
	}))
	defer srv.Close()

	c := omie.NewClient("k", "s")
	c.SetBaseURL(srv.URL)

	ini := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	movs, err := execExtrato().fetchAdaptive(context.Background(), c, 123, ini, ini.AddDate(0, 1, 0), cfgExtrato())
	if err != nil {
		t.Fatalf("ausência de registros não é falha: %v", err)
	}
	if len(movs) != 0 {
		t.Errorf("esperava zero movimentos, got %d", len(movs))
	}
}

// TestFetchAdaptive_CoberturaContigua verifica que a subdivisão não deixa buraco:
// todo dia do intervalo tem de ser coberto por exatamente uma chamada bem-sucedida.
func TestFetchAdaptive_CoberturaContigua(t *testing.T) {
	var aceitos []fatia
	fake := novoOmieFake(t, func(ini, fim time.Time) (any, bool) {
		dias := int(fim.Sub(ini).Hours()/24) + 1
		if dias > 10 {
			return nil, false
		}
		aceitos = append(aceitos, fatia{ini, fim})
		return movimentosEm(ini, 1), true
	})

	ini := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	fim := ini.AddDate(0, 0, 59)

	if _, err := execExtrato().fetchAdaptive(context.Background(), fake.client(), 123, ini, fim, cfgExtrato()); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	coberto := map[string]int{}
	for _, a := range aceitos {
		for d := a.inicio; !d.After(a.fim); d = d.AddDate(0, 0, 1) {
			coberto[d.Format("2006-01-02")]++
		}
	}
	for d := ini; !d.After(fim); d = d.AddDate(0, 0, 1) {
		switch coberto[d.Format("2006-01-02")] {
		case 1: // ok
		case 0:
			t.Fatalf("dia %s não foi coletado por nenhuma chamada", d.Format("02/01/2006"))
		default:
			t.Fatalf("dia %s coletado %d vezes (duplicaria lançamentos)",
				d.Format("02/01/2006"), coberto[d.Format("2006-01-02")])
		}
	}
}

// TestFetchAdaptive_CenarioProducao é o caso concreto que motivou a correção: a
// janela de 15/08/2026 a 15/08/2027 parava em 13/11/2026, exatamente a fronteira
// do segundo nível da subdivisão binária.
func TestFetchAdaptive_CenarioProducao(t *testing.T) {
	corte := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	fake := novoOmieFake(t, func(ini, fim time.Time) (any, bool) {
		// Recusa qualquer janela que ultrapasse 45 dias e passe do corte — imita a
		// conta cheia, cuja segunda metade tem registros demais.
		if int(fim.Sub(ini).Hours()/24)+1 > 45 {
			return nil, false
		}
		return movimentosEm(ini, 3), true
	})

	ini := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	var todos []movimentoColetado
	for _, f := range fatiarPeriodo(ini, horizonteAnos, mesesPorFatia) {
		movs, err := execExtrato().fetchAdaptive(context.Background(), fake.client(), 2798917380, f.inicio, f.fim, cfgExtrato())
		if err != nil {
			t.Fatalf("fatia %s..%s falhou: %v", f.inicio.Format("02/01/2006"), f.fim.Format("02/01/2006"), err)
		}
		todos = append(todos, movs...)
	}

	// O que interessa: existe dado DEPOIS de 13/11/2026.
	var depoisDoCorte int
	for _, m := range todos {
		d, err := time.Parse("02/01/2006", m.mv.DataLancamento)
		if err == nil && d.After(corte) {
			depoisDoCorte++
		}
	}
	if depoisDoCorte == 0 {
		t.Fatal("nenhum lançamento após 13/11/2026 — a perda de dados persiste")
	}
}
