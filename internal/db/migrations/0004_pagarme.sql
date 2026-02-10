-- Pagar.me V5 integration
-- Adds columns for Pagar.me recipient tracking, order/charge IDs
-- Creates webhook events log for idempotency

-- Producer Pagar.me recipient
ALTER TABLE producers ADD COLUMN pagarme_recipient_id TEXT;

-- Pagar.me order/charge tracking on orders
ALTER TABLE orders ADD COLUMN pagarme_order_id TEXT;
ALTER TABLE orders ADD COLUMN pagarme_charge_id TEXT;

-- Webhook events log for deduplication and auditing
CREATE TABLE IF NOT EXISTS pagarme_webhook_events (
  id TEXT PRIMARY KEY,
  pagarme_event_id TEXT NOT NULL UNIQUE,
  event_type TEXT NOT NULL,
  processed INTEGER NOT NULL DEFAULT 0,
  error_message TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_pagarme_wh_event_id ON pagarme_webhook_events(pagarme_event_id);
CREATE INDEX IF NOT EXISTS idx_producers_pagarme ON producers(pagarme_recipient_id);
CREATE INDEX IF NOT EXISTS idx_orders_pagarme_order ON orders(pagarme_order_id);
CREATE INDEX IF NOT EXISTS idx_orders_pagarme_charge ON orders(pagarme_charge_id);
