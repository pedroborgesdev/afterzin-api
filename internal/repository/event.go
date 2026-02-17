package repository

import (
	"database/sql"

	"github.com/google/uuid"
)

func ListEventsByProducerID(db *sql.DB, producerID string) ([]string, error) {
	rows, err := db.Query(`SELECT id FROM events WHERE producer_id = ? ORDER BY created_at DESC`, producerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListEventsByProducerIDExcludingDraft returns event IDs for a producer with status != 'DRAFT' (for public profile).
func ListEventsByProducerIDExcludingDraft(db *sql.DB, producerID string) ([]string, error) {
	rows, err := db.Query(`SELECT id FROM events WHERE producer_id = ? AND status != 'DRAFT' ORDER BY created_at DESC`, producerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// EventProducerID returns the producer_id for an event.
func EventProducerID(db *sql.DB, eventID string) (string, error) {
	var producerID string
	err := db.QueryRow(`SELECT producer_id FROM events WHERE id = ?`, eventID).Scan(&producerID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return producerID, err
}

func ListPublishedEvents(db *sql.DB, category, date, city *string) ([]string, error) {
	q := `SELECT id FROM events WHERE status = 'PUBLISHED'`
	args := []interface{}{}
	if category != nil && *category != "" {
		q += ` AND category = ?`
		args = append(args, *category)
	}
	if date != nil && *date != "" {
		q += ` AND id IN (SELECT event_id FROM event_dates WHERE date = ?)`
		args = append(args, *date)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func EventByID(db *sql.DB, id string) (*EventRow, error) {
	var e EventRow
	err := db.QueryRow(`SELECT id, producer_id, title, description, category, cover_image, location, address, status, featured FROM events WHERE id = ?`, id).Scan(
		&e.ID, &e.ProducerID, &e.Title, &e.Description, &e.Category, &e.CoverImage, &e.Location, &e.Address, &e.Status, &e.Featured,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

type EventRow struct {
	ID          string
	ProducerID  string
	Title       string
	Description string
	Category    string
	CoverImage  string
	Location    string
	Address     sql.NullString
	Status      string
	Featured    int
}

func EventDateIDsByEvent(db *sql.DB, eventID string) ([]string, error) {
	rows, err := db.Query(`SELECT id FROM event_dates WHERE event_id = ? ORDER BY date`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type EventDateRow struct {
	ID        string
	EventID   string
	Date      string
	StartTime sql.NullString
	EndTime   sql.NullString
}

func EventDateByID(db *sql.DB, id string) (*EventDateRow, error) {
	var d EventDateRow
	err := db.QueryRow(`SELECT id, event_id, date, start_time, end_time FROM event_dates WHERE id = ?`, id).Scan(
		&d.ID, &d.EventID, &d.Date, &d.StartTime, &d.EndTime,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func LotIDsByEventDate(db *sql.DB, dateID string) ([]string, error) {
	rows, err := db.Query(`SELECT id FROM lots WHERE event_date_id = ?`, dateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type LotRow struct {
	ID                string
	EventDateID       string
	Name              string
	StartsAt          string
	EndsAt            string
	TotalQuantity     int
	AvailableQuantity int
	Active            int
}

func LotByID(db *sql.DB, id string) (*LotRow, error) {
	var l LotRow
	err := db.QueryRow(`SELECT id, event_date_id, name, starts_at, ends_at, total_quantity, available_quantity, active FROM lots WHERE id = ?`, id).Scan(
		&l.ID, &l.EventDateID, &l.Name, &l.StartsAt, &l.EndsAt, &l.TotalQuantity, &l.AvailableQuantity, &l.Active,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func TicketTypeIDsByLot(db *sql.DB, lotID string) ([]string, error) {
	rows, err := db.Query(`SELECT id FROM ticket_types WHERE lot_id = ?`, lotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type TicketTypeRow struct {
	ID           string
	LotID        string
	Name         string
	Description  sql.NullString
	Price        float64
	Audience     string
	MaxQuantity  int
	SoldQuantity int
}

func TicketTypeByID(db *sql.DB, id string) (*TicketTypeRow, error) {
	var t TicketTypeRow
	err := db.QueryRow(`SELECT id, lot_id, name, description, price, audience, max_quantity, sold_quantity FROM ticket_types WHERE id = ?`, id).Scan(
		&t.ID, &t.LotID, &t.Name, &t.Description, &t.Price, &t.Audience, &t.MaxQuantity, &t.SoldQuantity,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func ProducerIDByUser(db *sql.DB, userID string) (string, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM producers WHERE user_id = ?`, userID).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}

type ProducerRow struct {
	ID          string
	UserID      string
	CompanyName sql.NullString
	Approved    int
}

func ProducerByID(db *sql.DB, id string) (*ProducerRow, error) {
	var p ProducerRow
	err := db.QueryRow(`SELECT id, user_id, company_name, approved FROM producers WHERE id = ?`, id).Scan(
		&p.ID, &p.UserID, &p.CompanyName, &p.Approved,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func CreateProducer(db *sql.DB, userID string) (string, error) {
	id := uuid.New().String()
	_, err := db.Exec(`INSERT INTO producers (id, user_id, approved) VALUES (?, ?, 1)`, id, userID)
	return id, err
}

func CreateEvent(db *sql.DB, producerID, title, description, category, coverImage, location string, address *string) (string, error) {
	id := uuid.New().String()
	var addr sql.NullString
	if address != nil {
		addr = sql.NullString{String: *address, Valid: true}
	}
	_, err := db.Exec(`INSERT INTO events (id, producer_id, title, description, category, cover_image, location, address, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'DRAFT')`,
		id, producerID, title, description, category, coverImage, location, addr,
	)
	return id, err
}

func CreateEventDate(db *sql.DB, eventID, date string, startTime, endTime *string) (string, error) {
	id := uuid.New().String()
	var st, et sql.NullString
	if startTime != nil {
		st = sql.NullString{String: *startTime, Valid: true}
	}
	if endTime != nil {
		et = sql.NullString{String: *endTime, Valid: true}
	}
	_, err := db.Exec(`INSERT INTO event_dates (id, event_id, date, start_time, end_time) VALUES (?, ?, ?, ?, ?)`,
		id, eventID, date, st, et,
	)
	return id, err
}

func CreateLot(db *sql.DB, eventDateID, name, startsAt, endsAt string, totalQuantity int) (string, error) {
	id := uuid.New().String()
	_, err := db.Exec(`INSERT INTO lots (id, event_date_id, name, starts_at, ends_at, total_quantity, available_quantity, active) VALUES (?, ?, ?, ?, ?, ?, ?, 1)`,
		id, eventDateID, name, startsAt, endsAt, totalQuantity, totalQuantity,
	)
	return id, err
}

func CreateTicketType(db *sql.DB, lotID, name string, description *string, price float64, audience string, maxQuantity int) (string, error) {
	id := uuid.New().String()
	var desc sql.NullString
	if description != nil {
		desc = sql.NullString{String: *description, Valid: true}
	}
	_, err := db.Exec(`INSERT INTO ticket_types (id, lot_id, name, description, price, audience, max_quantity, sold_quantity) VALUES (?, ?, ?, ?, ?, ?, ?, 0)`,
		id, lotID, name, desc, price, audience, maxQuantity,
	)
	return id, err
}

func UpdateEventStatus(db *sql.DB, eventID, status string) error {
	_, err := db.Exec(`UPDATE events SET status = ?, updated_at = datetime('now') WHERE id = ?`, status, eventID)
	return err
}

func UpdateEvent(db *sql.DB, eventID string, title, description, category, coverImage, location *string, address *string, featured *bool) error {
	if title == nil && description == nil && category == nil && coverImage == nil && location == nil && address == nil && featured == nil {
		return nil
	}
	q := `UPDATE events SET updated_at = datetime('now')`
	args := []interface{}{}
	if title != nil {
		q += `, title = ?`
		args = append(args, *title)
	}
	if description != nil {
		q += `, description = ?`
		args = append(args, *description)
	}
	if category != nil {
		q += `, category = ?`
		args = append(args, *category)
	}
	if coverImage != nil {
		q += `, cover_image = ?`
		args = append(args, *coverImage)
	}
	if location != nil {
		q += `, location = ?`
		args = append(args, *location)
	}
	if address != nil {
		q += `, address = ?`
		args = append(args, *address)
	}
	if featured != nil {
		v := 0
		if *featured {
			v = 1
		}
		q += `, featured = ?`
		args = append(args, v)
	}
	q += ` WHERE id = ?`
	args = append(args, eventID)
	_, err := db.Exec(q, args...)
	return err
}

// ---------- Transactional versions ----------

// EventByIDTx retrieves an event within a transaction.
func EventByIDTx(tx *sql.Tx, id string) (*EventRow, error) {
	var e EventRow
	err := tx.QueryRow(`SELECT id, producer_id, title, description, category, cover_image, location, address, status, featured FROM events WHERE id = ?`, id).Scan(
		&e.ID, &e.ProducerID, &e.Title, &e.Description, &e.Category, &e.CoverImage, &e.Location, &e.Address, &e.Status, &e.Featured,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// EventDateByIDTx retrieves an event date within a transaction.
func EventDateByIDTx(tx *sql.Tx, id string) (*EventDateRow, error) {
	var d EventDateRow
	err := tx.QueryRow(`SELECT id, event_id, date, start_time, end_time FROM event_dates WHERE id = ?`, id).Scan(
		&d.ID, &d.EventID, &d.Date, &d.StartTime, &d.EndTime,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// TicketTypeByIDTx retrieves a ticket type within a transaction.
func TicketTypeByIDTx(tx *sql.Tx, id string) (*TicketTypeRow, error) {
	var t TicketTypeRow
	err := tx.QueryRow(`SELECT id, lot_id, name, description, price, audience, max_quantity, sold_quantity FROM ticket_types WHERE id = ?`, id).Scan(
		&t.ID, &t.LotID, &t.Name, &t.Description, &t.Price, &t.Audience, &t.MaxQuantity, &t.SoldQuantity,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}
