CREATE TABLE IF NOT EXISTS tools (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
  timestamp TIMESTAMPTZ NOT NULL,
  tool_use_id VARCHAR(255) NOT NULL,
  tool_name VARCHAR(100) NOT NULL,
  input JSONB NOT NULL,
  output JSONB,
  is_error BOOLEAN DEFAULT FALSE,
  UNIQUE(session_id, tool_use_id)
);

CREATE INDEX IF NOT EXISTS idx_tools_session ON tools(session_id);
CREATE INDEX IF NOT EXISTS idx_tools_name ON tools(tool_name);
CREATE INDEX IF NOT EXISTS idx_tools_input_file_path ON tools((input->>'file_path'));
