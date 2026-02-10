-- users
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  cpf TEXT NOT NULL UNIQUE,
  birth_date TEXT NOT NULL,
  photo_url TEXT,
  role TEXT NOT NULL DEFAULT 'USER',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- producers
CREATE TABLE IF NOT EXISTS producers (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  company_name TEXT,
  approved INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- events
CREATE TABLE IF NOT EXISTS events (
  id TEXT PRIMARY KEY,
  producer_id TEXT NOT NULL REFERENCES producers(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  category TEXT NOT NULL,
  cover_image TEXT NOT NULL,
  location TEXT NOT NULL,
  address TEXT,
  status TEXT NOT NULL DEFAULT 'DRAFT',
  featured INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- event_dates
CREATE TABLE IF NOT EXISTS event_dates (
  id TEXT PRIMARY KEY,
  event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  date TEXT NOT NULL,
  start_time TEXT,
  end_time TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- lots
CREATE TABLE IF NOT EXISTS lots (
  id TEXT PRIMARY KEY,
  event_date_id TEXT NOT NULL REFERENCES event_dates(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  starts_at TEXT NOT NULL,
  ends_at TEXT NOT NULL,
  total_quantity INTEGER NOT NULL,
  available_quantity INTEGER NOT NULL,
  active INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ticket_types
CREATE TABLE IF NOT EXISTS ticket_types (
  id TEXT PRIMARY KEY,
  lot_id TEXT NOT NULL REFERENCES lots(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  price REAL NOT NULL,
  audience TEXT NOT NULL DEFAULT 'GENERAL',
  max_quantity INTEGER NOT NULL,
  sold_quantity INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- orders (checkout session)
CREATE TABLE IF NOT EXISTS orders (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'PENDING',
  total REAL NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  expires_at TEXT
);

-- order_items
CREATE TABLE IF NOT EXISTS order_items (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  event_date_id TEXT NOT NULL REFERENCES event_dates(id),
  ticket_type_id TEXT NOT NULL REFERENCES ticket_types(id),
  quantity INTEGER NOT NULL,
  unit_price REAL NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- tickets (issued after payment)
CREATE TABLE IF NOT EXISTS tickets (
  id TEXT PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  qr_code TEXT NOT NULL UNIQUE,
  order_id TEXT NOT NULL REFERENCES orders(id),
  order_item_id TEXT NOT NULL REFERENCES order_items(id),
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  event_id TEXT NOT NULL REFERENCES events(id),
  event_date_id TEXT NOT NULL REFERENCES event_dates(id),
  ticket_type_id TEXT NOT NULL REFERENCES ticket_types(id),
  used INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_events_producer ON events(producer_id);
CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);
CREATE INDEX IF NOT EXISTS idx_event_dates_event ON event_dates(event_id);
CREATE INDEX IF NOT EXISTS idx_lots_event_date ON lots(event_date_id);
CREATE INDEX IF NOT EXISTS idx_ticket_types_lot ON ticket_types(lot_id);
CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_tickets_user ON tickets(user_id);
CREATE INDEX IF NOT EXISTS idx_tickets_qr ON tickets(qr_code);
