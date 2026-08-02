-- Reverte a 000029. Usuario multi-grupo volta a perder o grupo selecionado a cada
-- renovacao silenciosa do access token.

ALTER TABLE _etl.refresh_tokens DROP COLUMN IF EXISTS grupo_id;
