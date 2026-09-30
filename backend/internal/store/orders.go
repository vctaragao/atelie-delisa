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
  o.date, o.due, o.status, o.payment, o.paid, o.notes,
  COALESCE(t.total_cents, 0)`

const orderFrom = `
  FROM orders o
  LEFT JOIN clients c ON c.id = o.client_id
  LEFT JOIN order_totals t ON t.order_id = o.id`

func scanOrder(row interface{ Scan(...any) error }) (Order, error) {
	var o Order
	err := row.Scan(
		&o.ID, &o.Number, &o.ClientID, &o.ClientName,
		&o.Date, &o.Due, &o.Status, &o.Payment, &o.Paid, &o.Notes,
		&o.TotalCents,
	)
	o.Items = []OrderItem{}
	return o, err
}

// queryOrders roda uma consulta de pedidos e anexa os itens de cada um.
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
		`SELECT order_id, id, service_id, name, qty, price_cents
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
		if err := rows.Scan(&orderID, &it.ID, &serviceID, &it.Name, &it.Qty, &it.PriceCents); err != nil {
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
		`INSERT INTO orders (number, client_id, date, due, status, payment, paid, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		number, o.ClientID, o.Date, o.Due, o.Status, o.Payment, boolToInt(o.Paid), o.Notes,
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
		`UPDATE orders SET client_id = ?, date = ?, due = ?, status = ?,
		        payment = ?, paid = ?, notes = ? WHERE id = ?`,
		o.ClientID, o.Date, o.Due, o.Status, o.Payment, boolToInt(o.Paid), o.Notes, id,
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
	if err := tx.Commit(); err != nil {
		return Order{}, err
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

// boolToInt converte para o 0/1 que a coluna INTEGER espera. Passar um bool
// direto depende de como o driver o traduz; assim o valor é explícito.
func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
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
			`INSERT INTO order_items (order_id, service_id, name, qty, price_cents, position)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			orderID, serviceID, it.Name, it.Qty, it.PriceCents, i,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
