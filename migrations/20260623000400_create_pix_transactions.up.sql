CREATE TABLE IF NOT EXISTS pix_transactions (
  id UUID PRIMARY KEY,
  transaction_type VARCHAR(20) NOT NULL DEFAULT 'pix',
  status VARCHAR(60) NOT NULL,
  amount_cents BIGINT NOT NULL,
  payer_bank_id UUID NOT NULL REFERENCES banks (id),
  payer_account_id UUID NOT NULL REFERENCES accounts (id),
  receiver_bank_id UUID NOT NULL REFERENCES banks (id),
  receiver_account_id UUID NOT NULL REFERENCES accounts (id),
  pix_key VARCHAR(254) NOT NULL,
  reserved_at TIMESTAMP NULL,
  settled_at TIMESTAMP NULL,
  credited_at TIMESTAMP NULL,
  failure_reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT pix_transactions_type_valid CHECK (transaction_type = 'pix'),
  CONSTRAINT pix_transactions_amount_positive CHECK (amount_cents > 0),
  CONSTRAINT pix_transactions_status_valid CHECK (status IN (
    'created',
    'payer_account_validated',
    'pix_key_consulted',
    'receiver_account_identified',
    'balance_validated',
    'funds_reserved',
    'sent_to_central_bank',
    'settled',
    'receiver_bank_notified',
    'receiver_account_credited',
    'completed',
    'failed'
  ))
);

CREATE INDEX IF NOT EXISTS pix_transactions_payer_bank_idx ON pix_transactions (payer_bank_id);
CREATE INDEX IF NOT EXISTS pix_transactions_receiver_bank_idx ON pix_transactions (receiver_bank_id);
CREATE INDEX IF NOT EXISTS pix_transactions_status_idx ON pix_transactions (status);

CREATE TABLE IF NOT EXISTS pix_transaction_events (
  id UUID PRIMARY KEY,
  transaction_id UUID NOT NULL REFERENCES pix_transactions (id),
  transaction_type VARCHAR(20) NOT NULL DEFAULT 'pix',
  event_type VARCHAR(80) NOT NULL,
  previous_status VARCHAR(60) NOT NULL,
  new_status VARCHAR(60) NOT NULL,
  message TEXT NOT NULL,
  service VARCHAR(80) NOT NULL,
  bank_id UUID NOT NULL REFERENCES banks (id),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS pix_transaction_events_transaction_idx ON pix_transaction_events (transaction_id, created_at);
