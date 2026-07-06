-- Renomear intervalo_min → intervalo_incremental_min
ALTER TABLE _etl.sync_control RENAME COLUMN intervalo_min TO intervalo_incremental_min;

-- Adicionar campos para sync completo
ALTER TABLE _etl.sync_control
  ADD COLUMN IF NOT EXISTS intervalo_full_dias     INT         NOT NULL DEFAULT 7,
  ADD COLUMN IF NOT EXISTS ultimo_full_sync_at     TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS proximo_full_sync_at    TIMESTAMPTZ;

-- Índice para o scheduler full
CREATE INDEX IF NOT EXISTS idx_sync_control_proximo_full
  ON _etl.sync_control (proximo_full_sync_at)
  WHERE ativo = true;
