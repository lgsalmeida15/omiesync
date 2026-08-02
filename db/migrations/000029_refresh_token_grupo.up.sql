-- Migration 000029: persiste o grupo selecionado junto ao refresh token.
--
-- Problema: /auth/refresh recebe apenas o refresh token opaco, sem as claims do
-- access token que expirou. Sem saber qual grupo estava ativo, o Refresh reconstruia
-- o JWT com usuarios.grupo_id -- o padrao do cadastro, nao o grupo escolhido.
--
-- Efeito para o usuario multi-grupo: apos ~15 minutos o access token expira, o
-- frontend renova em silencio, e o grupo volta sozinho para o padrao. A pessoa segue
-- olhando dados de outro grupo sem ter trocado nada.
--
-- O grupo passa a viajar com o refresh token, que ja e rotacionado a cada renovacao.

ALTER TABLE _etl.refresh_tokens
    ADD COLUMN IF NOT EXISTS grupo_id UUID REFERENCES _etl.grupos(id) ON DELETE SET NULL;

-- As sessoes em curso sao encerradas, nao migradas.
--
-- Nao ha como recuperar o grupo ativo delas: essa informacao nunca foi persistida.
-- Herdar usuarios.grupo_id parece conservador, mas perpetuaria exatamente o bug --
-- quem estivesse trabalhando no grupo B seria devolvido ao grupo A na proxima
-- renovacao, uma ultima vez.
--
-- Como o efeito e exibir dados de um cliente a quem acredita estar em outro, o
-- re-login e o custo aceitavel. Quem tiver access token valido segue ate 15 minutos;
-- depois disso precisa autenticar de novo, e a sessao nova ja nasce com o grupo
-- corretamente registrado.
UPDATE _etl.refresh_tokens
SET revoked = true
WHERE revoked = false;
