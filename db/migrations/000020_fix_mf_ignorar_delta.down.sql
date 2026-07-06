UPDATE _etl.omie_endpoint_config
SET ignorar_delta = true,
    notas         = 'Paginação especial: nPagina/nRegPorPagina; sempre busca tudo',
    updated_at    = NOW()
WHERE modulo = 'movimentos_financeiros';
