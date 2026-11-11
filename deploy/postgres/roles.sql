-- Run as the PostgreSQL database owner/admin, not as the Eye API role.
-- Supply the password through your secret manager; do not commit a real value.
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'eye_app') THEN
    CREATE ROLE eye_app LOGIN PASSWORD 'REPLACE_FROM_SECRET_MANAGER' NOSUPERUSER NOCREATEDB NOCREATEROLE;
  END IF;
END
$$;

GRANT CONNECT ON DATABASE eye TO eye_app;
GRANT USAGE ON SCHEMA public TO eye_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO eye_app;
GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO eye_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO eye_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO eye_app;

-- The owner/admin role runs migrations and applies RLS policies. The API
-- must connect as eye_app after policies are installed.
