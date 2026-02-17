-- Order status change audit trail
-- Tracks all status transitions for debugging and compliance

CREATE TABLE IF NOT EXISTS order_status_history (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  old_status TEXT NOT NULL,
  new_status TEXT NOT NULL,
  reason TEXT,
  pagarme_event_id TEXT,
  pagarme_order_id TEXT,
  pagarme_charge_id TEXT,
  error_message TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_order_status_history_order ON order_status_history(order_id);
CREATE INDEX IF NOT EXISTS idx_order_status_history_created ON order_status_history(created_at);

-- Add processed_at timestamp to webhook events for better tracking
ALTER TABLE pagarme_webhook_events ADD COLUMN processed_at TEXT;
