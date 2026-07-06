-- Migration 000007: Tabela de auditoria de requests
CREATE TABLE IF NOT EXISTS _etl.audit_logs (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id     TEXT,
    user_id        UUID,
    user_email     TEXT,
    role           TEXT,
    method         TEXT        NOT NULL,
    path           TEXT        NOT NULL,
    query_params   TEXT,
    status_code    INT,
    request_body   TEXT,
    response_body  TEXT,
    ip_address     TEXT,
    user_agent     TEXT,
    duration_ms    BIGINT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id    ON _etl.audit_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON _etl.audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_path       ON _etl.audit_logs (path);
