package seeds

import (
	"database/sql"
	"fmt"
	"time"

	"afterzin/api/internal/auth"
)

// Run clears seed-related data and inserts fresh seed data.
// Safe to run multiple times (resets to seed state).
func Run(db *sql.DB) error {
	if err := clear(db); err != nil {
		return fmt.Errorf("clear: %w", err)
	}
	if err := insert(db); err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

func clear(db *sql.DB) error {
	tables := []string{
		"tickets", "order_items", "orders",
		"ticket_types", "lots", "event_dates", "events",
		"producers", "users",
	}
	for _, t := range tables {
		if _, err := db.Exec("DELETE FROM " + t); err != nil {
			return fmt.Errorf("delete %s: %w", t, err)
		}
	}
	return nil
}

func insert(db *sql.DB) error {
	// Password for all seed users: "123456"
	passwordHash, err := auth.HashPassword("123456")
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Users
	users := []struct {
		id           string
		name         string
		email        string
		passwordHash string
		cpf          string
		birthDate    string
		role         string
	}{
		{"seed-user-1", "João Silva", "joao@email.com", passwordHash, "123.456.789-00", "1990-05-15", "USER"},
		{"seed-user-2", "Maria Santos", "maria@email.com", passwordHash, "987.654.321-00", "1988-11-20", "USER"},
		{"seed-producer-user", "Produtor Eventos", "produtor@email.com", passwordHash, "111.222.333-44", "1985-03-10", "USER"},
	}
	for _, u := range users {
		_, err := db.Exec(`INSERT INTO users (id, name, email, password_hash, cpf, birth_date, role, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			u.id, u.name, u.email, u.passwordHash, u.cpf, u.birthDate, u.role, now)
		if err != nil {
			return fmt.Errorf("insert user %s: %w", u.id, err)
		}
	}

	// Producers (produtor user becomes producer)
	_, err = db.Exec(`INSERT INTO producers (id, user_id, approved, created_at) VALUES (?, ?, 1, ?)`,
		"seed-producer-1", "seed-producer-user", now)
	if err != nil {
		return fmt.Errorf("insert producer: %w", err)
	}

	// Events (published so they show in catalog)
	events := []struct {
		id          string
		producerID  string
		title       string
		description string
		category    string
		coverImage  string
		location    string
		address     string
		status      string
		featured    int
	}{
		{
			"seed-event-1",
			"seed-producer-1",
			"Festival de Verão 2025",
			"O maior festival de música do Brasil está de volta! Com mais de 50 artistas nacionais e internacionais.",
			"festivais",
			"https://images.unsplash.com/photo-1470229722913-7c0e2dbbafd3?w=800&q=80",
			"Arena Fonte Nova",
			"Ladeira da Fonte das Pedras - Nazaré, Salvador - BA",
			"PUBLISHED",
			1,
		},
		{
			"seed-event-2",
			"seed-producer-1",
			"Show Anitta - Funk Generation Tour",
			"Anitta apresenta sua nova turnê mundial com um show espetacular cheio de hits e surpresas.",
			"shows",
			"https://images.unsplash.com/photo-1493225457124-a3eb161ffa5f?w=800&q=80",
			"Allianz Parque",
			"Av. Francisco Matarazzo, 1705 - São Paulo - SP",
			"PUBLISHED",
			1,
		},
		{
			"seed-event-3",
			"seed-producer-1",
			"Final Copa do Brasil 2025",
			"A grande decisão do futebol brasileiro! Viva a emoção de uma final histórica ao vivo no estádio.",
			"esportes",
			"https://images.unsplash.com/photo-1489944440615-453fc2b6a9a9?w=800&q=80",
			"Maracanã",
			"Av. Pres. Castelo Branco - Rio de Janeiro - RJ",
			"PUBLISHED",
			0,
		},
		{
			"seed-event-4",
			"seed-producer-1",
			"Baile da Favorita",
			"O baile mais famoso do Rio está de volta! Uma noite repleta de funk, pagode e muito mais.",
			"festas",
			"https://images.unsplash.com/photo-1514525253161-7a46d19cd819?w=800&q=80",
			"Vivo Rio",
			"Av. Infante Dom Henrique, 85 - Rio de Janeiro - RJ",
			"PUBLISHED",
			0,
		},
		{
			"seed-event-5",
			"seed-producer-1",
			"O Fantasma da Ópera",
			"O musical mais icônico de todos os tempos chega ao Brasil em uma produção grandiosa.",
			"teatro",
			"https://images.unsplash.com/photo-1503095396549-807759245b35?w=800&q=80",
			"Teatro Renault",
			"Av. Brigadeiro Luís Antônio, 411 - São Paulo - SP",
			"PUBLISHED",
			0,
		},
	}
	for _, e := range events {
		_, err := db.Exec(`INSERT INTO events (id, producer_id, title, description, category, cover_image, location, address, status, featured, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			e.id, e.producerID, e.title, e.description, e.category, e.coverImage, e.location, e.address, e.status, e.featured, now, now)
		if err != nil {
			return fmt.Errorf("insert event %s: %w", e.id, err)
		}
	}

	// Event dates
	type eventDate struct {
		id        string
		eventID   string
		date      string
		startTime string
		endTime   string
	}
	eventDates := []eventDate{
		{"seed-date-1a", "seed-event-1", "2025-02-15", "16:00", "23:00"},
		{"seed-date-1b", "seed-event-1", "2025-02-16", "16:00", "23:00"},
		{"seed-date-1c", "seed-event-1", "2025-02-17", "15:00", "22:00"},
		{"seed-date-2a", "seed-event-2", "2025-03-20", "21:00", ""},
		{"seed-date-3a", "seed-event-3", "2025-11-15", "17:00", ""},
		{"seed-date-4a", "seed-event-4", "2025-02-28", "23:00", ""},
		{"seed-date-5a", "seed-event-5", "2025-04-10", "20:00", ""},
		{"seed-date-5b", "seed-event-5", "2025-04-11", "20:00", ""},
	}
	for _, d := range eventDates {
		_, err := db.Exec(`INSERT INTO event_dates (id, event_id, date, start_time, end_time, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			d.id, d.eventID, d.date, d.startTime, d.endTime, now)
		if err != nil {
			return fmt.Errorf("insert event_date %s: %w", d.id, err)
		}
	}

	// Lots (one active lot per event date)
	lots := []struct {
		id                string
		eventDateID       string
		name              string
		startsAt          string
		endsAt            string
		totalQuantity     int
		availableQuantity int
		active            int
	}{
		{"seed-lot-1a", "seed-date-1a", "2º Lote", now, "2025-02-14T23:59:00Z", 3000, 3000, 1},
		{"seed-lot-1b", "seed-date-1b", "2º Lote", now, "2025-02-15T23:59:00Z", 3000, 3000, 1},
		{"seed-lot-1c", "seed-date-1c", "2º Lote", now, "2025-02-16T23:59:00Z", 3000, 3000, 1},
		{"seed-lot-2a", "seed-date-2a", "1º Lote", now, "2025-03-19T23:59:00Z", 15000, 15000, 1},
		{"seed-lot-3a", "seed-date-3a", "Vendas Abertas", now, "2025-11-14T23:59:00Z", 40000, 40000, 1},
		{"seed-lot-4a", "seed-date-4a", "3º Lote", now, "2025-02-27T23:59:00Z", 1000, 1000, 1},
		{"seed-lot-5a", "seed-date-5a", "Temporada", now, "2025-04-09T23:59:00Z", 500, 500, 1},
		{"seed-lot-5b", "seed-date-5b", "Temporada", now, "2025-04-10T23:59:00Z", 500, 500, 1},
	}
	for _, l := range lots {
		_, err := db.Exec(`INSERT INTO lots (id, event_date_id, name, starts_at, ends_at, total_quantity, available_quantity, active, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			l.id, l.eventDateID, l.name, l.startsAt, l.endsAt, l.totalQuantity, l.availableQuantity, l.active, now)
		if err != nil {
			return fmt.Errorf("insert lot %s: %w", l.id, err)
		}
	}

	// Ticket types
	ticketTypes := []struct {
		id          string
		lotID       string
		name        string
		description string
		price       float64
		audience    string
		maxQuantity int
	}{
		{"seed-tt-1a-p", "seed-lot-1a", "Pista", "Acesso à área de pista", 280, "GENERAL", 1500},
		{"seed-tt-1a-v", "seed-lot-1a", "VIP", "Área VIP com open bar", 580, "GENERAL", 200},
		{"seed-tt-1a-c", "seed-lot-1a", "Camarote Premium", "Vista privilegiada + buffet", 1200, "GENERAL", 50},
		{"seed-tt-1b-p", "seed-lot-1b", "Pista", "Acesso à área de pista", 280, "GENERAL", 1500},
		{"seed-tt-1b-v", "seed-lot-1b", "VIP", "Área VIP com open bar", 580, "GENERAL", 200},
		{"seed-tt-1c-p", "seed-lot-1c", "Pista", "Acesso à área de pista", 280, "GENERAL", 1500},
		{"seed-tt-2a-p", "seed-lot-2a", "Pista", "Acesso à pista", 180, "MALE", 8000},
		{"seed-tt-2a-pf", "seed-lot-2a", "Pista", "Acesso à pista", 150, "FEMALE", 7000},
		{"seed-tt-2a-c", "seed-lot-2a", "Cadeira Superior", "Assento numerado", 250, "GENERAL", 2000},
		{"seed-tt-3a-a", "seed-lot-3a", "Arquibancada", "Setor popular", 150, "GENERAL", 20000},
		{"seed-tt-3a-ac", "seed-lot-3a", "Arquibancada", "Setor popular", 75, "CHILD", 5000},
		{"seed-tt-3a-cc", "seed-lot-3a", "Cadeira Coberta", "Setor coberto", 350, "GENERAL", 5000},
		{"seed-tt-4a-p", "seed-lot-4a", "Pista", "Acesso à pista de dança", 120, "MALE", 500},
		{"seed-tt-4a-pf", "seed-lot-4a", "Pista", "Acesso à pista de dança", 80, "FEMALE", 500},
		{"seed-tt-4a-v", "seed-lot-4a", "Área VIP", "Open bar + área exclusiva", 280, "MALE", 100},
		{"seed-tt-4a-vf", "seed-lot-4a", "Área VIP", "Open bar + área exclusiva", 200, "FEMALE", 100},
		{"seed-tt-5a-a", "seed-lot-5a", "Plateia A", "Melhores lugares", 380, "GENERAL", 200},
		{"seed-tt-5a-b", "seed-lot-5a", "Plateia B", "Visão central", 280, "GENERAL", 300},
		{"seed-tt-5b-a", "seed-lot-5b", "Plateia A", "Melhores lugares", 380, "GENERAL", 200},
		{"seed-tt-5b-b", "seed-lot-5b", "Plateia B", "Visão central", 280, "GENERAL", 300},
	}
	for _, tt := range ticketTypes {
		_, err := db.Exec(`INSERT INTO ticket_types (id, lot_id, name, description, price, audience, max_quantity, sold_quantity, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?)`,
			tt.id, tt.lotID, tt.name, tt.description, tt.price, tt.audience, tt.maxQuantity, now)
		if err != nil {
			return fmt.Errorf("insert ticket_type %s: %w", tt.id, err)
		}
	}

	return nil
}
