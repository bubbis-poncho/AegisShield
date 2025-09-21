-- Migration: 003_create_transactions_table
-- Description: Drop transactions table
-- Down Migration

DROP INDEX IF EXISTS idx_transactions_external_id_unique;
DROP INDEX IF EXISTS idx_transactions_business_rules;
DROP INDEX IF EXISTS idx_transactions_enriched_data;
DROP INDEX IF EXISTS idx_transactions_alert_timestamp;
DROP INDEX IF EXISTS idx_transactions_risk_timestamp;
DROP INDEX IF EXISTS idx_transactions_amount_timestamp;
DROP INDEX IF EXISTS idx_transactions_account_timestamp;
DROP INDEX IF EXISTS idx_transactions_created_at;
DROP INDEX IF EXISTS idx_transactions_processed_at;
DROP INDEX IF EXISTS idx_transactions_alert_triggered;
DROP INDEX IF EXISTS idx_transactions_status;
DROP INDEX IF EXISTS idx_transactions_risk_score;
DROP INDEX IF EXISTS idx_transactions_dest_account;
DROP INDEX IF EXISTS idx_transactions_source_account;
DROP INDEX IF EXISTS idx_transactions_timestamp;
DROP INDEX IF EXISTS idx_transactions_type;
DROP INDEX IF EXISTS idx_transactions_currency;
DROP INDEX IF EXISTS idx_transactions_amount;
DROP INDEX IF EXISTS idx_transactions_external_id;

DROP TABLE IF EXISTS transactions;