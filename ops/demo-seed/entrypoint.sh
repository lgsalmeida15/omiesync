#!/bin/bash
# Popula o grupo Demo (HML) com cópia anonimizada de um grupo real (PROD).
#
# Env:
#   MODE         seed | rollback-dados | rollback-total
#   HML_URL      banco de destino (único que recebe escrita)
#   PROD_URL     banco de origem — só leitura (sessão read-only forçada)
#   SRC_SCHEMA   schema do grupo de origem em PROD
#   DEMO_GRUPO   uuid do grupo Demo no HML
#   MAP          pares origem:destino de empresa_id, separados por vírgula
#   FATOR        multiplicador aplicado a todo valor monetário (default 0.63)
#
# Nenhum dado de cliente nem credencial fica neste repositório: tudo vem por env.
set -euo pipefail
# Falha também não pode sair: o Coolify reiniciaria o container em loop.
trap 'echo DEMO_SEED_FAILED; sleep 86400' ERR

FATOR="${FATOR:-0.63}"
MAP_VALUES=""
SRC_IN=""
for par in ${MAP//,/ }; do
  src="${par%%:*}"; dst="${par##*:}"
  MAP_VALUES+="${MAP_VALUES:+,}('$src'::uuid,'$dst'::uuid)"
  SRC_IN+="${SRC_IN:+,}'$src'"
done

hml() { psql "$HML_URL" -X -q -v ON_ERROR_STOP=1 -v map_values="$MAP_VALUES" -v fator="$FATOR" -v demo_grupo="$DEMO_GRUPO" "$@"; }
prod() { PGOPTIONS="-c default_transaction_read_only=on" psql "$PROD_URL" -X -q -v ON_ERROR_STOP=1 "$@"; }

declare -A COLS=(
  [clientes]="id,empresa_id,codigo_cliente_omie,razao_social,nome_fantasia,cnpj_cpf,email,telefone1_ddd,telefone1_numero,endereco,cidade,estado,cep,ativo,data_alteracao,raw,synced_at"
  [categorias]="id,empresa_id,codigo,descricao,id_conta_corrente,raw,synced_at"
  [departamentos]="id,empresa_id,codigo,descricao,inativo,raw,synced_at"
  [contas_correntes]="id,empresa_id,codigo_conta_corrente,descricao,tipo,saldo_inicial,fluxo_caixa,raw,synced_at"
  [contas_pagar]="id,empresa_id,codigo_lancamento,data_vencimento,data_previsao,data_pagamento,valor_documento,valor_pago,status_titulo,codigo_cliente,codigo_categoria,observacao,raw,synced_at"
  [contas_receber]="id,empresa_id,codigo_lancamento,data_vencimento,data_previsao,data_recebimento,valor_documento,valor_recebido,status_titulo,codigo_cliente,codigo_categoria,observacao,raw,synced_at"
  [movimentos_financeiros]="id,empresa_id,codigo_titulo,numero_titulo,codigo_mov_cc,codigo_mov_cc_repet,codigo_titrepet,data_lancamento,valor,tipo,codigo_conta_corrente,codigo_categoria,raw,synced_at"
  [extrato]="id,empresa_id,codigo_lancamento,data_lancamento,valor,tipo_lancamento,codigo_conta_corrente,descricao,fluxo_caixa,raw,synced_at"
)

echo "== demo-seed MODE=$MODE FATOR=$FATOR =="

case "$MODE" in
  seed)
    hml -f /sql/00_stage.sql
    for t in "${!COLS[@]}"; do
      c="${COLS[$t]}"
      echo "-- copiando $t"
      prod -c "\copy (SELECT $c FROM \"$SRC_SCHEMA\".$t WHERE empresa_id IN ($SRC_IN)) TO STDOUT" \
        | hml -c "\copy demo_stage.$t ($c) FROM STDIN"
    done
    hml -f /sql/01_transform.sql
    hml -f /sql/02_refresh.sql
    echo "-- conferência: totais da view ano corrente, origem x demo"
    prod -c "SELECT receita_despesa, ROUND(SUM(valor_final) * $FATOR, 2) AS origem_x_fator, COUNT(*) AS linhas
             FROM \"$SRC_SCHEMA\".matvw_gerencial_ano_corrente WHERE empresa_id IN ($SRC_IN) GROUP BY 1 ORDER BY 1"
    hml -c "SELECT receita_despesa, ROUND(SUM(valor_final), 2) AS demo, COUNT(*) AS linhas
            FROM grupo_demo.matvw_gerencial_ano_corrente GROUP BY 1 ORDER BY 1"
    ;;
  rollback-dados)
    hml -f /sql/90_rollback_dados.sql
    ;;
  rollback-total)
    hml -f /sql/91_rollback_total.sql
    ;;
  *)
    echo "MODE inválido: $MODE"; exit 1 ;;
esac

echo "DEMO_SEED_DONE"
# Mantém o container de pé: sair faria o Coolify reiniciar e rodar tudo de novo.
sleep 86400
