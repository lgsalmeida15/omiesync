-- db/migrations/000018_add_heartbeat_to_sync_jobs.down.sql
ALTER TABLE _etl.sync_jobs
    DROP COLUMN IF EXISTS ultimo_heartbeat_at;
