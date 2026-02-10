package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type TicketRow struct {
	ID            string
	Code          string
	QRCode        string
	OrderID       string
	OrderItemID   string
	UserID        string
	EventID       string
	EventDateID   string
	TicketTypeID  string
	Used          int
	UsedAt        sql.NullString
	CreatedAt     time.Time
}

func parseDateTime(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	if t.IsZero() {
		return time.Now()
	}
	return t
}

func TicketsByUserID(db *sql.DB, userID string) ([]*TicketRow, error) {
	rows, err := db.Query(`SELECT id, code, qr_code, order_id, order_item_id, user_id, event_id, event_date_id, ticket_type_id, used, used_at, created_at FROM tickets WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*TicketRow
	for rows.Next() {
		t, err := scanTicketRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func scanTicketRow(rows interface {
	Scan(dest ...interface{}) error
}) (*TicketRow, error) {
	var t TicketRow
	var usedAt, createdAt sql.NullString
	err := rows.Scan(&t.ID, &t.Code, &t.QRCode, &t.OrderID, &t.OrderItemID, &t.UserID, &t.EventID, &t.EventDateID, &t.TicketTypeID, &t.Used, &usedAt, &createdAt)
	if err != nil {
		return nil, err
	}
	if usedAt.Valid {
		t.UsedAt = usedAt
	}
	if createdAt.Valid {
		t.CreatedAt = parseDateTime(createdAt.String)
	}
	return &t, nil
}

func TicketByID(db *sql.DB, id string) (*TicketRow, error) {
	var t TicketRow
	var usedAt, createdAt sql.NullString
	err := db.QueryRow(`SELECT id, code, qr_code, order_id, order_item_id, user_id, event_id, event_date_id, ticket_type_id, used, COALESCE(used_at,'') as used_at, created_at FROM tickets WHERE id = ?`, id).Scan(
		&t.ID, &t.Code, &t.QRCode, &t.OrderID, &t.OrderItemID, &t.UserID, &t.EventID, &t.EventDateID, &t.TicketTypeID, &t.Used, &usedAt, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if usedAt.Valid && usedAt.String != "" {
		t.UsedAt = usedAt
	}
	if createdAt.Valid {
		t.CreatedAt = parseDateTime(createdAt.String)
	}
	return &t, nil
}

func TicketByQRCode(db *sql.DB, qrCode string) (*TicketRow, error) {
	var t TicketRow
	var usedAt, createdAt sql.NullString
	err := db.QueryRow(`SELECT id, code, qr_code, order_id, order_item_id, user_id, event_id, event_date_id, ticket_type_id, used, used_at, created_at FROM tickets WHERE qr_code = ?`, qrCode).Scan(
		&t.ID, &t.Code, &t.QRCode, &t.OrderID, &t.OrderItemID, &t.UserID, &t.EventID, &t.EventDateID, &t.TicketTypeID, &t.Used, &usedAt, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if usedAt.Valid {
		t.UsedAt = usedAt
	}
	if createdAt.Valid {
		t.CreatedAt = parseDateTime(createdAt.String)
	}
	return &t, nil
}

func MarkTicketUsed(db *sql.DB, id string) error {
	_, err := db.Exec(`UPDATE tickets SET used = 1, used_at = datetime('now') WHERE id = ?`, id)
	return err
}

// MarkTicketUsedIfNotUsed marks the ticket as used only if it is not already used.
// Returns true if the row was updated (exactly one row), false if already used or not found.
// Used for concurrent-safe validation: only one request can succeed.
func MarkTicketUsedIfNotUsed(db *sql.DB, id string) (updated bool, err error) {
	res, err := db.Exec(`UPDATE tickets SET used = 1, used_at = datetime('now') WHERE id = ? AND used = 0`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func InsertTicketValidation(db *sql.DB, ticketID, eventID, producerID string) error {
	id := uuid.New().String()
	_, err := db.Exec(`INSERT INTO ticket_validations (id, ticket_id, event_id, producer_id) VALUES (?, ?, ?, ?)`,
		id, ticketID, eventID, producerID,
	)
	return err
}

func GenerateTicketCode() string {
	return uuid.New().String()[:8]
}

func GenerateQRCode() string {
	return uuid.New().String()
}
