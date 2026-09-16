package ia

import (
	"strings"
	"testing"
	"time"
)

func TestSystemPrompt_VazioUsaOPadrao(t *testing.T) {
	p := SystemPrompt("", 2026, 9)

	if !strings.Contains(p, "NUNCA invente") {
		t.Fatal("não caiu no prompt padrão")
	}
}

/*
O prompt do admin substitui o padrão por inteiro — foi a decisão tomada.

O teste fixa isso: nada do texto versionado deve sobrar, senão o administrador
ajustaria o prompt e continuaria com instruções antigas agindo por baixo, sem
conseguir descobrir de onde vêm.
*/
func TestSystemPrompt_PersonalizadoSubstituiOPadrao(t *testing.T) {
	p := SystemPrompt("Responda sempre em uma frase.", 2026, 9)

	if !strings.Contains(p, "Responda sempre em uma frase.") {
		t.Fatal("ignorou o prompt do administrador")
	}
	if strings.Contains(p, "NUNCA invente") {
		t.Fatal("o padrão sobreviveu à substituição")
	}
}

// Só espaços é o mesmo que vazio: quem apagou o campo esperava o padrão de
// volta, não um prompt em branco.
func TestSystemPrompt_SoEspacosCaiNoPadrao(t *testing.T) {
	if !strings.Contains(SystemPrompt("   \n\t ", 2026, 9), "NUNCA invente") {
		t.Fatal("prompt em branco não restaurou o padrão")
	}
}

/*
O contexto temporal acompanha os dois casos.

Mesmo com prompt personalizado, o modelo precisa saber em que dia está — sem
isso "este mês" responde sobre o mês errado. É informação de execução, não
conteúdo editorial, então não é substituível.
*/
func TestSystemPrompt_ContextoTemporalSempreAcompanha(t *testing.T) {
	for nome, custom := range map[string]string{"padrão": "", "personalizado": "Seja breve."} {
		t.Run(nome, func(t *testing.T) {
			p := SystemPrompt(custom, 2026, 9)
			if !strings.Contains(p, "CONTEXTO ATUAL") {
				t.Fatal("perdeu o bloco de contexto")
			}
			if !strings.Contains(p, "horário de Brasília") {
				t.Fatal("não informou a data ao modelo")
			}
		})
	}
}

/*
O fuso é o ponto do item 3.

O container roda em UTC. Às 22h de Brasília já é o dia seguinte em UTC, e o
assistente responderia "hoje" sobre amanhã. Este teste fixa um instante que
divide os dois dias e confere qual deles o prompt declara.
*/
func TestContextoAtual_UsaOFusoDeBrasiliaENaoUTC(t *testing.T) {
	// 15/09/2026 às 00:30 UTC é ainda 14/09 às 21:30 em Brasília.
	emUTC := time.Date(2026, 9, 15, 0, 30, 0, 0, time.UTC)

	texto := contextoAtual(emUTC.In(fusoBrasilia), 2026, 9)

	if !strings.Contains(texto, "14 de setembro de 2026") {
		t.Fatalf("usou a data de UTC em vez da de Brasília: %q", texto)
	}
	if !strings.Contains(texto, "21:30") {
		t.Fatalf("hora errada: %q", texto)
	}
	if !strings.Contains(texto, "segunda-feira") {
		t.Fatalf("dia da semana errado: %q", texto)
	}
}

/*
Quando a tela mostra um mês que não é o corrente, o modelo precisa ser avisado.

É a confusão mais provável: a pessoa navega para março, pergunta "quanto entrou
este mês", e sem o aviso o modelo trata março como o presente.
*/
func TestContextoAtual_AvisaQuandoATelaNaoEOMesCorrente(t *testing.T) {
	agora := time.Date(2026, 9, 14, 10, 0, 0, 0, fusoBrasilia)

	t.Run("tela em outro mês avisa", func(t *testing.T) {
		texto := contextoAtual(agora, 2026, 3)
		if !strings.Contains(texto, "ATENÇÃO") {
			t.Fatalf("não avisou da divergência: %q", texto)
		}
	})

	t.Run("tela no mês corrente não avisa", func(t *testing.T) {
		texto := contextoAtual(agora, 2026, 9)
		if strings.Contains(texto, "ATENÇÃO") {
			t.Fatalf("avisou sem divergência: %q", texto)
		}
	})
}

// O padrão precisa ensinar os quatro tipos, senão o modelo imita o único
// exemplo que encontrar — que foi exatamente o defeito do "só sai barra".
func TestPromptPadrao_EnsinaTodosOsTiposDeGrafico(t *testing.T) {
	p := PromptPadrao()

	for tipo := range tiposValidos {
		if !strings.Contains(p, `"`+tipo+`"`) {
			t.Errorf("o prompt não exemplifica o tipo %q", tipo)
		}
	}
}

// O JSON solto na resposta foi relatado pelo usuário. O prompt precisa proibir
// explicitamente — antes ele só proibia HTML.
func TestPromptPadrao_ProibeJSONForaDoBloco(t *testing.T) {
	if !strings.Contains(PromptPadrao(), "NUNCA escreva JSON") {
		t.Fatal("o prompt não proíbe JSON solto")
	}
}
