CREATE TABLE IF NOT EXISTS customers (
  id UUID PRIMARY KEY,
  bank_id UUID NOT NULL REFERENCES banks (id),
  name VARCHAR(160) NOT NULL,
  document VARCHAR(14) NOT NULL,
  email VARCHAR(254) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT customers_document_format CHECK (document ~ '^[0-9]{11}$|^[0-9]{14}$'),
  CONSTRAINT customers_email_format CHECK (email ~* '^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$')
);

CREATE UNIQUE INDEX IF NOT EXISTS customers_bank_document_idx ON customers (bank_id, document);
CREATE INDEX IF NOT EXISTS customers_bank_idx ON customers (bank_id);

CREATE TABLE IF NOT EXISTS accounts (
  id UUID PRIMARY KEY,
  bank_id UUID NOT NULL REFERENCES banks (id),
  customer_id UUID NOT NULL REFERENCES customers (id),
  agency VARCHAR(8) NOT NULL,
  number VARCHAR(20) NOT NULL,
  balance_cents BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT accounts_agency_format CHECK (agency ~ '^[0-9]{1,8}$'),
  CONSTRAINT accounts_number_format CHECK (number ~ '^[0-9]{1,20}$'),
  CONSTRAINT accounts_balance_non_negative CHECK (balance_cents >= 0),
  CONSTRAINT accounts_status_valid CHECK (status IN ('active', 'blocked', 'closed'))
);

CREATE UNIQUE INDEX IF NOT EXISTS accounts_bank_agency_number_idx ON accounts (bank_id, agency, number);
CREATE INDEX IF NOT EXISTS accounts_bank_idx ON accounts (bank_id);
CREATE INDEX IF NOT EXISTS accounts_customer_idx ON accounts (customer_id);
