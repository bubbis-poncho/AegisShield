-- Migration: 003_create_transactions_table
-- Description: Create transactions table for processed transaction data
-- Up Migration

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(255),
    amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    transaction_type VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    source_account_id VARCHAR(100) NOT NULL,
    destination_account_id VARCHAR(100),
    description TEXT,
    risk_score DECIMAL(3,2) DEFAULT 0.00,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    alert_triggered BOOLEAN DEFAULT FALSE,
    enriched_data JSONB,
    business_rule_results JSONB,
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_transactions_external_id ON transactions(external_id);
CREATE INDEX IF NOT EXISTS idx_transactions_amount ON transactions(amount);
CREATE INDEX IF NOT EXISTS idx_transactions_currency ON transactions(currency);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(transaction_type);
CREATE INDEX IF NOT EXISTS idx_transactions_timestamp ON transactions(timestamp);
CREATE INDEX IF NOT EXISTS idx_transactions_source_account ON transactions(source_account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_dest_account ON transactions(destination_account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_risk_score ON transactions(risk_score);
CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);
CREATE INDEX IF NOT EXISTS idx_transactions_alert_triggered ON transactions(alert_triggered);
CREATE INDEX IF NOT EXISTS idx_transactions_processed_at ON transactions(processed_at);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_transactions_account_timestamp ON transactions(source_account_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_amount_timestamp ON transactions(amount DESC, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_risk_timestamp ON transactions(risk_score DESC, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_alert_timestamp ON transactions(alert_triggered, timestamp DESC) WHERE alert_triggered = TRUE;

-- GIN indexes for JSONB fields
CREATE INDEX IF NOT EXISTS idx_transactions_enriched_data ON transactions USING GIN(enriched_data);
CREATE INDEX IF NOT EXISTS idx_transactions_business_rules ON transactions USING GIN(business_rule_results);

-- Check constraints
ALTER TABLE transactions 
ADD CONSTRAINT chk_transactions_amount 
CHECK (amount > 0);

ALTER TABLE transactions 
ADD CONSTRAINT chk_transactions_currency 
CHECK (LENGTH(currency) = 3);

ALTER TABLE transactions 
ADD CONSTRAINT chk_transactions_risk_score 
CHECK (risk_score >= 0.00 AND risk_score <= 1.00);

ALTER TABLE transactions 
ADD CONSTRAINT chk_transactions_status 
CHECK (status IN ('pending', 'processing', 'processed', 'failed', 'rejected'));

ALTER TABLE transactions 
ADD CONSTRAINT chk_transactions_type 
CHECK (transaction_type IN ('CREDIT_CARD', 'DEBIT_CARD', 'WIRE_TRANSFER', 'ACH_TRANSFER', 
                           'CASH_DEPOSIT', 'CASH_WITHDRAWAL', 'CHECK_DEPOSIT', 'CHECK_WITHDRAWAL',
                           'ONLINE_PURCHASE', 'ATM_WITHDRAWAL', 'DIRECT_DEBIT', 'DIRECT_CREDIT',
                           'PEER_TO_PEER', 'MOBILE_PAYMENT', 'CRYPTOCURRENCY', 'OTHER'));

-- Unique constraint on external_id to prevent duplicates
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_external_id_unique ON transactions(external_id) WHERE external_id IS NOT NULL;

-- Comments
COMMENT ON TABLE transactions IS 'Stores processed transaction data with enrichment and risk analysis';
COMMENT ON COLUMN transactions.id IS 'Internal unique identifier for the transaction';
COMMENT ON COLUMN transactions.external_id IS 'External system transaction identifier';
COMMENT ON COLUMN transactions.amount IS 'Transaction amount in the specified currency';
COMMENT ON COLUMN transactions.currency IS 'ISO 4217 currency code';
COMMENT ON COLUMN transactions.transaction_type IS 'Type of transaction';
COMMENT ON COLUMN transactions.timestamp IS 'When the transaction occurred';
COMMENT ON COLUMN transactions.source_account_id IS 'Source account identifier';
COMMENT ON COLUMN transactions.destination_account_id IS 'Destination account identifier (if applicable)';
COMMENT ON COLUMN transactions.description IS 'Transaction description or memo';
COMMENT ON COLUMN transactions.risk_score IS 'Calculated risk score (0.00 to 1.00)';
COMMENT ON COLUMN transactions.status IS 'Processing status of the transaction';
COMMENT ON COLUMN transactions.alert_triggered IS 'Whether this transaction triggered any alerts';
COMMENT ON COLUMN transactions.enriched_data IS 'Additional enriched data as JSON';
COMMENT ON COLUMN transactions.business_rule_results IS 'Business rule evaluation results as JSON';
COMMENT ON COLUMN transactions.processed_at IS 'When the transaction was processed by the system';