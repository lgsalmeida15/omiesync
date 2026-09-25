-- Remove só os DADOS das empresas Demo. Grupo, empresas e usuários continuam.
BEGIN;
CREATE TEMP TABLE demo_emp ON COMMIT DROP AS
  SELECT id FROM _etl.empresas WHERE grupo_id = :'demo_grupo';
DELETE FROM grupo_demo.clientes               WHERE empresa_id IN (SELECT id FROM demo_emp);
DELETE FROM grupo_demo.categorias             WHERE empresa_id IN (SELECT id FROM demo_emp);
DELETE FROM grupo_demo.departamentos          WHERE empresa_id IN (SELECT id FROM demo_emp);
DELETE FROM grupo_demo.contas_correntes       WHERE empresa_id IN (SELECT id FROM demo_emp);
DELETE FROM grupo_demo.contas_pagar           WHERE empresa_id IN (SELECT id FROM demo_emp);
DELETE FROM grupo_demo.contas_receber         WHERE empresa_id IN (SELECT id FROM demo_emp);
DELETE FROM grupo_demo.movimentos_financeiros WHERE empresa_id IN (SELECT id FROM demo_emp);
DELETE FROM grupo_demo.extrato                WHERE empresa_id IN (SELECT id FROM demo_emp);
DELETE FROM grupo_demo.ordens_servico         WHERE empresa_id IN (SELECT id FROM demo_emp);
DELETE FROM grupo_demo.projetos               WHERE empresa_id IN (SELECT id FROM demo_emp);
COMMIT;
REFRESH MATERIALIZED VIEW grupo_demo.matvw_gerencial_ano_corrente;
REFRESH MATERIALIZED VIEW grupo_demo.matvw_gerencial_historico;
DROP SCHEMA IF EXISTS demo_stage CASCADE;
SELECT 'rollback-dados ok' AS resultado;
