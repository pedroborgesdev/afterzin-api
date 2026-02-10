-- Stripe Connect integration
-- Adds columns for Stripe account tracking, products/prices, checkout sessions
-- Creates webhook events log for idempotency

-- Producer Stripe Connect account
ALTER TABLE producers ADD COLUMN stripe_account_id TEXT;
ALTER TABLE producers ADD COLUMN stripe_onboarding_complete INTEGER NOT NULL DEFAULT 0;
ALTER TABLE producers ADD COLUMN pix_key TEXT;
ALTER TABLE producers ADD COLUMN pix_key_type TEXT;

-- Stripe Product/Price IDs on ticket_types (products live on platform account)
ALTER TABLE ticket_types ADD COLUMN stripe_product_id TEXT;
ALTER TABLE ticket_types ADD COLUMN stripe_price_id TEXT;

-- Stripe session/payment tracking on orders
ALTER TABLE orders ADD COLUMN stripe_checkout_session_id TEXT;
ALTER TABLE orders ADD COLUMN stripe_payment_intent_id TEXT;

-- Webhook events log for deduplication and auditing
CREATE TABLE IF NOT EXISTS stripe_webhook_events (
  id TEXT PRIMARY KEY,
  stripe_event_id TEXT NOT NULL UNIQUE,
  event_type TEXT NOT NULL,
  processed INTEGER NOT NULL DEFAULT 0,
  error_message TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_stripe_wh_event_id ON stripe_webhook_events(stripe_event_id);
CREATE INDEX IF NOT EXISTS idx_producers_stripe ON producers(stripe_account_id);
CREATE INDEX IF NOT EXISTS idx_orders_stripe_session ON orders(stripe_checkout_session_id);
CREATE INDEX IF NOT EXISTS idx_orders_stripe_pi ON orders(stripe_payment_intent_id);
