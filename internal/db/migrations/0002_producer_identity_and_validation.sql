-- tickets: timestamp de uso do ingresso
ALTER TABLE tickets ADD COLUMN used_at TEXT;

-- auditoria de validação de ingressos
CREATE TABLE IF NOT EXISTS ticket_validations (
  id TEXT PRIMARY KEY,
  ticket_id TEXT NOT NULL REFERENCES tickets(id),
  event_id TEXT NOT NULL REFERENCES events(id),
  producer_id TEXT NOT NULL REFERENCES producers(id),
  validated_at TEXT NOT NULL DEFAULT (datetime('now')),
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_ticket_validations_ticket ON ticket_validations(ticket_id);
CREATE INDEX IF NOT EXISTS idx_ticket_validations_event ON ticket_validations(event_id);
CREATE INDEX IF NOT EXISTS idx_ticket_validations_producer ON ticket_validations(producer_id);
