-- Dá a todo usuário do ambiente a visão do grupo Demo, como 'viewer'
-- (vê dashboard; não gerencia empresas/usuários nem dispara sync).
-- Vínculos já existentes NÃO são rebaixados (ON CONFLICT DO NOTHING).
-- Usuários criados depois ganham o vínculo pelo trigger abaixo.
-- Reverter: rollback-total (remove grupo, vínculos, trigger e função).
BEGIN;
SELECT set_config('demo.grupo', :'demo_grupo', true);
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM _etl.grupos
                 WHERE id = current_setting('demo.grupo')::uuid AND slug = 'demo') THEN
    RAISE EXCEPTION 'grupo % não é o grupo demo — abortando', current_setting('demo.grupo');
  END IF;
END $$;

INSERT INTO _etl.usuario_grupos (usuario_id, grupo_id, role)
SELECT u.id, :'demo_grupo'::uuid, 'viewer'
FROM _etl.usuarios u
WHERE u.deleted_at IS NULL
ON CONFLICT (usuario_id, grupo_id) DO NOTHING;

CREATE OR REPLACE FUNCTION _etl.demo_vincular_novo_usuario() RETURNS trigger
LANGUAGE plpgsql AS $f$
BEGIN
  INSERT INTO _etl.usuario_grupos (usuario_id, grupo_id, role)
  SELECT NEW.id, g.id, 'viewer'
  FROM _etl.grupos g
  WHERE g.slug = 'demo' AND g.deleted_at IS NULL
  ON CONFLICT (usuario_id, grupo_id) DO NOTHING;
  RETURN NEW;
END $f$;

DROP TRIGGER IF EXISTS trg_demo_vincular_novo_usuario ON _etl.usuarios;
CREATE TRIGGER trg_demo_vincular_novo_usuario
  AFTER INSERT ON _etl.usuarios
  FOR EACH ROW EXECUTE FUNCTION _etl.demo_vincular_novo_usuario();
COMMIT;

SELECT u.email, u.role AS papel_global, ug.role AS papel_no_demo, u.ativo
FROM _etl.usuarios u
JOIN _etl.usuario_grupos ug ON ug.usuario_id = u.id AND ug.grupo_id = :'demo_grupo'::uuid
WHERE u.deleted_at IS NULL
ORDER BY u.email;
SELECT COUNT(*) AS usuarios_sem_demo FROM _etl.usuarios u
WHERE u.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM _etl.usuario_grupos ug
                  WHERE ug.usuario_id = u.id AND ug.grupo_id = :'demo_grupo'::uuid);
