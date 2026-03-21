-- Cascata Tenant Apartment Template v1.0.0.0
-- This SQL is executed on the physical tenant DB upon creation.

-- Auth Schema (Identity Sovereignty)
CREATE SCHEMA IF NOT EXISTS auth;

CREATE TABLE IF NOT EXISTS auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE,
    phone TEXT UNIQUE,
    password_hash TEXT,
    raw_app_metadata JSONB DEFAULT '{}',
    raw_user_metadata JSONB DEFAULT '{}',
    is_super_admin BOOLEAN DEFAULT false,
    confirmed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Attach the Security Perimeter Trigger.
CREATE TRIGGER tr_enforce_security_auth_users
    BEFORE INSERT OR UPDATE OR DELETE ON auth.users
    FOR EACH ROW EXECUTE FUNCTION auth.enforce_dynamic_locks();

-- Security Perimeter: Session Variables & Dynamic Locks
-- These variables are set by the Go orchestrator via SET LOCAL.
-- If these aren't set, the trigger below will FAIL-CLOSED.

CREATE OR REPLACE FUNCTION auth.enforce_dynamic_locks()
RETURNS TRIGGER AS $$
DECLARE
    current_role TEXT;
    project_slug TEXT;
BEGIN
    -- 1. Extract session variables injected by Go Orchestrator
    current_role := current_setting('request.jwt.claim.role', true);
    project_slug := current_setting('cascata.project_slug', true);

    -- 2. Fail-Closed: If security context is missing, block EVERYTHING.
    IF current_role IS NULL OR project_slug IS NULL THEN
        RAISE EXCEPTION 'Security Perimeter Breach: Missing cascading context. Operation aborted.';
    END IF;

    -- 3. Logic for additional dynamic locks (e.g., Panic Mode check)
    -- This can be expanded of the Phase 3B features.
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Zero-Trust: Revoke all by default (Apartment entry restricted)
REVOKE ALL ON SCHEMA public FROM public;
GRANT USAGE ON SCHEMA public TO postgres;
GRANT USAGE ON SCHEMA public TO anon;
GRANT USAGE ON SCHEMA public TO authenticated;
GRANT USAGE ON SCHEMA public TO service_role;

-- Tables born inside the perimeter
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO postgres;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO anon;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO service_role;
