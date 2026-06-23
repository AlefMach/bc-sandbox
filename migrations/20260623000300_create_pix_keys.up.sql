CREATE TABLE IF NOT EXISTS pix_keys (
  id UUID PRIMARY KEY,
  account_id UUID NOT NULL REFERENCES accounts (id),
  key_type VARCHAR(20) NOT NULL,
  key_value VARCHAR(254) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT pix_keys_type_valid CHECK (key_type IN ('cpf', 'cnpj', 'email', 'phone', 'random')),
  CONSTRAINT pix_keys_status_valid CHECK (status IN ('active', 'inactive')),
  CONSTRAINT pix_keys_value_present CHECK (length(key_value) > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS pix_keys_key_value_idx ON pix_keys (key_value);
CREATE INDEX IF NOT EXISTS pix_keys_account_idx ON pix_keys (account_id);
