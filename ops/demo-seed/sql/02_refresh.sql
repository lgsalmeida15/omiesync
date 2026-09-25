-- Primeira carga: as views nascem WITH NO DATA, então o refresh não pode ser CONCURRENTLY.
REFRESH MATERIALIZED VIEW grupo_demo.matvw_gerencial_ano_corrente;
REFRESH MATERIALIZED VIEW grupo_demo.matvw_gerencial_historico;
DROP SCHEMA IF EXISTS demo_stage CASCADE;
