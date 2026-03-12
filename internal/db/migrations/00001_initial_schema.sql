-- +goose Up

-- Enable pgcrypto for gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================
-- Users
-- ============================================================
CREATE TABLE users (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    email       TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL DEFAULT '',
    avatar_url  TEXT NOT NULL DEFAULT '',
    issuer_sub  TEXT NOT NULL DEFAULT '',  -- OIDC subject claim
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_issuer_sub ON users (issuer_sub) WHERE issuer_sub != '';

-- ============================================================
-- Teams
-- ============================================================
CREATE TABLE teams (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE team_members (
    team_id    TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'member',  -- owner | admin | member
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, user_id)
);

-- ============================================================
-- Projects
-- ============================================================
CREATE TABLE projects (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    team_id     TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team_id, slug)
);

CREATE TABLE memberships (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'developer',  -- admin | project-owner | developer | viewer
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, project_id)
);

-- ============================================================
-- Resource Types (catalog)
-- ============================================================
CREATE TABLE resource_types (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name         TEXT NOT NULL,
    slug         TEXT NOT NULL UNIQUE,
    category     TEXT NOT NULL,  -- database | cache | storage | messaging | network | secret
    description  TEXT NOT NULL DEFAULT '',
    default_spec JSONB NOT NULL DEFAULT '{}',
    spec_schema  JSONB NOT NULL DEFAULT '{}',
    enabled      BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Resources (provisioned instances)
-- ============================================================
CREATE TABLE resources (
    id               TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id       TEXT NOT NULL REFERENCES projects(id) ON DELETE RESTRICT,
    resource_type_id TEXT NOT NULL REFERENCES resource_types(id) ON DELETE RESTRICT,
    name             TEXT NOT NULL,
    slug             TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'requested',
    spec             JSONB NOT NULL DEFAULT '{}',
    provider_ref     TEXT NOT NULL DEFAULT '',
    status_message   TEXT NOT NULL DEFAULT '',
    requested_by     TEXT NOT NULL REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, slug)
);

CREATE INDEX idx_resources_project_id ON resources (project_id);
CREATE INDEX idx_resources_status ON resources (status);

-- ============================================================
-- Provisioning Events (resource audit trail)
-- ============================================================
CREATE TABLE provisioning_events (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    resource_id TEXT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,  -- status_change | spec_change | error
    old_status  TEXT NOT NULL DEFAULT '',
    new_status  TEXT NOT NULL DEFAULT '',
    message     TEXT NOT NULL DEFAULT '',
    detail      JSONB NOT NULL DEFAULT '{}',
    actor_id    TEXT NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_provisioning_events_resource_id ON provisioning_events (resource_id);

-- ============================================================
-- Policies
-- ============================================================
CREATE TABLE policies (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    scope       TEXT NOT NULL DEFAULT 'global',  -- global | project
    project_id  TEXT REFERENCES projects(id) ON DELETE CASCADE,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_by  TEXT NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE policy_rules (
    id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    policy_id     TEXT NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    description   TEXT NOT NULL DEFAULT '',
    resource_type TEXT NOT NULL DEFAULT '*',
    attribute     TEXT NOT NULL,
    operator      TEXT NOT NULL,  -- eq | neq | in | nin | lt | gt | lte | gte
    value         TEXT NOT NULL,
    effect        TEXT NOT NULL DEFAULT 'deny',  -- allow | deny | require_approval
    metadata      JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_policy_rules_policy_id ON policy_rules (policy_id);

-- ============================================================
-- Quotas
-- ============================================================
CREATE TABLE quotas (
    id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id    TEXT REFERENCES projects(id) ON DELETE CASCADE,  -- NULL = global default
    resource_type TEXT NOT NULL,
    "limit"       INTEGER NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, resource_type)
);

-- ============================================================
-- Approval Requests
-- ============================================================
CREATE TABLE approval_requests (
    id               TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id       TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    requested_by     TEXT NOT NULL REFERENCES users(id),
    resource_type    TEXT NOT NULL,
    request_payload  JSONB NOT NULL DEFAULT '{}',
    reasons          JSONB NOT NULL DEFAULT '[]',
    matched_policies JSONB NOT NULL DEFAULT '[]',
    status           TEXT NOT NULL DEFAULT 'pending',  -- pending | approved | denied | expired
    reviewed_by      TEXT REFERENCES users(id),
    review_comment   TEXT NOT NULL DEFAULT '',
    reviewed_at      TIMESTAMPTZ,
    expires_at       TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_approval_requests_project_id ON approval_requests (project_id);
CREATE INDEX idx_approval_requests_status ON approval_requests (status);

-- ============================================================
-- Audit Log
-- ============================================================
CREATE TABLE audit_entries (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    actor_id    TEXT NOT NULL REFERENCES users(id),
    project_id  TEXT REFERENCES projects(id) ON DELETE SET NULL,
    action      TEXT NOT NULL,
    target_type TEXT NOT NULL DEFAULT '',
    target_id   TEXT NOT NULL DEFAULT '',
    detail      JSONB NOT NULL DEFAULT '{}',
    allowed     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_entries_actor_id ON audit_entries (actor_id);
CREATE INDEX idx_audit_entries_project_id ON audit_entries (project_id);
CREATE INDEX idx_audit_entries_action ON audit_entries (action);
CREATE INDEX idx_audit_entries_created_at ON audit_entries (created_at);

-- +goose Down

DROP TABLE IF EXISTS audit_entries;
DROP TABLE IF EXISTS approval_requests;
DROP TABLE IF EXISTS quotas;
DROP TABLE IF EXISTS policy_rules;
DROP TABLE IF EXISTS policies;
DROP TABLE IF EXISTS provisioning_events;
DROP TABLE IF EXISTS resources;
DROP TABLE IF EXISTS resource_types;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS users;
