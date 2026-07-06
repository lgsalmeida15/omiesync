-- Migration 000001: Cria o schema _etl e a extensão pgcrypto
-- UP
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE SCHEMA IF NOT EXISTS _etl;
