CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tool VARCHAR(50) NOT NULL CHECK (tool IN ('claude-code', 'cursor')),
  session_id VARCHAR(255) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  ended_at TIMESTAMPTZ,
  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  cwd TEXT NOT NULL,
  slug TEXT,
  uploaded_by VARCHAR(255) NOT NULL,
  api_key_id UUID NOT NULL,
  duration_seconds INTEGER,
  message_count INTEGER DEFAULT 0,
  tool_use_count INTEGER DEFAULT 0,
  UNIQUE(tool, session_id)
);

CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_tool ON sessions(tool);
CREATE INDEX IF NOT EXISTS idx_sessions_uploaded_by ON sessions(uploaded_by);
CREATE INDEX IF NOT EXISTS idx_sessions_uploaded_at ON sessions(uploaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_cwd ON sessions USING gin(to_tsvector('english', cwd));
