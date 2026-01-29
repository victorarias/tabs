CREATE TABLE IF NOT EXISTS tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  tag_key VARCHAR(100) NOT NULL,
  tag_value VARCHAR(255) NOT NULL,
  UNIQUE(session_id, tag_key, tag_value)
);

CREATE INDEX IF NOT EXISTS idx_tags_session ON tags(session_id);
CREATE INDEX IF NOT EXISTS idx_tags_key_value ON tags(tag_key, tag_value);
CREATE INDEX IF NOT EXISTS idx_tags_key ON tags(tag_key);
