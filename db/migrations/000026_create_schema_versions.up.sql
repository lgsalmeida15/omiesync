-- Migration 000026: tabela de controle de versão de schemas tenant.
-- Usada pelo Provisioner para evitar re-execução de DDL em cada job de sync.
-- O worker só chama ProvisionSchema quando schema_versions.version < CurrentSchemaVersion.

CREATE TABLE IF NOT EXISTS _etl.schema_versions (
    schema_name TEXT        PRIMARY KEY,
    version     INTEGER     NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
