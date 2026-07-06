-- movimentos_financeiros agora gerencia datas internamente (dDtIncDe/dDtAltDe),
-- não depende mais do flag ignorar_delta do worker.
UPDATE _etl.omie_endpoint_config
SET ignorar_delta = false,
    notas         = 'Paginação especial nPagina/nRegPorPagina; filtros via dDtIncDe/dDtIncAte/dDtAltDe/dDtAltAte',
    updated_at    = NOW()
WHERE modulo = 'movimentos_financeiros';
