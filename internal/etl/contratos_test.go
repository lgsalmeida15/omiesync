package etl

import (
	"encoding/json"
	"testing"

	"omie-sync-api/internal/worker"
)

// respostaListarContratos é um recorte fiel da documentação do ListarContratos:
// os campos principais vêm aninhados em "cabecalho", não na raiz do elemento.
const respostaListarContratos = `{
  "pagina": 1,
  "total_de_paginas": 1,
  "registros": 2,
  "total_de_registros": 2,
  "contratoCadastro": [
    {
      "cabecalho": {
        "nCodCtr": 2203958,
        "cCodIntCtr": "019289Tb2",
        "nCodCli": 2370765,
        "cNumCtr": "2020/01001",
        "cCodSit": "10",
        "dVigInicial": "29/03/2020",
        "dVigFinal": "30/03/2022",
        "nDiaFat": 30,
        "nValTotMes": 3302,
        "cTipoFat": "01"
      },
      "departamentos": [
        {"cCodDep": "D01", "cDesDep": "Comercial", "nValDep": 2311.4, "nPerDep": 70},
        {"cCodDep": "D02", "cDesDep": "Suporte",   "nValDep": 990.6,  "nPerDep": 30}
      ],
      "infAdic": {
        "cCodCateg": "1.01.02",
        "nCodCC": 1208238,
        "nCodProj": 0
      },
      "itensContrato": [
        {"itemCabecalho": {"codItem": 1, "cCodCategItem": "1.01.03", "valorTotal": 3002}},
        {"itemCabecalho": {"codItem": 2, "cCodCategItem": "",        "valorTotal": 300}}
      ]
    },
    {
      "cabecalho": {
        "nCodCtr": 0,
        "cNumCtr": "SEM-CODIGO"
      },
      "infAdic": {}
    }
  ]
}`

func TestExtractArrayContratos_MapeiaCabecalhoAninhado(t *testing.T) {
	var resp map[string]json.RawMessage
	if err := json.Unmarshal([]byte(respostaListarContratos), &resp); err != nil {
		t.Fatalf("payload de teste inválido: %v", err)
	}

	items, raws, totalPaginas, err := extractArray[OmieContrato](resp, "contratoCadastro")
	if err != nil {
		t.Fatalf("extractArray: %v", err)
	}
	if totalPaginas != 1 {
		t.Errorf("total_de_paginas = %d, esperado 1", totalPaginas)
	}
	if len(items) != 2 {
		t.Fatalf("esperado 2 contratos, got %d", len(items))
	}

	cab := items[0].Cabecalho
	if cab.CodigoContrato != 2203958 {
		t.Errorf("nCodCtr = %d, esperado 2203958", cab.CodigoContrato)
	}
	if cab.CodigoCliente != 2370765 {
		t.Errorf("nCodCli = %d, esperado 2370765", cab.CodigoCliente)
	}
	if cab.NumeroContrato != "2020/01001" {
		t.Errorf("cNumCtr = %q, esperado \"2020/01001\"", cab.NumeroContrato)
	}
	if cab.CodigoSituacao != "10" {
		t.Errorf("cCodSit = %q, esperado \"10\"", cab.CodigoSituacao)
	}
	if cab.DiaFaturamento != 30 {
		t.Errorf("nDiaFat = %d, esperado 30", cab.DiaFaturamento)
	}
	if cab.ValorTotalMes != 3302 {
		t.Errorf("nValTotMes = %v, esperado 3302", cab.ValorTotalMes)
	}
	if cab.TipoFaturamento != "01" {
		t.Errorf("cTipoFat = %q, esperado \"01\"", cab.TipoFaturamento)
	}

	inf := items[0].InfAdic
	if inf.CodigoCategoria != "1.01.02" {
		t.Errorf("cCodCateg = %q, esperado \"1.01.02\"", inf.CodigoCategoria)
	}
	if inf.CodigoContaCorrente != 1208238 {
		t.Errorf("nCodCC = %d, esperado 1208238", inf.CodigoContaCorrente)
	}

	// Vigências chegam em DD/MM/YYYY e viram o literal ISO que o INSERT usa.
	if got := parseOmieDate(cab.VigenciaInicial); got != "2020-03-29" {
		t.Errorf("vigência inicial = %q, esperado \"2020-03-29\"", got)
	}
	if got := parseOmieDate(cab.VigenciaFinal); got != "2022-03-30" {
		t.Errorf("vigência final = %q, esperado \"2022-03-30\"", got)
	}

	// O rateio por departamento e as categorias por item não têm campo tipado —
	// é raw que os carrega para o enriquecimento futuro das matvw. Se o JSON
	// original deixar de ser preservado, esta asserção quebra.
	if len(raws) != 2 {
		t.Fatalf("esperado 2 JSONs brutos, got %d", len(raws))
	}
	var bruto struct {
		Departamentos []struct {
			CodigoDep  string  `json:"cCodDep"`
			Percentual float64 `json:"nPerDep"`
		} `json:"departamentos"`
		ItensContrato []struct {
			ItemCabecalho struct {
				CodigoCategItem string `json:"cCodCategItem"`
			} `json:"itemCabecalho"`
		} `json:"itensContrato"`
	}
	if err := json.Unmarshal(raws[0], &bruto); err != nil {
		t.Fatalf("raw do primeiro contrato não decodifica: %v", err)
	}
	if len(bruto.Departamentos) != 2 || bruto.Departamentos[0].CodigoDep != "D01" {
		t.Errorf("rateio por departamento perdido no raw: %+v", bruto.Departamentos)
	}
	if len(bruto.ItensContrato) != 2 || bruto.ItensContrato[1].ItemCabecalho.CodigoCategItem != "" {
		t.Errorf("itens do contrato perdidos no raw: %+v", bruto.ItensContrato)
	}
}

// upsertContratos pula contrato sem nCodCtr: sem chave, não há o que casar com o
// extrato, e a coluna é NOT NULL. O guard é testado via a mesma condição que ele
// aplica, já que o upsert exige um pool de verdade.
func TestContratos_GuardCodigoZero(t *testing.T) {
	var resp map[string]json.RawMessage
	if err := json.Unmarshal([]byte(respostaListarContratos), &resp); err != nil {
		t.Fatalf("payload de teste inválido: %v", err)
	}
	items, _, _, err := extractArray[OmieContrato](resp, "contratoCadastro")
	if err != nil {
		t.Fatalf("extractArray: %v", err)
	}
	if items[1].Cabecalho.CodigoContrato != 0 {
		t.Fatalf("o segundo item do payload deveria ter nCodCtr = 0, got %d",
			items[1].Cabecalho.CodigoContrato)
	}

	gravaveis := 0
	for _, it := range items {
		if it.Cabecalho.CodigoContrato == 0 {
			continue
		}
		gravaveis++
	}
	if gravaveis != 1 {
		t.Errorf("esperado 1 contrato gravável, got %d", gravaveis)
	}
}

func TestBuildContratosParams_ApenasImportadoAPI(t *testing.T) {
	// Incremental: o ListarContratos aceita filtrar_por_data_de, então o delta vale.
	p := buildContratosParams(2, 50, worker.SyncOptions{UltimoSyncAt: "01/11/2026"})
	if p.ApenasImportadoAPI != "N" {
		t.Errorf("apenas_importado_api = %q, esperado \"N\"", p.ApenasImportadoAPI)
	}
	if p.Pagina != 2 || p.RegistrosPorPagina != 50 {
		t.Errorf("paginação incorreta: %+v", p)
	}
	if p.FiltrarPorDataDe != "01/11/2026" {
		t.Errorf("filtro delta = %q, esperado \"01/11/2026\"", p.FiltrarPorDataDe)
	}

	// Full: sem filtro de data, mas apenas_importado_api continua obrigatório.
	pf := buildContratosParams(1, 50, worker.SyncOptions{Full: true, UltimoSyncAt: "01/11/2026"})
	if pf.ApenasImportadoAPI != "N" {
		t.Errorf("full: apenas_importado_api = %q, esperado \"N\"", pf.ApenasImportadoAPI)
	}
	if pf.FiltrarPorDataDe != "" {
		t.Errorf("full não deveria filtrar por data, got %q", pf.FiltrarPorDataDe)
	}

	// O campo precisa de fato ir no JSON enviado ao Omie — omitempty o esconderia
	// se o valor fosse vazio, e a API rejeita a chamada sem ele.
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var enviado map[string]any
	if err := json.Unmarshal(b, &enviado); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if enviado["apenas_importado_api"] != "N" {
		t.Errorf("payload enviado sem apenas_importado_api: %s", b)
	}
}
