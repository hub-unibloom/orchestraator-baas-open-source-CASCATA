CREATE SCHEMA IF NOT EXISTS system;
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA system;

-- Identity Distinction (Crucial for auditing)
DO $$ BEGIN
    CREATE TYPE system.member_type AS ENUM ('MEMBER', 'AGENT');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE system.member_role AS ENUM ('worner', 'manager', 'developer', 'analyst', 'agent');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;


-- Registry for all projects/tenants
CREATE TABLE IF NOT EXISTS system.projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    db_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    custom_domain TEXT UNIQUE, -- e.g. api.clientapp.com
    default_domain TEXT UNIQUE, -- e.g. clientapp.cascata.io
    service_key TEXT NOT NULL,
    anon_key TEXT NOT NULL,
    jwt_secret TEXT NOT NULL,
    secondary_secret_hash TEXT, -- Extra security layer (Phase 24: Agency mode)
    region TEXT DEFAULT 'global',
    timezone TEXT DEFAULT 'UTC',
    max_users INTEGER DEFAULT 100,
    max_conns INTEGER DEFAULT 10,
    max_storage_mb BIGINT DEFAULT 1024,
    max_db_weight_mb BIGINT DEFAULT 1024,
    log_retention_days INTEGER DEFAULT 30,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_projects_custom_domain ON system.projects (custom_domain);
CREATE INDEX IF NOT EXISTS idx_projects_slug ON system.projects (slug);

-- AI Architect Sessions
CREATE TABLE IF NOT EXISTS system.ai_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_slug TEXT NOT NULL,
    user_id UUID NOT NULL,
    title TEXT DEFAULT 'Nova conversa',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- AI Architect Messages
CREATE TABLE IF NOT EXISTS system.ai_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES system.ai_sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL, -- user, assistant, system
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Registry for Storage Objects (The VFS Index)
CREATE TABLE IF NOT EXISTS system.storage_objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_slug TEXT NOT NULL,
    bucket TEXT NOT NULL DEFAULT 'default',
    name TEXT NOT NULL,
    parent_path TEXT NOT NULL DEFAULT '',
    full_path TEXT NOT NULL,
    is_folder BOOLEAN DEFAULT false,
    size BIGINT DEFAULT 0,
    mime_type TEXT,
    provider TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_slug, bucket, full_path)
);


-- Webhook Receivers (Inbound Gateway)
CREATE TABLE IF NOT EXISTS system.webhook_receivers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_slug TEXT NOT NULL,
    name TEXT NOT NULL,
    path_slug TEXT NOT NULL,
    auth_method TEXT DEFAULT 'none', -- none, hmac_sha256
    secret_key TEXT,
    target_type TEXT NOT NULL, -- AUTOMATION, TABLE, TOPIC
    target_id TEXT, -- ID of the automation or table name
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_slug, path_slug)
);

-- Automation & Webhook Execution Logs
CREATE TABLE IF NOT EXISTS system.automation_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_slug TEXT NOT NULL,
    trigger_type TEXT NOT NULL, -- WEBHOOK, CRON, EVENT, MANUAL
    trigger_id TEXT,
    status TEXT NOT NULL, -- success, failure, running
    execution_time_ms INTEGER,
    input_payload JSONB,
    output_result JSONB,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_storage_parent ON system.storage_objects (project_slug, bucket, parent_path);
CREATE INDEX IF NOT EXISTS idx_automation_project ON system.automation_runs (project_slug, created_at DESC);

-- Notification Rules (Neural Pulse)
CREATE TABLE IF NOT EXISTS system.notification_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_slug TEXT NOT NULL,
    name TEXT NOT NULL,
    trigger_table TEXT NOT NULL,
    trigger_event TEXT NOT NULL DEFAULT 'ALL', -- ALL, INSERT, UPDATE
    recipient_column TEXT NOT NULL, -- e.g. user_id
    title_template TEXT NOT NULL,
    body_template TEXT NOT NULL,
    data_payload JSONB DEFAULT '{}',
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Automations (The Blueprint)
CREATE TABLE IF NOT EXISTS system.automations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_slug TEXT NOT NULL,
    name TEXT NOT NULL,
    trigger_type TEXT NOT NULL, -- WEBHOOK, CRON, API_INTERCEPT
    trigger_config JSONB DEFAULT '{}',
    nodes JSONB NOT NULL, -- The graph structure
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

    -- Project Environment Variables (Phase 10B)
    CREATE TABLE IF NOT EXISTS system.project_envs (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        project_slug TEXT NOT NULL,
        key TEXT NOT NULL,
        value TEXT NOT NULL, -- Plain value if public, Encrypted value (cascata:base64) if secret
        is_secret BOOLEAN DEFAULT false,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(project_slug, key)
    );

-- System Members (The highest level operators starting with the Worner)
CREATE TABLE IF NOT EXISTS system.members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL, 
    role system.member_role NOT NULL DEFAULT 'worner',
    type system.member_type NOT NULL DEFAULT 'MEMBER',
    mfa_enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Central Audit Ledger (Blockchain-style Immutable Log for Dashboard/Management actions - Phase 19)
CREATE TABLE IF NOT EXISTS system.audit_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_slug TEXT, -- NULL if global
    operation TEXT NOT NULL, -- e.g., 'CREATE_PROJECT', 'DELETE_USER'
    identity_id TEXT NOT NULL, -- Who did it (MemberID, ResidentID, AgentID)
    identity_type TEXT NOT NULL, -- MEMBER, RESIDENT, AGENT
    table_name TEXT, -- Optional contextual table
    payload JSONB DEFAULT '{}', -- Diff or metadata
    prev_hash TEXT, -- Chain pointer
    entry_hash TEXT NOT NULL UNIQUE, -- Cryptographic signature of this record
    signature TEXT, -- Optional Native Security signature (Phase 24)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_project_slug ON system.audit_ledger (project_slug);
CREATE INDEX IF NOT EXISTS idx_audit_identity ON system.audit_ledger (identity_id, identity_type);
CREATE INDEX IF NOT EXISTS idx_audit_created ON system.audit_ledger (created_at DESC);
