-- db/migrations/000018_add_heartbeat_to_sync_jobs.up.sql
ALTER TABLE _etl.sync_jobs
    ADD COLUMN IF NOT EXISTS ultimo_heartbeat_at TIMESTAMPTZ;
