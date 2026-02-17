package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

func CreateOrder(db *sql.DB, userID string, total float64, exp time.Duration) (string, error) {
	id := uuid.New().String()
	expAt := time.Now().Add(exp).UTC().Format(time.RFC3339)
	_, err := db.Exec(`INSERT INTO orders (id, user_id, status, total, expires_at) VALUES (?, ?, 'PENDING', ?, ?)`, id, userID, total, expAt)
	return id, err
}

func OrderByID(db *sql.DB, id string) (userID string, status string, total float64, err error) {
	err = db.QueryRow(`SELECT user_id, status, total FROM orders WHERE id = ?`, id).Scan(&userID, &status, &total)
	return
}

func ConfirmOrder(db *sql.DB, orderID string) error {
	_, err := db.Exec(`UPDATE orders SET status = 'PAID' WHERE id = ? AND status IN ('PENDING','PROCESSING')`, orderID)
	return err
}

// ClaimOrderProcessing atomically marks an order as PROCESSING if it's currently PENDING.
// Returns true if the claim succeeded (rows affected == 1).
func ClaimOrderProcessing(db *sql.DB, orderID string) (bool, error) {
	res, err := db.Exec(`UPDATE orders SET status = 'PROCESSING' WHERE id = ? AND status = 'PENDING'`, orderID)
	if err != nil {
		return false, err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return ra == 1, nil
}

func CreateOrderItem(db *sql.DB, orderID, eventDateID, ticketTypeID string, quantity int, unitPrice float64) (string, error) {
	id := uuid.New().String()
	_, err := db.Exec(`INSERT INTO order_items (id, order_id, event_date_id, ticket_type_id, quantity, unit_price) VALUES (?, ?, ?, ?, ?, ?)`,
		id, orderID, eventDateID, ticketTypeID, quantity, unitPrice,
	)
	return id, err
}

func OrderItemsByOrderID(db *sql.DB, orderID string) ([]OrderItemRow, error) {
	rows, err := db.Query(`SELECT id, order_id, event_date_id, ticket_type_id, quantity, unit_price FROM order_items WHERE order_id = ?`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []OrderItemRow
	for rows.Next() {
		var o OrderItemRow
		if err := rows.Scan(&o.ID, &o.OrderID, &o.EventDateID, &o.TicketTypeID, &o.Quantity, &o.UnitPrice); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

type OrderItemRow struct {
	ID           string
	OrderID      string
	EventDateID  string
	TicketTypeID string
	Quantity     int
	UnitPrice    float64
}

func CreateTicket(db *sql.DB, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID string) (string, error) {
	id := uuid.New().String()
	_, err := db.Exec(`INSERT INTO tickets (id, code, qr_code, order_id, order_item_id, user_id, event_id, event_date_id, ticket_type_id, used) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID,
	)
	return id, err
}

// CreateTicketWithID inserts a ticket with the given id and qr_code (e.g. signed payload). Used when QR is generated from ticket id.
func CreateTicketWithID(db *sql.DB, id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID string) error {
	_, err := db.Exec(`INSERT INTO tickets (id, code, qr_code, order_id, order_item_id, user_id, event_id, event_date_id, ticket_type_id, used) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID,
	)
	return err
}

func IncrementTicketTypeSold(db *sql.DB, ticketTypeID string, n int) error {
	_, err := db.Exec(`UPDATE ticket_types SET sold_quantity = sold_quantity + ? WHERE id = ?`, n, ticketTypeID)
	return err
}

func DecrementLotAvailable(db *sql.DB, lotID string, n int) error {
	_, err := db.Exec(`UPDATE lots SET available_quantity = available_quantity - ? WHERE id = ? AND available_quantity >= ?`, n, lotID, n)
	return err
}

func LotIDByTicketTypeID(db *sql.DB, ticketTypeID string) (string, error) {
	var lotID string
	err := db.QueryRow(`SELECT lot_id FROM ticket_types WHERE id = ?`, ticketTypeID).Scan(&lotID)
	return lotID, err
}

// ---------- Transactional versions ----------

// ClaimOrderProcessingTx atomically marks an order as PROCESSING if it's currently PENDING.
// Returns true if the claim succeeded (rows affected == 1).
// This provides an optimistic lock to prevent race conditions in webhook processing.
func ClaimOrderProcessingTx(tx *sql.Tx, orderID string) (bool, error) {
	res, err := tx.Exec(`UPDATE orders SET status = 'PROCESSING' WHERE id = ? AND status = 'PENDING'`, orderID)
	if err != nil {
		return false, err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return ra == 1, nil
}

// ConfirmOrderTx confirms an order within a transaction.
// Only updates if order is in PROCESSING state to ensure proper state transition.
func ConfirmOrderTx(tx *sql.Tx, orderID string) error {
	res, err := tx.Exec(`UPDATE orders SET status = 'PAID' WHERE id = ? AND status = 'PROCESSING'`, orderID)
	if err != nil {
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if ra != 1 {
		return sql.ErrNoRows // Order not in PROCESSING state
	}
	return nil
}

// GetOrderTotalTx retrieves the order total within a transaction.
func GetOrderTotalTx(tx *sql.Tx, orderID string) (float64, error) {
	var total float64
	err := tx.QueryRow(`SELECT total FROM orders WHERE id = ?`, orderID).Scan(&total)
	return total, err
}

// OrderByIDTx returns order details within a transaction.
func OrderByIDTx(tx *sql.Tx, id string) (userID string, status string, total float64, err error) {
	err = tx.QueryRow(`SELECT user_id, status, total FROM orders WHERE id = ?`, id).Scan(&userID, &status, &total)
	return
}

// OrderItemsByOrderIDTx returns order items within a transaction.
func OrderItemsByOrderIDTx(tx *sql.Tx, orderID string) ([]OrderItemRow, error) {
	rows, err := tx.Query(`SELECT id, order_id, event_date_id, ticket_type_id, quantity, unit_price FROM order_items WHERE order_id = ?`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []OrderItemRow
	for rows.Next() {
		var o OrderItemRow
		if err := rows.Scan(&o.ID, &o.OrderID, &o.EventDateID, &o.TicketTypeID, &o.Quantity, &o.UnitPrice); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// CreateTicketWithIDTx inserts a ticket within a transaction.
func CreateTicketWithIDTx(tx *sql.Tx, id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID string) error {
	_, err := tx.Exec(`INSERT INTO tickets (id, code, qr_code, order_id, order_item_id, user_id, event_id, event_date_id, ticket_type_id, used) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID,
	)
	return err
}

// IncrementTicketTypeSoldTx increments sold_quantity within a transaction.
func IncrementTicketTypeSoldTx(tx *sql.Tx, ticketTypeID string, n int) error {
	_, err := tx.Exec(`UPDATE ticket_types SET sold_quantity = sold_quantity + ? WHERE id = ?`, n, ticketTypeID)
	return err
}

// DecrementLotAvailableTx decrements available_quantity within a transaction.
// Only decrements if sufficient quantity is available (prevents negative values).
func DecrementLotAvailableTx(tx *sql.Tx, lotID string, n int) error {
	res, err := tx.Exec(`UPDATE lots SET available_quantity = available_quantity - ? WHERE id = ? AND available_quantity >= ?`, n, lotID, n)
	if err != nil {
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if ra != 1 {
		return sql.ErrNoRows // Insufficient quantity available
	}
	return err
}

// LotIDByTicketTypeIDTx retrieves lot ID within a transaction.
func LotIDByTicketTypeIDTx(tx *sql.Tx, ticketTypeID string) (string, error) {
	var lotID string
	err := tx.QueryRow(`SELECT lot_id FROM ticket_types WHERE id = ?`, ticketTypeID).Scan(&lotID)
	return lotID, err
}
