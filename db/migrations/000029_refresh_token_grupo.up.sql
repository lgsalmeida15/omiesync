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

-- Backfill: tokens ativos herdam o grupo padrao do usuario. Sem isso as sessoes em
-- curso continuariam caindo no comportamento antigo ate expirarem.
UPDATE _etl.refresh_tokens rt
SET grupo_id = u.grupo_id
FROM _etl.usuarios u
WHERE rt.usuario_id = u.id
  AND rt.grupo_id IS NULL
  AND rt.revoked = false;
