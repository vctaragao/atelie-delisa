package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const orderCols = `
  o.id, o.number, o.client_id,
  COALESCE(c.name, 'Cliente removido'),
  o.date, o.due, o.due_time, o.status, o.payment, o.paid, o.notes,
  o.discount_percent, o.rush_percent,
  COALESCE(t.items_cents, 0), COALESCE(t.discount_cents, 0),
  COALESCE(t.rush_cents, 0), COALESCE(t.total_cents, 0),
  COALESCE((SELECT SUM(p.value_cents) FROM order_payments p WHERE p.order_id = o.id), 0)`

const orderFrom = `
  FROM orders o
  LEFT JOIN clients c ON c.id = o.client_id
  LEFT JOIN order_totals t ON t.order_id = o.id`

func scanOrder(row interface{ Scan(...any) error }) (Order, error) {
	var o Order
	err := row.Scan(
		&o.ID, &o.Number, &o.ClientID, &o.ClientName,
		&o.Date, &o.Due, &o.DueTime, &o.Status, &o.Payment, &o.Paid, &o.Notes,
		&o.DiscountPercent, &o.RushPercent,
		&o.ItemsCents, &o.DiscountCents, &o.RushCents, &o.TotalCents,
		&o.PaidCents,
	)
	// O saldo pode ficar negativo: pagamento a mais existe, e esconder isso
	// faria a tela mentir sobre o que a cliente entregou.
	o.BalanceCents = o.TotalCents - o.PaidCents
	o.Items = []OrderItem{}
	o.Payments = []Payment{}
	return o, err
}

// queryOrders roda uma consulta de pedidos e anexa os itens e os pagamentos
// de cada um.
func (s *Store) queryOrders(where string, args ...any) ([]Order, error) {
	rows, err := s.db.Query(`SELECT `+orderCols+orderFrom+` `+where, args...)
	if err != nil {
		return nil, err
	}

	orders := []Order{}
	byID := map[int64]int{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		byID[o.ID] = len(orders)
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	if len(orders) == 0 {
		return orders, nil
	}
	if err := s.attachItems(orders, byID); err != nil {
		return nil, err
	}
	if err := s.attachPayments(orders, byID); err != nil {
		return nil, err
	}
	return orders, nil
}

// attachItems busca os itens de todos os pedidos numa única consulta, em vez
// de uma por pedido.
func (s *Store) attachItems(orders []Order, byID map[int64]int) error {
	ids := make([]any, 0, len(orders))
	for _, o := range orders {
		ids = append(ids, o.ID)
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")

	rows, err := s.db.Query(
		`SELECT order_id, id, service_id, name, piece, qty, price_cents
		 FROM order_items WHERE order_id IN (`+placeholders+`)
		 ORDER BY order_id, position, id`, ids...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var orderID int64
		var it OrderItem
		var serviceID sql.NullInt64
		if err := rows.Scan(&orderID, &it.ID, &serviceID, &it.Name, &it.Piece, &it.Qty, &it.PriceCents); err != nil {
			return err
		}
		idx, ok := byID[orderID]
		if !ok {
			continue // pedido fora da página consultada
		}
		if serviceID.Valid {
			id := serviceID.Int64
			it.ServiceID = &id
		}
		orders[idx].Items = append(orders[idx].Items, it)
	}
	return rows.Err()
}

// attachPayments busca os pagamentos de todos os pedidos de uma vez, pelo mesmo
// motivo de attachItems: uma consulta por pedido deixaria a lista de pedidos
// lenta à medida que o ateliê trabalha.
func (s *Store) attachPayments(orders []Order, byID map[int64]int) error {
	ids := make([]any, 0, len(orders))
	for _, o := range orders {
		ids = append(ids, o.ID)
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")

	rows, err := s.db.Query(
		`SELECT `+paymentCols+` FROM order_payments
		 WHERE order_id IN (`+placeholders+`)
		 ORDER BY date, id`, ids...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return err
		}
		idx, ok := byID[p.OrderID]
		if !ok {
			continue // pedido fora da página consultada
		}
		orders[idx].Payments = append(orders[idx].Payments, p)
	}
	return rows.Err()
}

func (s *Store) ListOrders() ([]Order, error) {
	return s.queryOrders(`ORDER BY o.date DESC, o.id DESC`)
}

func (s *Store) GetOrder(id int64) (Order, error) {
	list, err := s.queryOrders(`WHERE o.id = ?`, id)
	if err != nil {
		return Order{}, err
	}
	if len(list) == 0 {
		return Order{}, ErrNotFound
	}
	return list[0], nil
}

func (s *Store) CreateOrder(o Order) (Order, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback()

	number, err := nextOrderNumber(tx)
	if err != nil {
		return Order{}, err
	}

	res, err := tx.Exec(
		`INSERT INTO orders (number, client_id, date, due, due_time, status, payment,
		                     notes, discount_percent, rush_percent)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		number, o.ClientID, o.Date, o.Due, o.DueTime, o.Status, o.Payment,
		o.Notes, o.DiscountPercent, o.RushPercent,
	)
	if err != nil {
		return Order{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Order{}, err
	}
	if err := replaceItems(tx, id, o.Items); err != nil {
		return Order{}, err
	}
	if err := tx.Commit(); err != nil {
		return Order{}, err
	}
	return s.GetOrder(id)
}

func (s *Store) UpdateOrder(id int64, o Order) (Order, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`UPDATE orders SET client_id = ?, date = ?, due = ?, due_time = ?, status = ?,
		        payment = ?, notes = ?, discount_percent = ?, rush_percent = ?
		 WHERE id = ?`,
		o.ClientID, o.Date, o.Due, o.DueTime, o.Status, o.Payment,
		o.Notes, o.DiscountPercent, o.RushPercent, id,
	)
	if err != nil {
		return Order{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Order{}, ErrNotFound
	}
	if err := replaceItems(tx, id, o.Items); err != nil {
		return Order{}, err
	}
	// Mexer em preço ou quantidade muda o total, e com ele a conta do que já
	// foi pago: um pedido quitado que encarece volta a ter saldo em aberto.
	if err := syncPaid(tx, id); err != nil {
		return Order{}, err
	}
	if err := tx.Commit(); err != nil {
		return Order{}, err
	}
	return s.GetOrder(id)
}

// SetOrderStatus move o pedido de etapa, sem tocar em nada mais. É o que a
// lista de ordens usa para a Maria mandar um pedido para produção com um
// clique — mandar o pedido inteiro de volta só para trocar o status
// reescreveria os itens à toa.
func (s *Store) SetOrderStatus(id int64, status string) (Order, error) {
	res, err := s.db.Exec(`UPDATE orders SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return Order{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Order{}, ErrNotFound
	}
	return s.GetOrder(id)
}

func (s *Store) DeleteOrder(id int64) error {
	res, err := s.db.Exec(`DELETE FROM orders WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// nextOrderNumber devolve o próximo número sequencial com 4 dígitos. Usa
// MAX em vez de COUNT para não reutilizar o número de um pedido apagado.
func nextOrderNumber(tx *sql.Tx) (string, error) {
	var max int64
	err := tx.QueryRow(`SELECT COALESCE(MAX(CAST(number AS INTEGER)), 0) FROM orders`).Scan(&max)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	return fmt.Sprintf("%04d", max+1), nil
}

// replaceItems troca todos os itens do pedido pelos recebidos. É mais simples
// e previsível do que fazer diff item por item.
func replaceItems(tx *sql.Tx, orderID int64, items []OrderItem) error {
	if _, err := tx.Exec(`DELETE FROM order_items WHERE order_id = ?`, orderID); err != nil {
		return err
	}
	for i, it := range items {
		var serviceID any
		if it.ServiceID != nil {
			serviceID = *it.ServiceID
		}
		_, err := tx.Exec(
			`INSERT INTO order_items (order_id, service_id, name, piece, qty, price_cents, position)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			orderID, serviceID, it.Name, it.Piece, it.Qty, it.PriceCents, i,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
