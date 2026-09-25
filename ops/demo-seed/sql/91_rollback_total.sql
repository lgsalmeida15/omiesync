-- Remove o grupo Demo inteiro: schema do tenant, empresas, vínculos de usuário,
-- config/conversas de IA. Tudo que referencia _etl.grupos/_etl.empresas é
-- ON DELETE CASCADE. Trava de segurança: só age se o grupo for o slug 'demo'.
BEGIN;
SELECT set_config('demo.grupo', :'demo_grupo', true);
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM _etl.grupos
                 WHERE id = current_setting('demo.grupo')::uuid AND slug = 'demo') THEN
    RAISE EXCEPTION 'grupo % não é o grupo demo — abortando', current_setting('demo.grupo');
  END IF;
END $$;
DROP SCHEMA IF EXISTS grupo_demo CASCADE;
DROP SCHEMA IF EXISTS demo_stage CASCADE;
DELETE FROM _etl.schema_versions WHERE schema_name = 'grupo_demo';
DELETE FROM _etl.grupos WHERE id = :'demo_grupo' AND slug = 'demo';
COMMIT;
SELECT 'rollback-total ok' AS resultado;
