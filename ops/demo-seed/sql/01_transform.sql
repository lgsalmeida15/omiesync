-- Transforma a cópia crua (demo_stage) em dados de demonstração no grupo_demo.
--
-- Regras:
--   * empresa_id da origem é trocado pelo da empresa Demo correspondente (MAP);
--   * nomes de clientes/fornecedores viram nomes fictícios determinísticos, e
--     contatos/documentos/observações são descartados;
--   * todo valor monetário é multiplicado por :fator (esconde o faturamento real
--     sem alterar a forma dos gráficos); percentuais ficam intactos;
--   * o raw é RECONSTRUÍDO por lista branca — só os campos que as views e o
--     dashboard leem sobrevivem. Qualquer outro campo do Omie é descartado.
--
-- Tudo numa transação: ou a demo inteira entra, ou nada muda.
-- Idempotente: apaga os dados Demo existentes antes de inserir.

BEGIN;

SELECT set_config('demo.fator', :'fator', true);

CREATE TEMP TABLE map_emp (src uuid PRIMARY KEY, dst uuid NOT NULL) ON COMMIT DROP;
INSERT INTO map_emp VALUES :map_values;

-- Trava: todo destino tem de ser empresa do grupo Demo.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM map_emp m
             LEFT JOIN _etl.empresas e ON e.id = m.dst
             LEFT JOIN _etl.grupos g   ON g.id = e.grupo_id
             WHERE g.slug IS DISTINCT FROM 'demo') THEN
    RAISE EXCEPTION 'MAP aponta para empresa fora do grupo demo — abortando';
  END IF;
END $$;

-- ── Funções auxiliares (pg_temp: somem no fim da sessão) ─────────────────────

-- Multiplica um valor JSON numérico (número ou string numérica) pelo fator.
CREATE FUNCTION pg_temp.esc(v jsonb) RETURNS jsonb LANGUAGE sql STABLE AS $f$
  SELECT CASE
    WHEN v IS NULL THEN NULL
    WHEN jsonb_typeof(v) = 'number'
      THEN to_jsonb(ROUND((v #>> '{}')::numeric * current_setting('demo.fator')::numeric, 2))
    WHEN jsonb_typeof(v) = 'string' AND (v #>> '{}') ~ '^-?[0-9]+(\.[0-9]+)?$'
      THEN to_jsonb(ROUND((v #>> '{}')::numeric * current_setting('demo.fator')::numeric, 2))
    ELSE v END
$f$;

-- Mantém só as chaves da lista branca; escala as chaves monetárias.
CREATE FUNCTION pg_temp.pick(obj jsonb, manter text[], escalar text[]) RETURNS jsonb LANGUAGE sql STABLE AS $f$
  SELECT CASE WHEN jsonb_typeof(obj) = 'object' THEN
    (SELECT COALESCE(jsonb_object_agg(key, CASE WHEN key = ANY(escalar) THEN pg_temp.esc(value) ELSE value END), '{}'::jsonb)
       FROM jsonb_each(obj)
      WHERE key = ANY(manter) OR key = ANY(escalar))
  END
$f$;

-- Escala todo número do objeto (bloco "resumo" do movimento: só valores e flags).
CREATE FUNCTION pg_temp.esc_tudo(obj jsonb) RETURNS jsonb LANGUAGE sql STABLE AS $f$
  SELECT CASE WHEN jsonb_typeof(obj) = 'object' THEN
    (SELECT COALESCE(jsonb_object_agg(key, CASE WHEN jsonb_typeof(value) = 'number' THEN pg_temp.esc(value) ELSE value END), '{}'::jsonb)
       FROM jsonb_each(obj))
  END
$f$;

-- Aplica pick() a cada elemento de um array JSON; NULL se não for array.
CREATE FUNCTION pg_temp.pick_arr(arr jsonb, manter text[], escalar text[]) RETURNS jsonb LANGUAGE sql STABLE AS $f$
  SELECT CASE WHEN jsonb_typeof(arr) = 'array' THEN
    (SELECT COALESCE(jsonb_agg(pg_temp.pick(e, manter, escalar) ORDER BY ord), '[]'::jsonb)
       FROM jsonb_array_elements(arr) WITH ORDINALITY AS x(e, ord))
  END
$f$;

CREATE FUNCTION pg_temp.val(v numeric) RETURNS numeric LANGUAGE sql STABLE AS $f$
  SELECT ROUND(v * current_setting('demo.fator')::numeric, 2)
$f$;

-- ── Nomes fictícios de clientes/fornecedores ────────────────────────────────
-- Um nome por código Omie, único e determinístico (mesmo código → mesmo nome em
-- todas as tabelas), atribuído pela posição do código na lista ordenada.
CREATE TEMP TABLE map_cli ON COMMIT DROP AS
WITH codigos AS (
  SELECT codigo_cliente_omie AS cod FROM demo_stage.clientes
  UNION SELECT codigo_cliente FROM demo_stage.contas_pagar
  UNION SELECT codigo_cliente FROM demo_stage.contas_receber
  UNION SELECT NULLIF(raw -> 'detalhes' ->> 'nCodCliente', '')::bigint FROM demo_stage.movimentos_financeiros
  UNION SELECT NULLIF(raw ->> 'nCodCliente', '')::bigint FROM demo_stage.extrato
),
numerados AS (
  SELECT cod, (ROW_NUMBER() OVER (ORDER BY cod) - 1)::int AS n
  FROM codigos WHERE cod IS NOT NULL AND cod <> 0
),
p AS (SELECT ARRAY['Alfa','Atlas','Aurora','Boreal','Cedro','Cristal','Delta','Duna','Eixo','Estrela',
                   'Farol','Fênix','Granito','Horizonte','Ipê','Jade','Jequitibá','Lume','Marco','Meridiano',
                   'Nexo','Nobre','Órbita','Pampa','Pilar','Prisma','Quartzo','Ramos','Rota','Safira',
                   'Serra','Sigma','Solar','Terra','Topázio','Urbe','Vale','Vértice','Zênite','Âncora'] AS a),
r AS (SELECT ARRAY['Comércio','Distribuidora','Serviços','Engenharia','Logística','Transportes','Indústria',
                   'Materiais','Tecnologia','Soluções','Equipamentos','Construções','Manutenção','Hidráulica',
                   'Elétrica','Metalúrgica','Alimentos','Consultoria','Locações','Saneamento','Ambiental',
                   'Máquinas','Suprimentos','Montagens','Instalações','Automação','Energia','Química',
                   'Embalagens','Segurança'] AS a),
s AS (SELECT ARRAY['Ltda','S.A.','do Brasil Ltda','Nacional Ltda','Sul Ltda','Norte Ltda','Paulista Ltda',
                   'Mineira Ltda','Nordeste Ltda','Centro-Oeste Ltda','Atlântico Ltda','Litoral Ltda'] AS a)
SELECT n.cod,
       p.a[n.n % 40 + 1] || ' ' || r.a[(n.n / 40) % 30 + 1] || ' ' || s.a[(n.n / 1200) % 12 + 1]
         || CASE WHEN n.n >= 14400 THEN ' ' || (n.n / 14400 + 1)::text ELSE '' END AS nome
FROM numerados n, p, r, s;
CREATE UNIQUE INDEX ON map_cli (cod);

-- ── Limpa a carga anterior (idempotência) ───────────────────────────────────
DELETE FROM grupo_demo.clientes               WHERE empresa_id IN (SELECT dst FROM map_emp);
DELETE FROM grupo_demo.categorias             WHERE empresa_id IN (SELECT dst FROM map_emp);
DELETE FROM grupo_demo.departamentos          WHERE empresa_id IN (SELECT dst FROM map_emp);
DELETE FROM grupo_demo.contas_correntes       WHERE empresa_id IN (SELECT dst FROM map_emp);
DELETE FROM grupo_demo.contas_pagar           WHERE empresa_id IN (SELECT dst FROM map_emp);
DELETE FROM grupo_demo.contas_receber         WHERE empresa_id IN (SELECT dst FROM map_emp);
DELETE FROM grupo_demo.movimentos_financeiros WHERE empresa_id IN (SELECT dst FROM map_emp);
DELETE FROM grupo_demo.extrato                WHERE empresa_id IN (SELECT dst FROM map_emp);

-- ── Cadastros ───────────────────────────────────────────────────────────────

-- Clientes: só nome fictício + cidade/UF. Documento, e-mail, telefone e endereço saem.
INSERT INTO grupo_demo.clientes
  (empresa_id, codigo_cliente_omie, razao_social, nome_fantasia, cnpj_cpf, email,
   telefone1_ddd, telefone1_numero, endereco, cidade, estado, cep, ativo, data_alteracao, raw, synced_at)
SELECT m.dst, c.codigo_cliente_omie, COALESCE(mc.nome, 'Cliente Demo'), COALESCE(mc.nome, 'Cliente Demo'), '', '',
       NULL, NULL, NULL, c.cidade, c.estado, NULL, c.ativo, c.data_alteracao,
       jsonb_strip_nulls(jsonb_build_object(
         'codigo_cliente_omie', c.codigo_cliente_omie,
         'razao_social', COALESCE(mc.nome, 'Cliente Demo'), 'nome_fantasia', COALESCE(mc.nome, 'Cliente Demo'),
         'cidade', c.cidade, 'estado', c.estado,
         'inativo', CASE WHEN c.ativo THEN 'N' ELSE 'S' END)),
       c.synced_at
FROM demo_stage.clientes c
JOIN map_emp m  ON m.src = c.empresa_id
LEFT JOIN map_cli mc ON mc.cod = c.codigo_cliente_omie;

-- Categorias e departamentos: plano de contas genérico, copiado como está.
INSERT INTO grupo_demo.categorias (empresa_id, codigo, descricao, id_conta_corrente, raw, synced_at)
SELECT m.dst, c.codigo, c.descricao, c.id_conta_corrente, c.raw, c.synced_at
FROM demo_stage.categorias c JOIN map_emp m ON m.src = c.empresa_id;

INSERT INTO grupo_demo.departamentos (empresa_id, codigo, descricao, inativo, raw, synced_at)
SELECT m.dst, d.codigo, d.descricao, d.inativo, d.raw, d.synced_at
FROM demo_stage.departamentos d JOIN map_emp m ON m.src = d.empresa_id;

-- Contas correntes: nome genérico por tipo + letra grega; banco/agência/conta saem do raw.
INSERT INTO grupo_demo.contas_correntes
  (empresa_id, codigo_conta_corrente, descricao, tipo, saldo_inicial, fluxo_caixa, raw, synced_at)
SELECT dst, codigo_conta_corrente, nome, tipo, pg_temp.val(saldo_inicial), fluxo_caixa,
       jsonb_strip_nulls(jsonb_build_object('nCodCC', codigo_conta_corrente, 'descricao', nome,
                                            'tipo_conta_corrente', tipo)),
       synced_at
FROM (
  SELECT m.dst, cc.*,
         CASE UPPER(COALESCE(cc.tipo, ''))
           WHEN 'CX' THEN 'Caixa' WHEN 'CA' THEN 'Aplicação' WHEN 'CP' THEN 'Poupança'
           WHEN 'CR' THEN 'Cartão' WHEN 'AD' THEN 'Adiantamento' ELSE 'Banco' END
         || ' ' ||
         (ARRAY['Alfa','Beta','Gama','Delta','Épsilon','Zeta','Eta','Teta','Iota','Kapa','Lambda','Mi',
                'Ni','Csi','Ômicron','Pi','Rô','Sigma','Tau','Ípsilon','Fi','Qui','Psi','Ômega'])
           [((ROW_NUMBER() OVER (PARTITION BY cc.empresa_id ORDER BY cc.codigo_conta_corrente) - 1) % 24) + 1]
         || CASE WHEN ROW_NUMBER() OVER (PARTITION BY cc.empresa_id ORDER BY cc.codigo_conta_corrente) > 24
                 THEN ' ' || ((ROW_NUMBER() OVER (PARTITION BY cc.empresa_id ORDER BY cc.codigo_conta_corrente) - 1) / 24 + 1)
                 ELSE '' END AS nome
  FROM demo_stage.contas_correntes cc JOIN map_emp m ON m.src = cc.empresa_id
) x;

-- ── Títulos ─────────────────────────────────────────────────────────────────
-- raw: campos lidos pelas views/inadimplência (categorias, distribuicao,
-- id_conta_corrente) + datas/status. Observação, documento fiscal, código de
-- barras e afins saem.

INSERT INTO grupo_demo.contas_pagar
  (empresa_id, codigo_lancamento, data_vencimento, data_previsao, data_pagamento,
   valor_documento, valor_pago, status_titulo, codigo_cliente, codigo_categoria, observacao, raw, synced_at)
SELECT m.dst, t.codigo_lancamento, t.data_vencimento, t.data_previsao, t.data_pagamento,
       pg_temp.val(t.valor_documento), pg_temp.val(t.valor_pago), t.status_titulo, t.codigo_cliente,
       t.codigo_categoria, NULL,
       jsonb_strip_nulls(
         COALESCE(pg_temp.pick(t.raw,
           ARRAY['codigo_lancamento_omie','codigo_cliente_fornecedor','data_vencimento','data_previsao',
                 'data_emissao','data_entrada','status_titulo','codigo_categoria','id_conta_corrente',
                 'numero_parcela','codigo_tipo_documento'],
           ARRAY['valor_documento']), '{}'::jsonb)
         || jsonb_build_object(
           'categorias',   pg_temp.pick_arr(t.raw -> 'categorias',   ARRAY['codigo_categoria','percentual'], ARRAY['valor']),
           'distribuicao', pg_temp.pick_arr(t.raw -> 'distribuicao', ARRAY['cCodDep','cDesDep','nPerDep'],  ARRAY['nValDep']))),
       t.synced_at
FROM demo_stage.contas_pagar t JOIN map_emp m ON m.src = t.empresa_id;

INSERT INTO grupo_demo.contas_receber
  (empresa_id, codigo_lancamento, data_vencimento, data_previsao, data_recebimento,
   valor_documento, valor_recebido, status_titulo, codigo_cliente, codigo_categoria, observacao, raw, synced_at)
SELECT m.dst, t.codigo_lancamento, t.data_vencimento, t.data_previsao, t.data_recebimento,
       pg_temp.val(t.valor_documento), pg_temp.val(t.valor_recebido), t.status_titulo, t.codigo_cliente,
       t.codigo_categoria, NULL,
       jsonb_strip_nulls(
         COALESCE(pg_temp.pick(t.raw,
           ARRAY['codigo_lancamento_omie','codigo_cliente_fornecedor','data_vencimento','data_previsao',
                 'data_emissao','data_entrada','status_titulo','codigo_categoria','id_conta_corrente',
                 'numero_parcela','codigo_tipo_documento'],
           ARRAY['valor_documento']), '{}'::jsonb)
         || jsonb_build_object(
           'categorias',   pg_temp.pick_arr(t.raw -> 'categorias',   ARRAY['codigo_categoria','percentual'], ARRAY['valor']),
           'distribuicao', pg_temp.pick_arr(t.raw -> 'distribuicao', ARRAY['cCodDep','cDesDep','nPerDep'],  ARRAY['nValDep']))),
       t.synced_at
FROM demo_stage.contas_receber t JOIN map_emp m ON m.src = t.empresa_id;

-- ── Movimentos (realizado) ──────────────────────────────────────────────────
INSERT INTO grupo_demo.movimentos_financeiros
  (empresa_id, codigo_titulo, numero_titulo, codigo_mov_cc, codigo_mov_cc_repet, codigo_titrepet,
   data_lancamento, valor, tipo, codigo_conta_corrente, codigo_categoria, raw, synced_at)
SELECT m.dst, t.codigo_titulo, t.numero_titulo, t.codigo_mov_cc, t.codigo_mov_cc_repet, t.codigo_titrepet,
       t.data_lancamento, pg_temp.val(t.valor), t.tipo, t.codigo_conta_corrente, t.codigo_categoria,
       jsonb_strip_nulls(jsonb_build_object(
         'detalhes', pg_temp.pick(t.raw -> 'detalhes',
           ARRAY['nCodTitulo','nCodTitRepet','nCodMovCC','nCodMovCCRepet','dDtRegistro','dDtEmissao',
                 'dDtVenc','dDtPrevisao','dDtPagamento','dDtConcilia','cStatus','cGrupo','nCodCC',
                 'cCodCateg','cNumTitulo','cNatureza','nCodCliente','cTipo','cOrigem','cNumParcela'],
           ARRAY['nValorTitulo','nValorMovCC']),
         'resumo',        pg_temp.esc_tudo(t.raw -> 'resumo'),
         'departamentos', pg_temp.pick_arr(t.raw -> 'departamentos',
                            ARRAY['cCodDepartamento','nDistrPercentual'], ARRAY['nDistrValor']),
         'categorias',    pg_temp.pick_arr(t.raw -> 'categorias',
                            ARRAY['cCodCateg','nDistrPercentual'], ARRAY['nDistrValor']))),
       t.synced_at
FROM demo_stage.movimentos_financeiros t JOIN map_emp m ON m.src = t.empresa_id;

-- ── Extrato (previsto) ──────────────────────────────────────────────────────
-- descricao e cRazCliente/cDesCliente trazem o nome do cliente: viram o fictício.
INSERT INTO grupo_demo.extrato
  (empresa_id, codigo_lancamento, data_lancamento, valor, tipo_lancamento, codigo_conta_corrente,
   descricao, fluxo_caixa, raw, synced_at)
SELECT m.dst, t.codigo_lancamento, t.data_lancamento, pg_temp.val(t.valor), t.tipo_lancamento,
       t.codigo_conta_corrente, COALESCE(mc.nome, 'Lançamento previsto'), t.fluxo_caixa,
       jsonb_strip_nulls(
         COALESCE(pg_temp.pick(t.raw,
           ARRAY['nCodLancamento','nCodCliente','cCodCategoria','cDesCategoria','cSituacao',
                 'dDataLancamento','cTipoDocumento','cNatureza','cOrigem','nCodTitulo'],
           ARRAY['nValorDocumento','nSaldo']), '{}'::jsonb)
         || jsonb_build_object('cRazCliente', mc.nome, 'cDesCliente', mc.nome)),
       t.synced_at
FROM demo_stage.extrato t
JOIN map_emp m ON m.src = t.empresa_id
LEFT JOIN map_cli mc ON mc.cod = NULLIF(t.raw ->> 'nCodCliente', '')::bigint;

-- Garantia extra: nenhuma empresa Demo com agendamento de sync ativo.
UPDATE _etl.sync_control SET ativo = false WHERE empresa_id IN (SELECT dst FROM map_emp);

-- ── Conferência de contagem: origem x destino, por tabela ──────────────────
SELECT t.tabela, t.origem, t.demo, CASE WHEN t.origem = t.demo THEN 'ok' ELSE 'DIVERGE' END AS status
FROM (
  SELECT 'clientes' AS tabela, (SELECT COUNT(*) FROM demo_stage.clientes) AS origem,
         (SELECT COUNT(*) FROM grupo_demo.clientes WHERE empresa_id IN (SELECT dst FROM map_emp)) AS demo
  UNION ALL SELECT 'categorias', (SELECT COUNT(*) FROM demo_stage.categorias),
         (SELECT COUNT(*) FROM grupo_demo.categorias WHERE empresa_id IN (SELECT dst FROM map_emp))
  UNION ALL SELECT 'departamentos', (SELECT COUNT(*) FROM demo_stage.departamentos),
         (SELECT COUNT(*) FROM grupo_demo.departamentos WHERE empresa_id IN (SELECT dst FROM map_emp))
  UNION ALL SELECT 'contas_correntes', (SELECT COUNT(*) FROM demo_stage.contas_correntes),
         (SELECT COUNT(*) FROM grupo_demo.contas_correntes WHERE empresa_id IN (SELECT dst FROM map_emp))
  UNION ALL SELECT 'contas_pagar', (SELECT COUNT(*) FROM demo_stage.contas_pagar),
         (SELECT COUNT(*) FROM grupo_demo.contas_pagar WHERE empresa_id IN (SELECT dst FROM map_emp))
  UNION ALL SELECT 'contas_receber', (SELECT COUNT(*) FROM demo_stage.contas_receber),
         (SELECT COUNT(*) FROM grupo_demo.contas_receber WHERE empresa_id IN (SELECT dst FROM map_emp))
  UNION ALL SELECT 'movimentos_financeiros', (SELECT COUNT(*) FROM demo_stage.movimentos_financeiros),
         (SELECT COUNT(*) FROM grupo_demo.movimentos_financeiros WHERE empresa_id IN (SELECT dst FROM map_emp))
  UNION ALL SELECT 'extrato', (SELECT COUNT(*) FROM demo_stage.extrato),
         (SELECT COUNT(*) FROM grupo_demo.extrato WHERE empresa_id IN (SELECT dst FROM map_emp))
) t;

COMMIT;
