# Mapeamento: View Gerencial — Estrutura Antiga → Estrutura Nova

Este documento registra a equivalência entre a estrutura de dados da versão anterior
(tudo em JSONB `dados_json`) e a estrutura atual do omie-sync (colunas tipadas + `raw` JSONB).

---

## Tabelas de Origem

| Tabela Antiga          | Tabela Nova              | Observação                                      |
|------------------------|--------------------------|-------------------------------------------------|
| `contas_a_pagar`       | `contas_pagar`           | Nome alterado; dados tipados + `raw` completo   |
| `contas_a_receber`     | `contas_receber`         | Nome alterado; dados tipados + `raw` completo   |
| `movimentos`           | `movimentos_financeiros` | Nome alterado; estrutura aninhada em `raw`      |
| `contas_corrente`      | `contas_correntes`       | Nome alterado; extrato **separado** (ver abaixo)|
| `categorias`           | `categorias`             | Igual; campos promovidos para colunas           |
| `departamentos`        | `departamentos`          | Igual; campos promovidos para colunas           |
| `clientes`             | `clientes`               | Igual; campos promovidos para colunas           |
| *(dentro contas_corrente.dados_json → 'extrato')* | `extrato` | **Agora é tabela própria** |

---

## Views Antigas → Equivalente Novo

| View Antiga                                      | Equivalente Novo                                                    |
|--------------------------------------------------|----------------------------------------------------------------------|
| `vw_contas_corrente`                             | Query em `contas_correntes` + `raw ->> 'cFluxoCaixa'`              |
| `contas_a_pagar_e_receber_mescladas_expandida`   | `contas_pagar` ∪ `contas_receber` + `jsonb_array_elements(raw -> 'categorias')` |
| `contas_a_pagar_e_receber_distribuicao_expandida`| Idem + `jsonb_array_elements(raw -> 'distribuicao')`               |
| `movimentos_expandido` / `vw_movimentos_expandido` | `movimentos_financeiros` + `raw -> 'detalhes'` e `raw -> 'resumo'` |
| `vw_extrato_expandido`                           | Tabela `extrato` diretamente                                        |

---

## Mapeamento Coluna a Coluna

### Extrato

| Campo Antigo (dados_json)                                  | Campo Novo                                        | Tipo    |
|------------------------------------------------------------|---------------------------------------------------|---------|
| `dados_json -> 'extrato' ->> 'nCodCC'`                    | `extrato.codigo_conta_corrente`                   | BIGINT  |
| `mov.value ->> 'dDataLancamento'`                          | `extrato.data_lancamento`                         | DATE    |
| `mov.value ->> 'nValorDocumento'`                          | `extrato.valor`                                   | NUMERIC |
| `mov.value ->> 'cCodCategoria'`                            | `extrato.raw ->> 'cCodCategoria'`                 | TEXT    |
| `mov.value ->> 'nCodCliente'`                              | `extrato.raw ->> 'nCodCliente'`                   | TEXT    |
| `mov.value ->> 'nCodLancamento'`                           | `extrato.raw ->> 'nCodLancamento'`                | TEXT    |
| `mov.value ->> 'cSituacao' = 'Previsto'`                   | `extrato.raw ->> 'cSituacao' = 'Previsto'`        | TEXT    |
| `dados_json -> 'extrato' ->> 'cFluxoCaixa' = 'S'`         | `contas_correntes.raw ->> 'cFluxoCaixa' = 'S'`    | TEXT    |

### Movimentos Financeiros

| Campo Antigo (dados_json)                                  | Campo Novo                                                    | Tipo    |
|------------------------------------------------------------|---------------------------------------------------------------|---------|
| `dados_json -> 'detalhes' ->> 'nCodCC'`                   | `movimentos_financeiros.codigo_conta_corrente`                | BIGINT  |
| `dados_json -> 'detalhes' ->> 'nCodCliente'`              | `movimentos_financeiros.raw -> 'detalhes' ->> 'nCodCliente'`  | TEXT    |
| `dados_json -> 'detalhes' ->> 'nCodTitulo'`               | `movimentos_financeiros.codigo_lancamento`                    | BIGINT  |
| `dados_json -> 'detalhes' ->> 'cGrupo'`                   | `movimentos_financeiros.raw -> 'detalhes' ->> 'cGrupo'`       | TEXT    |
| `dados_json -> 'detalhes' ->> 'dDtPagamento'`             | `movimentos_financeiros.raw -> 'detalhes' ->> 'dDtPagamento'` | TEXT    |
| `dados_json -> 'detalhes' ->> 'dDtPrevisao'`              | `movimentos_financeiros.raw -> 'detalhes' ->> 'dDtPrevisao'`  | TEXT    |
| `dados_json -> 'detalhes' ->> 'cCodCateg'`                | `movimentos_financeiros.codigo_categoria`                     | TEXT    |
| `dados_json -> 'detalhes' ->> 'cStatus'`                  | `movimentos_financeiros.raw -> 'detalhes' ->> 'cStatus'`      | TEXT    |
| `dados_json -> 'resumo' ->> 'nValLiquido'`                | `movimentos_financeiros.raw -> 'resumo' ->> 'nValLiquido'`    | TEXT    |
| `dados_json -> 'resumo' ->> 'nValPago'`                   | `movimentos_financeiros.raw -> 'resumo' ->> 'nValPago'`       | TEXT    |
| `departamentos[].cCodDepartamento`                         | `movimentos_financeiros.raw -> 'departamentos'` (array JSONB) | JSONB   |
| `departamentos[].nDistrPercentual`                         | `movimentos_financeiros.raw -> 'departamentos' -> 'nDistrPercentual'` | NUMERIC |
| `departamentos[].nDistrValor`                              | `movimentos_financeiros.raw -> 'departamentos' -> 'nDistrValor'` | NUMERIC |

### Contas a Pagar / Receber

| Campo Antigo (dados_json)                         | Campo Novo                                              | Tipo    |
|---------------------------------------------------|---------------------------------------------------------|---------|
| `dados_json ->> 'codigo_lancamento_omie'`         | `contas_pagar.codigo_lancamento` / `contas_receber.codigo_lancamento` | BIGINT |
| `dados_json ->> 'valor_documento'`                | `contas_pagar.valor_documento` / `contas_receber.valor_documento` | NUMERIC |
| `dados_json -> 'categorias'[]`                    | `contas_pagar.raw -> 'categorias'` (jsonb_array_elements) | JSONB   |
| `categoria.value ->> 'codigo_categoria'`          | `raw -> 'categorias' -> elem ->> 'codigo_categoria'`    | TEXT    |
| `categoria.value ->> 'percentual'`                | `raw -> 'categorias' -> elem ->> 'percentual'`          | NUMERIC |
| `categoria.value ->> 'valor'`                     | `raw -> 'categorias' -> elem ->> 'valor'`               | NUMERIC |
| `dados_json -> 'distribuicao'[]`                  | `contas_pagar.raw -> 'distribuicao'` (jsonb_array_elements) | JSONB |
| `distribuicao.value ->> 'cCodDep'`                | `raw -> 'distribuicao' -> elem ->> 'cCodDep'`           | TEXT    |
| `distribuicao.value ->> 'nValDep'`                | `raw -> 'distribuicao' -> elem ->> 'nValDep'`           | NUMERIC |
| `distribuicao.value ->> 'nPerDep'`                | `raw -> 'distribuicao' -> elem ->> 'nPerDep'`           | NUMERIC |

### Categorias / Departamentos / Clientes

| Campo Antigo (dados_json)                         | Campo Novo                          | Tipo  |
|---------------------------------------------------|-------------------------------------|-------|
| `categorias.dados_json ->> 'codigo'`              | `categorias.codigo`                 | TEXT  |
| `categorias.dados_json ->> 'descricao'`           | `categorias.descricao`              | TEXT  |
| `departamentos.dados_json ->> 'codigo'`           | `departamentos.codigo`              | TEXT  |
| `departamentos.dados_json ->> 'descricao'`        | `departamentos.descricao`           | TEXT  |
| `clientes.dados_json ->> 'nome_fantasia'`         | `clientes.nome_fantasia`            | TEXT  |
| `clientes.dados_json ->> 'codigo_cliente_omie'`   | `clientes.codigo_cliente_omie`      | BIGINT|

---

## Campos Exclusivos do raw (sem coluna própria)

Estes campos existem apenas dentro da coluna `raw` (JSONB) e são acessados via operadores JSONB:

| Tabela                    | Campo no raw                          | Uso na view gerencial              |
|---------------------------|---------------------------------------|------------------------------------|
| `contas_correntes`        | `raw ->> 'cFluxoCaixa'`              | Filtro: apenas contas de fluxo caixa |
| `extrato`                 | `raw ->> 'cSituacao'`                | Filtro: apenas 'Previsto'          |
| `extrato`                 | `raw ->> 'cCodCategoria'`            | Categoria do lançamento            |
| `extrato`                 | `raw ->> 'nCodCliente'`              | Código do cliente                  |
| `movimentos_financeiros`  | `raw -> 'detalhes' ->> 'cGrupo'`    | Filtro: CONTA_CORRENTE_REC/PAG     |
| `movimentos_financeiros`  | `raw -> 'detalhes' ->> 'cStatus'`   | Status: RECEBIDO / PAGO            |
| `movimentos_financeiros`  | `raw -> 'resumo' ->> 'nValLiquido'` | Valor líquido do movimento         |
| `movimentos_financeiros`  | `raw -> 'resumo' ->> 'nValPago'`    | Valor pago do movimento            |
| `movimentos_financeiros`  | `raw -> 'departamentos'`             | Array de distribuição por depto    |
| `contas_pagar`            | `raw -> 'categorias'`                | Array de categorias com percentual |
| `contas_pagar`            | `raw -> 'distribuicao'`              | Array de distribuição por depto    |
| `contas_receber`          | `raw -> 'categorias'`                | Array de categorias com percentual |
| `contas_receber`          | `raw -> 'distribuicao'`              | Array de distribuição por depto    |

---

## Notas de Compatibilidade

1. **Datas**: Na estrutura antiga, datas eram strings `DD-MM-YYYY` no JSONB.
   Na nova, colunas tipadas são `DATE` (ISO). Campos ainda em `raw` continuam como string `DD/MM/YYYY`.
   Usar `TO_DATE(campo, 'DD/MM/YYYY')` ao extrair do raw.

2. **empresa_id**: Todas as tabelas do tenant possuem `empresa_id UUID` (adicionado na migration 000021).
   A view gerencial deve sempre filtrar por `empresa_id` para garantir isolamento multi-tenant.

3. **extrato**: Não possui UNIQUE — é recriado a cada sync via DELETE+INSERT por conta corrente.
   A view gerencial deve tratar isso sem joins problemáticos.

4. **raw é sempre o JSON bruto**: Mesmo que o struct Go não mapeie todos os campos,
   o `raw` contém o JSON completo retornado pela API Omie, incluindo arrays `categorias` e `distribuicao`.
