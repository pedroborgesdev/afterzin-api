package repository

import (
	"database/sql"
	"log"
	"time"

	"github.com/google/uuid"
)

func CreateOrder(db *sql.DB, userID string, total float64, exp time.Duration) (string, error) {
	id := uuid.New().String()
	expAt := time.Now().Add(exp).UTC().Format(time.RFC3339)
	log.Printf("[CreateOrder] Creating order: id=%s userID=%s total=%.2f expAt=%s", id, userID, total, expAt)
	_, err := db.Exec(`INSERT INTO orders (id, user_id, status, total, expires_at) VALUES (?, ?, 'PENDING', ?, ?)`, id, userID, total, expAt)
	if err != nil {
		log.Printf("[CreateOrder] Error: %v", err)
	} else {
		log.Printf("[CreateOrder] Order created successfully: id=%s", id)
	}
	return id, err
}

func OrderByID(db *sql.DB, id string) (userID string, status string, total float64, err error) {
	log.Printf("[OrderByID] Fetching order by id: %s", id)
	err = db.QueryRow(`SELECT user_id, status, total FROM orders WHERE id = ?`, id).Scan(&userID, &status, &total)
	if err != nil {
		log.Printf("[OrderByID] Error: %v", err)
	} else {
		log.Printf("[OrderByID] Found: userID=%s status=%s total=%.2f", userID, status, total)
	}
	return
}

func ConfirmOrder(db *sql.DB, orderID string) error {
	log.Printf("[ConfirmOrder] Confirming order: id=%s", orderID)
	_, err := db.Exec(`UPDATE orders SET status = 'PAID' WHERE id = ? AND status IN ('PENDING','PROCESSING')`, orderID)
	if err != nil {
		log.Printf("[ConfirmOrder] Error: %v", err)
	} else {
		log.Printf("[ConfirmOrder] Order confirmed: id=%s", orderID)
	}
	return err
}

// ClaimOrderProcessing atomically marks an order as PROCESSING if it's currently PENDING.
// Returns true if the claim succeeded (rows affected == 1).
func ClaimOrderProcessing(db *sql.DB, orderID string) (bool, error) {
	log.Printf("[ClaimOrderProcessing] Claiming processing for order: id=%s", orderID)
	res, err := db.Exec(`UPDATE orders SET status = 'PROCESSING' WHERE id = ? AND status = 'PENDING'`, orderID)
	if err != nil {
		log.Printf("[ClaimOrderProcessing] Error: %v", err)
		return false, err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		log.Printf("[ClaimOrderProcessing] Error getting rows affected: %v", err)
		return false, err
	}
	log.Printf("[ClaimOrderProcessing] Rows affected: %d", ra)
	return ra == 1, nil
}

func CreateOrderItem(db *sql.DB, orderID, eventDateID, ticketTypeID string, quantity int, unitPrice float64) (string, error) {
	id := uuid.New().String()
	log.Printf("[CreateOrderItem] Creating order item: id=%s orderID=%s eventDateID=%s ticketTypeID=%s quantity=%d unitPrice=%.2f", id, orderID, eventDateID, ticketTypeID, quantity, unitPrice)
	_, err := db.Exec(`INSERT INTO order_items (id, order_id, event_date_id, ticket_type_id, quantity, unit_price) VALUES (?, ?, ?, ?, ?, ?)`,
		id, orderID, eventDateID, ticketTypeID, quantity, unitPrice,
	)
	if err != nil {
		log.Printf("[CreateOrderItem] Error: %v", err)
	} else {
		log.Printf("[CreateOrderItem] Order item created: id=%s", id)
	}
	return id, err
}

func OrderItemsByOrderID(db *sql.DB, orderID string) ([]OrderItemRow, error) {
	log.Printf("[OrderItemsByOrderID] Fetching order items for orderID=%s", orderID)
	rows, err := db.Query(`SELECT id, order_id, event_date_id, ticket_type_id, quantity, unit_price FROM order_items WHERE order_id = ?`, orderID)
	if err != nil {
		log.Printf("[OrderItemsByOrderID] Error: %v", err)
		return nil, err
	}
	defer rows.Close()
	var list []OrderItemRow
	for rows.Next() {
		var o OrderItemRow
		if err := rows.Scan(&o.ID, &o.OrderID, &o.EventDateID, &o.TicketTypeID, &o.Quantity, &o.UnitPrice); err != nil {
			log.Printf("[OrderItemsByOrderID] Error scanning row: %v", err)
			return nil, err
		}
		log.Printf("[OrderItemsByOrderID] Found item: %+v", o)
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
	log.Printf("[CreateTicket] Creating ticket: id=%s code=%s orderID=%s orderItemID=%s userID=%s eventID=%s eventDateID=%s ticketTypeID=%s", id, code, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID)
	_, err := db.Exec(`INSERT INTO tickets (id, code, qr_code, order_id, order_item_id, user_id, event_id, event_date_id, ticket_type_id, used) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID,
	)
	if err != nil {
		log.Printf("[CreateTicket] Error: %v", err)
	} else {
		log.Printf("[CreateTicket] Ticket created: id=%s", id)
	}
	return id, err
}

// CreateTicketWithID inserts a ticket with the given id and qr_code (e.g. signed payload). Used when QR is generated from ticket id.
func CreateTicketWithID(db *sql.DB, id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID string) error {
	log.Printf("[CreateTicketWithID] Creating ticket with ID: id=%s code=%s orderID=%s orderItemID=%s userID=%s eventID=%s eventDateID=%s ticketTypeID=%s", id, code, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID)
	_, err := db.Exec(`INSERT INTO tickets (id, code, qr_code, order_id, order_item_id, user_id, event_id, event_date_id, ticket_type_id, used) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID,
	)
	if err != nil {
		log.Printf("[CreateTicketWithID] Error: %v", err)
	} else {
		log.Printf("[CreateTicketWithID] Ticket created: id=%s", id)
	}
	return err
}

func IncrementTicketTypeSold(db *sql.DB, ticketTypeID string, n int) error {
	log.Printf("[IncrementTicketTypeSold] Incrementing sold quantity: ticketTypeID=%s n=%d", ticketTypeID, n)
	_, err := db.Exec(`UPDATE ticket_types SET sold_quantity = sold_quantity + ? WHERE id = ?`, n, ticketTypeID)
	if err != nil {
		log.Printf("[IncrementTicketTypeSold] Error: %v", err)
	} else {
		log.Printf("[IncrementTicketTypeSold] Sold quantity incremented: ticketTypeID=%s", ticketTypeID)
	}
	return err
}

func DecrementLotAvailable(db *sql.DB, lotID string, n int) error {
	log.Printf("[DecrementLotAvailable] Decrementing lot available: lotID=%s n=%d", lotID, n)
	_, err := db.Exec(`UPDATE lots SET available_quantity = available_quantity - ? WHERE id = ? AND available_quantity >= ?`, n, lotID, n)
	if err != nil {
		log.Printf("[DecrementLotAvailable] Error: %v", err)
	} else {
		log.Printf("[DecrementLotAvailable] Lot decremented: lotID=%s", lotID)
	}
	return err
}

func LotIDByTicketTypeID(db *sql.DB, ticketTypeID string) (string, error) {
	log.Printf("[LotIDByTicketTypeID] Fetching lotID for ticketTypeID=%s", ticketTypeID)
	var lotID string
	err := db.QueryRow(`SELECT lot_id FROM ticket_types WHERE id = ?`, ticketTypeID).Scan(&lotID)
	if err != nil {
		log.Printf("[LotIDByTicketTypeID] Error: %v", err)
	} else {
		log.Printf("[LotIDByTicketTypeID] Found lotID=%s for ticketTypeID=%s", lotID, ticketTypeID)
	}
	return lotID, err
}

// ---------- Transactional versions ----------

// ClaimOrderProcessingTx atomically marks an order as PROCESSING if it's currently PENDING.
// Returns true if the claim succeeded (rows affected == 1).
// This provides an optimistic lock to prevent race conditions in webhook processing.
func ClaimOrderProcessingTx(tx *sql.Tx, orderID string) (bool, error) {
	log.Printf("[ClaimOrderProcessingTx] Claiming processing for order: id=%s", orderID)
	res, err := tx.Exec(`UPDATE orders SET status = 'PROCESSING' WHERE id = ? AND status = 'PENDING'`, orderID)
	if err != nil {
		log.Printf("[ClaimOrderProcessingTx] Error: %v", err)
		return false, err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		log.Printf("[ClaimOrderProcessingTx] Error getting rows affected: %v", err)
		return false, err
	}
	log.Printf("[ClaimOrderProcessingTx] Rows affected: %d", ra)
	return ra == 1, nil
}

// ConfirmOrderTx confirms an order within a transaction.
// Only updates if order is in PROCESSING state to ensure proper state transition.
func ConfirmOrderTx(tx *sql.Tx, orderID string) error {
	log.Printf("[ConfirmOrderTx] Confirming order in transaction: id=%s", orderID)
	res, err := tx.Exec(`UPDATE orders SET status = 'PAID' WHERE id = ? AND status = 'PROCESSING'`, orderID)
	if err != nil {
		log.Printf("[ConfirmOrderTx] Error: %v", err)
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		log.Printf("[ConfirmOrderTx] Error getting rows affected: %v", err)
		return err
	}
	log.Printf("[ConfirmOrderTx] Rows affected: %d", ra)
	if ra != 1 {
		log.Printf("[ConfirmOrderTx] Order not in PROCESSING state: id=%s", orderID)
		return sql.ErrNoRows // Order not in PROCESSING state
	}
	log.Printf("[ConfirmOrderTx] Order confirmed: id=%s", orderID)
	return nil
}

// GetOrderTotalTx retrieves the order total within a transaction.
func GetOrderTotalTx(tx *sql.Tx, orderID string) (float64, error) {
	log.Printf("[GetOrderTotalTx] Getting order total: orderID=%s", orderID)
	var total float64
	err := tx.QueryRow(`SELECT total FROM orders WHERE id = ?`, orderID).Scan(&total)
	if err != nil {
		log.Printf("[GetOrderTotalTx] Error: %v", err)
	} else {
		log.Printf("[GetOrderTotalTx] Total: %.2f", total)
	}
	return total, err
}

// OrderByIDTx returns order details within a transaction.
func OrderByIDTx(tx *sql.Tx, id string) (userID string, status string, total float64, err error) {
	log.Printf("[OrderByIDTx] Fetching order by id in transaction: %s", id)
	err = tx.QueryRow(`SELECT user_id, status, total FROM orders WHERE id = ?`, id).Scan(&userID, &status, &total)
	if err != nil {
		log.Printf("[OrderByIDTx] Error: %v", err)
	} else {
		log.Printf("[OrderByIDTx] Found: userID=%s status=%s total=%.2f", userID, status, total)
	}
	return
}

// OrderItemsByOrderIDTx returns order items within a transaction.
func OrderItemsByOrderIDTx(tx *sql.Tx, orderID string) ([]OrderItemRow, error) {
	log.Printf("[OrderItemsByOrderIDTx] Fetching order items for orderID=%s in transaction", orderID)
	rows, err := tx.Query(`SELECT id, order_id, event_date_id, ticket_type_id, quantity, unit_price FROM order_items WHERE order_id = ?`, orderID)
	if err != nil {
		log.Printf("[OrderItemsByOrderIDTx] Error: %v", err)
		return nil, err
	}
	defer rows.Close()
	var list []OrderItemRow
	for rows.Next() {
		var o OrderItemRow
		if err := rows.Scan(&o.ID, &o.OrderID, &o.EventDateID, &o.TicketTypeID, &o.Quantity, &o.UnitPrice); err != nil {
			log.Printf("[OrderItemsByOrderIDTx] Error scanning row: %v", err)
			return nil, err
		}
		log.Printf("[OrderItemsByOrderIDTx] Found item: %+v", o)
		list = append(list, o)
	}
	return list, rows.Err()
}

// CreateTicketWithIDTx inserts a ticket within a transaction.
func CreateTicketWithIDTx(tx *sql.Tx, id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID string) error {
	log.Printf("[CreateTicketWithIDTx] Creating ticket with ID in transaction: id=%s code=%s orderID=%s orderItemID=%s userID=%s eventID=%s eventDateID=%s ticketTypeID=%s", id, code, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID)
	_, err := tx.Exec(`INSERT INTO tickets (id, code, qr_code, order_id, order_item_id, user_id, event_id, event_date_id, ticket_type_id, used) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID,
	)
	if err != nil {
		log.Printf("[CreateTicketWithIDTx] Error: %v", err)
	} else {
		log.Printf("[CreateTicketWithIDTx] Ticket created: id=%s", id)
	}
	return err
}

// IncrementTicketTypeSoldTx increments sold_quantity within a transaction.
func IncrementTicketTypeSoldTx(tx *sql.Tx, ticketTypeID string, n int) error {
	log.Printf("[IncrementTicketTypeSoldTx] Incrementing sold quantity in transaction: ticketTypeID=%s n=%d", ticketTypeID, n)
	_, err := tx.Exec(`UPDATE ticket_types SET sold_quantity = sold_quantity + ? WHERE id = ?`, n, ticketTypeID)
	if err != nil {
		log.Printf("[IncrementTicketTypeSoldTx] Error: %v", err)
	} else {
		log.Printf("[IncrementTicketTypeSoldTx] Sold quantity incremented: ticketTypeID=%s", ticketTypeID)
	}
	return err
}

// DecrementLotAvailableTx decrements available_quantity within a transaction.
// Only decrements if sufficient quantity is available (prevents negative values).
func DecrementLotAvailableTx(tx *sql.Tx, lotID string, n int) error {
	log.Printf("[DecrementLotAvailableTx] Decrementing lot available in transaction: lotID=%s n=%d", lotID, n)
	res, err := tx.Exec(`UPDATE lots SET available_quantity = available_quantity - ? WHERE id = ? AND available_quantity >= ?`, n, lotID, n)
	if err != nil {
		log.Printf("[DecrementLotAvailableTx] Error: %v", err)
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		log.Printf("[DecrementLotAvailableTx] Error getting rows affected: %v", err)
		return err
	}
	log.Printf("[DecrementLotAvailableTx] Rows affected: %d", ra)
	if ra != 1 {
		log.Printf("[DecrementLotAvailableTx] Insufficient quantity available: lotID=%s", lotID)
		return sql.ErrNoRows // Insufficient quantity available
	}
	log.Printf("[DecrementLotAvailableTx] Lot decremented: lotID=%s", lotID)
	return err
}

// LotIDByTicketTypeIDTx retrieves lot ID within a transaction.
func LotIDByTicketTypeIDTx(tx *sql.Tx, ticketTypeID string) (string, error) {
	log.Printf("[LotIDByTicketTypeIDTx] Fetching lotID for ticketTypeID=%s in transaction", ticketTypeID)
	var lotID string
	err := tx.QueryRow(`SELECT lot_id FROM ticket_types WHERE id = ?`, ticketTypeID).Scan(&lotID)
	if err != nil {
		log.Printf("[LotIDByTicketTypeIDTx] Error: %v", err)
	} else {
		log.Printf("[LotIDByTicketTypeIDTx] Found lotID=%s for ticketTypeID=%s", lotID, ticketTypeID)
	}
	return lotID, err
}
