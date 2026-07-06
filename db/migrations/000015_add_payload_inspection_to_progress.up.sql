-- Migração 000015: Adiciona colunas para inspeção de payload em sync_job_progress
ALTER TABLE _etl.sync_job_progress
    ADD COLUMN IF NOT EXISTS ultimo_payload  JSONB,
    ADD COLUMN IF NOT EXISTS ultimo_response JSONB,
    ADD COLUMN IF NOT EXISTS erro_payload    JSONB,
    ADD COLUMN IF NOT EXISTS erro_response   TEXT;
