package repository

import (
	"database/sql"
	"time"

	"afterzin/api/internal/logger"

	"github.com/google/uuid"
)

func CreateOrder(db *sql.DB, userID string, total float64, exp time.Duration) (string, error) {
	id := uuid.New().String()
	expAt := time.Now().Add(exp).UTC().Format(time.RFC3339)
	logger.Debugf("criando pedido: id=%s usuario=%s total=%.2f expAt=%s", id, userID, total, expAt)
	_, err := db.Exec(`INSERT INTO orders (id, user_id, status, total, expires_at) VALUES (?, ?, 'PENDING', ?, ?)`, id, userID, total, expAt)
	if err != nil {
		logger.Errorf("erro ao criar pedido: %v", err)
	} else {
		logger.Infof("pedido criado com sucesso: id=%s", id)
	}
	return id, err
}

func OrderByID(db *sql.DB, id string) (userID string, status string, total float64, err error) {
	logger.Debugf("buscando pedido por id: %s", id)
	err = db.QueryRow(`SELECT user_id, status, total FROM orders WHERE id = ?`, id).Scan(&userID, &status, &total)
	if err != nil {
		logger.Errorf("erro ao buscar pedido %s: %v", id, err)
	} else {
		logger.Infof("pedido encontrado: usuario=%s status=%s total=%.2f", userID, status, total)
	}
	return
}

func ConfirmOrder(db *sql.DB, orderID string) error {
	logger.Debugf("confirmando pedido: id=%s", orderID)
	_, err := db.Exec(`UPDATE orders SET status = 'PAID' WHERE id = ? AND status IN ('PENDING','PROCESSING')`, orderID)
	if err != nil {
		logger.Errorf("erro ao confirmar pedido %s: %v", orderID, err)
	} else {
		logger.Infof("pedido confirmado: id=%s", orderID)
	}
	return err
}

// ClaimOrderProcessing atomically marks an order as PROCESSING if it's currently PENDING.
// Returns true if the claim succeeded (rows affected == 1).
func ClaimOrderProcessing(db *sql.DB, orderID string) (bool, error) {
	logger.Debugf("reivindicando processamento do pedido: id=%s", orderID)
	res, err := db.Exec(`UPDATE orders SET status = 'PROCESSING' WHERE id = ? AND status = 'PENDING'`, orderID)
	if err != nil {
		logger.Errorf("erro ao reivindicar processamento do pedido %s: %v", orderID, err)
		return false, err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		logger.Errorf("erro ao obter linhas afetadas ao reivindicar pedido %s: %v", orderID, err)
		return false, err
	}
	logger.Debugf("linhas afetadas ao reivindicar pedido: %d", ra)
	return ra == 1, nil
}

func CreateOrderItem(db *sql.DB, orderID, eventDateID, ticketTypeID string, quantity int, unitPrice float64) (string, error) {
	id := uuid.New().String()
	logger.Debugf("criando item do pedido: id=%s pedido=%s dataEvento=%s tipoIngresso=%s quantidade=%d precoUnitario=%.2f", id, orderID, eventDateID, ticketTypeID, quantity, unitPrice)
	_, err := db.Exec(`INSERT INTO order_items (id, order_id, event_date_id, ticket_type_id, quantity, unit_price) VALUES (?, ?, ?, ?, ?, ?)`,
		id, orderID, eventDateID, ticketTypeID, quantity, unitPrice,
	)
	if err != nil {
		logger.Errorf("erro ao criar item do pedido: %v", err)
	} else {
		logger.Infof("item do pedido criado: id=%s", id)
	}
	return id, err
}

func OrderItemsByOrderID(db *sql.DB, orderID string) ([]OrderItemRow, error) {
	logger.Debugf("buscando itens do pedido: pedido=%s", orderID)
	rows, err := db.Query(`SELECT id, order_id, event_date_id, ticket_type_id, quantity, unit_price FROM order_items WHERE order_id = ?`, orderID)
	if err != nil {
		logger.Errorf("erro ao buscar itens do pedido %s: %v", orderID, err)
		return nil, err
	}
	defer rows.Close()
	var list []OrderItemRow
	for rows.Next() {
		var o OrderItemRow
		if err := rows.Scan(&o.ID, &o.OrderID, &o.EventDateID, &o.TicketTypeID, &o.Quantity, &o.UnitPrice); err != nil {
			logger.Errorf("erro ao ler item do pedido: %v", err)
			return nil, err
		}
		logger.Debugf("item do pedido encontrado: %+v", o)
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
	logger.Debugf("criando ingresso: id=%s codigo=%s pedido=%s itemPedido=%s usuario=%s evento=%s dataEvento=%s tipoIngresso=%s", id, code, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID)
	_, err := db.Exec(`INSERT INTO tickets (id, code, qr_code, order_id, order_item_id, user_id, event_id, event_date_id, ticket_type_id, used) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID,
	)
	if err != nil {
		logger.Errorf("erro ao criar ingresso: %v", err)
	} else {
		logger.Infof("ingresso criado: id=%s", id)
	}
	return id, err
}

// CreateTicketWithID inserts a ticket with the given id and qr_code (e.g. signed payload). Used when QR is generated from ticket id.
func CreateTicketWithID(db *sql.DB, id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID string) error {
	logger.Debugf("criando ingresso com id fornecido: id=%s codigo=%s pedido=%s itemPedido=%s usuario=%s evento=%s dataEvento=%s tipoIngresso=%s", id, code, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID)
	_, err := db.Exec(`INSERT INTO tickets (id, code, qr_code, order_id, order_item_id, user_id, event_id, event_date_id, ticket_type_id, used) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID,
	)
	if err != nil {
		logger.Errorf("erro ao criar ingresso com id: %v", err)
	} else {
		logger.Infof("ingresso criado com id: id=%s", id)
	}
	return err
}

func IncrementTicketTypeSold(db *sql.DB, ticketTypeID string, n int) error {
	logger.Debugf("incrementando contagem vendida: tipoIngresso=%s n=%d", ticketTypeID, n)
	_, err := db.Exec(`UPDATE ticket_types SET sold_quantity = sold_quantity + ? WHERE id = ?`, n, ticketTypeID)
	if err != nil {
		logger.Errorf("erro ao incrementar vendidos: %v", err)
	} else {
		logger.Debugf("vendidos incrementados para tipo de ingresso: %s", ticketTypeID)
	}
	return err
}

func DecrementLotAvailable(db *sql.DB, lotID string, n int) error {
	logger.Debugf("decrementando disponível no lote: lote=%s n=%d", lotID, n)
	_, err := db.Exec(`UPDATE lots SET available_quantity = available_quantity - ? WHERE id = ? AND available_quantity >= ?`, n, lotID, n)
	if err != nil {
		logger.Errorf("erro ao decrementar disponível no lote: %v", err)
	} else {
		logger.Debugf("lote decrementado: %s", lotID)
	}
	return err
}

func LotIDByTicketTypeID(db *sql.DB, ticketTypeID string) (string, error) {
	logger.Debugf("buscando lote para tipo de ingresso: %s", ticketTypeID)
	var lotID string
	err := db.QueryRow(`SELECT lot_id FROM ticket_types WHERE id = ?`, ticketTypeID).Scan(&lotID)
	if err != nil {
		logger.Errorf("erro ao buscar lote para tipo %s: %v", ticketTypeID, err)
	} else {
		logger.Debugf("lote encontrado %s para tipoIngresso=%s", lotID, ticketTypeID)
	}
	return lotID, err
}

// ---------- Transactional versions ----------

// ClaimOrderProcessingTx atomically marks an order as PROCESSING if it's currently PENDING.
// Returns true if the claim succeeded (rows affected == 1).
// This provides an optimistic lock to prevent race conditions in webhook processing.
func ClaimOrderProcessingTx(tx *sql.Tx, orderID string) (bool, error) {
	logger.Debugf("reivindicando processamento do pedido (tx): id=%s", orderID)
	res, err := tx.Exec(`UPDATE orders SET status = 'PROCESSING' WHERE id = ? AND status = 'PENDING'`, orderID)
	if err != nil {
		logger.Errorf("erro ao reivindicar processamento do pedido (tx) %s: %v", orderID, err)
		return false, err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		logger.Errorf("erro ao obter linhas afetadas (tx) para pedido %s: %v", orderID, err)
		return false, err
	}
	logger.Debugf("linhas afetadas (tx): %d", ra)
	return ra == 1, nil
}

// ConfirmOrderTx confirms an order within a transaction.
// Only updates if order is in PROCESSING state to ensure proper state transition.
func ConfirmOrderTx(tx *sql.Tx, orderID string) error {
	logger.Debugf("confirmando pedido (tx): id=%s", orderID)
	res, err := tx.Exec(`UPDATE orders SET status = 'PAID' WHERE id = ? AND status = 'PROCESSING'`, orderID)
	if err != nil {
		logger.Errorf("erro ao confirmar pedido (tx) %s: %v", orderID, err)
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		logger.Errorf("erro ao obter linhas afetadas ao confirmar pedido (tx) %s: %v", orderID, err)
		return err
	}
	logger.Debugf("linhas afetadas ao confirmar (tx): %d", ra)
	if ra != 1 {
		logger.Warnf("pedido não está em estado PROCESSING: id=%s", orderID)
		return sql.ErrNoRows // Order not in PROCESSING state
	}
	logger.Infof("pedido confirmado (tx): id=%s", orderID)
	return nil
}

// GetOrderTotalTx retrieves the order total within a transaction.
func GetOrderTotalTx(tx *sql.Tx, orderID string) (float64, error) {
	logger.Debugf("obtendo total do pedido (tx): %s", orderID)
	var total float64
	err := tx.QueryRow(`SELECT total FROM orders WHERE id = ?`, orderID).Scan(&total)
	if err != nil {
		logger.Errorf("erro ao obter total do pedido (tx) %s: %v", orderID, err)
	} else {
		logger.Debugf("total do pedido (tx): %.2f", total)
	}
	return total, err
}

// OrderByIDTx returns order details within a transaction.
func OrderByIDTx(tx *sql.Tx, id string) (userID string, status string, total float64, err error) {
	logger.Debugf("buscando pedido por id (tx): %s", id)
	err = tx.QueryRow(`SELECT user_id, status, total FROM orders WHERE id = ?`, id).Scan(&userID, &status, &total)
	if err != nil {
		logger.Errorf("erro ao buscar pedido (tx) %s: %v", id, err)
	} else {
		logger.Debugf("pedido encontrado (tx): usuario=%s status=%s total=%.2f", userID, status, total)
	}
	return
}

// OrderItemsByOrderIDTx returns order items within a transaction.
func OrderItemsByOrderIDTx(tx *sql.Tx, orderID string) ([]OrderItemRow, error) {
	logger.Debugf("buscando itens do pedido (tx): pedido=%s", orderID)
	rows, err := tx.Query(`SELECT id, order_id, event_date_id, ticket_type_id, quantity, unit_price FROM order_items WHERE order_id = ?`, orderID)
	if err != nil {
		logger.Errorf("erro ao buscar itens do pedido (tx) %s: %v", orderID, err)
		return nil, err
	}
	defer rows.Close()
	var list []OrderItemRow
	for rows.Next() {
		var o OrderItemRow
		if err := rows.Scan(&o.ID, &o.OrderID, &o.EventDateID, &o.TicketTypeID, &o.Quantity, &o.UnitPrice); err != nil {
			logger.Errorf("erro ao ler item do pedido (tx): %v", err)
			return nil, err
		}
		logger.Debugf("item do pedido (tx) encontrado: %+v", o)
		list = append(list, o)
	}
	return list, rows.Err()
}

// CreateTicketWithIDTx inserts a ticket within a transaction.
func CreateTicketWithIDTx(tx *sql.Tx, id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID string) error {
	logger.Debugf("criando ingresso (tx) com id: id=%s codigo=%s pedido=%s itemPedido=%s usuario=%s evento=%s dataEvento=%s tipoIngresso=%s", id, code, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID)
	_, err := tx.Exec(`INSERT INTO tickets (id, code, qr_code, order_id, order_item_id, user_id, event_id, event_date_id, ticket_type_id, used) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		id, code, qrCode, orderID, orderItemID, userID, eventID, eventDateID, ticketTypeID,
	)
	if err != nil {
		logger.Errorf("erro ao criar ingresso (tx): %v", err)
	} else {
		logger.Infof("ingresso criado (tx): id=%s", id)
	}
	return err
}

// IncrementTicketTypeSoldTx increments sold_quantity within a transaction.
func IncrementTicketTypeSoldTx(tx *sql.Tx, ticketTypeID string, n int) error {
	logger.Debugf("incrementando vendidos (tx): tipoIngresso=%s n=%d", ticketTypeID, n)
	_, err := tx.Exec(`UPDATE ticket_types SET sold_quantity = sold_quantity + ? WHERE id = ?`, n, ticketTypeID)
	if err != nil {
		logger.Errorf("erro ao incrementar vendidos (tx): %v", err)
	} else {
		logger.Debugf("vendidos incrementados (tx) para tipo: %s", ticketTypeID)
	}
	return err
}

// DecrementLotAvailableTx decrements available_quantity within a transaction.
// Only decrements if sufficient quantity is available (prevents negative values).
func DecrementLotAvailableTx(tx *sql.Tx, lotID string, n int) error {
	logger.Debugf("decrementando disponível no lote (tx): lote=%s n=%d", lotID, n)
	res, err := tx.Exec(`UPDATE lots SET available_quantity = available_quantity - ? WHERE id = ? AND available_quantity >= ?`, n, lotID, n)
	if err != nil {
		logger.Errorf("erro ao decrementar disponível no lote (tx): %v", err)
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		logger.Errorf("erro ao obter linhas afetadas (tx) no lote %s: %v", lotID, err)
		return err
	}
	logger.Debugf("linhas afetadas (tx) no lote: %d", ra)
	if ra != 1 {
		logger.Warnf("quantidade insuficiente no lote: %s", lotID)
		return sql.ErrNoRows // Insufficient quantity available
	}
	logger.Debugf("lote decrementado (tx): %s", lotID)
	return err
}

// LotIDByTicketTypeIDTx retrieves lot ID within a transaction.
func LotIDByTicketTypeIDTx(tx *sql.Tx, ticketTypeID string) (string, error) {
	logger.Debugf("buscando lote (tx) para tipo de ingresso: %s", ticketTypeID)
	var lotID string
	err := tx.QueryRow(`SELECT lot_id FROM ticket_types WHERE id = ?`, ticketTypeID).Scan(&lotID)
	if err != nil {
		logger.Errorf("erro ao buscar lote (tx) para tipo %s: %v", ticketTypeID, err)
	} else {
		logger.Debugf("lote encontrado (tx) %s para tipoIngresso=%s", lotID, ticketTypeID)
	}
	return lotID, err
}
