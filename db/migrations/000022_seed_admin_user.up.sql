INSERT INTO _etl.usuarios (nome, email, password, role, status)
VALUES (
  'Leonardo Gabriel',
  'leonardo.gabriel@ejfa.com.br',
  '$2a$10$CNgmtmFGhZRS09QOW9u4Qu.gv9CZ0gAASXkP7bdt3Uu.evyEjh/.2',
  'admin_global',
  'ativo'
)
ON CONFLICT (email) DO NOTHING;
