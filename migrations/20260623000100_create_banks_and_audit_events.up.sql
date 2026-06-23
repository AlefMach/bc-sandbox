CREATE TABLE IF NOT EXISTS banks (
  id UUID PRIMARY KEY,
  name VARCHAR(160) NOT NULL,
  code VARCHAR(3) NOT NULL UNIQUE,
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT banks_code_format CHECK (code ~ '^[0-9]{3}$'),
  CONSTRAINT banks_status_valid CHECK (status IN ('active', 'inactive', 'offline', 'maintenance'))
);

CREATE TABLE IF NOT EXISTS audit_events (
  id UUID PRIMARY KEY,
  entity_type VARCHAR(80) NOT NULL,
  entity_id UUID NOT NULL,
  event_type VARCHAR(120) NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS audit_events_entity_idx ON audit_events (entity_type, entity_id);
CREATE INDEX IF NOT EXISTS audit_events_created_at_idx ON audit_events (created_at);

INSERT INTO banks (id, name, code, status, created_at, updated_at)
VALUES
  ('00000000-0000-4000-8000-000000000001', 'Banco Central Sandbox', '001', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('00000000-0000-4000-8000-000000000237', 'Banco Participante 237', '237', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (code) DO NOTHING;
