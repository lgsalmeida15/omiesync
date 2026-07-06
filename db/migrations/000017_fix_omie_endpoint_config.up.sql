-- Categorias: não suporta filtrar_por_data_de → forçar ignorar_delta
UPDATE _etl.omie_endpoint_config
SET ignorar_delta = true,
    notas         = 'API não suporta filtrar_por_data_de; sempre busca tudo',
    updated_at    = NOW()
WHERE modulo = 'categorias';

-- Departamentos: corrige array_field (resposta usa "departamentos", não "DepartamentoCadastro")
--                e activa ignorar_delta pelo mesmo motivo que categorias
UPDATE _etl.omie_endpoint_config
SET array_field   = 'departamentos',
    ignorar_delta = true,
    notas         = 'array_field corrigido para "departamentos"; API não suporta filtrar_por_data_de',
    updated_at    = NOW()
WHERE modulo = 'departamentos';
