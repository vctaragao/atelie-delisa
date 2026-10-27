package store

import (
	"database/sql"
	"errors"
)

// Pagamentos do pedido: uma linha por dinheiro recebido.
//
// A soma dos pagamentos é a verdade sobre o que o pedido tem de pago. A
// coluna orders.paid continua existindo e é recalculada a partir dessa soma
// a cada mudança — ver syncPaid e a migração 0006.

const paymentCols = `id, order_id, date, value_cents, method, notes`

func scanPayment(row interface{ Scan(...any) error }) (Payment, error) {
	var p Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.Date, &p.ValueCents, &p.Method, &p.Notes)
	return p, err
}

// ListPayments devolve os pagamentos de um pedido, do mais antigo para o mais
// recente: é a ordem em que o dinheiro entrou.
func (s *Store) ListPayments(orderID int64) ([]Payment, error) {
	rows, err := s.db.Query(
		`SELECT `+paymentCols+` FROM order_payments
		 WHERE order_id = ? ORDER BY date, id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Payment{}
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetPayment(id int64) (Payment, error) {
	p, err := scanPayment(s.db.QueryRow(
		`SELECT `+paymentCols+` FROM order_payments WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Payment{}, ErrNotFound
	}
	return p, err
}

// CreatePayment registra um recebimento e devolve o pedido já com a conta
// refeita. Quem chama é a tela de dar baixa, que precisa mostrar na hora
// quanto ainda falta.
func (s *Store) CreatePayment(p Payment) (Order, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback()

	// O pedido tem de existir antes: sem isto a chave estrangeira acusaria o
	// erro com uma mensagem que não diz nada para quem está na tela.
	var existe int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM orders WHERE id = ?`, p.OrderID).Scan(&existe); err != nil {
		return Order{}, err
	}
	if existe == 0 {
		return Order{}, ErrNotFound
	}

	if _, err := tx.Exec(
		`INSERT INTO order_payments (order_id, date, value_cents, method, notes)
		 VALUES (?, ?, ?, ?, ?)`,
		p.OrderID, p.Date, p.ValueCents, p.Method, p.Notes,
	); err != nil {
		return Order{}, err
	}
	if err := syncPaid(tx, p.OrderID); err != nil {
		return Order{}, err
	}
	if err := tx.Commit(); err != nil {
		return Order{}, err
	}
	return s.GetOrder(p.OrderID)
}

// DeletePayment desfaz uma baixa lançada errado e devolve o pedido refeito.
func (s *Store) DeletePayment(id int64) (Order, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback()

	var orderID int64
	err = tx.QueryRow(`SELECT order_id FROM order_payments WHERE id = ?`, id).Scan(&orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, err
	}

	if _, err := tx.Exec(`DELETE FROM order_payments WHERE id = ?`, id); err != nil {
		return Order{}, err
	}
	if err := syncPaid(tx, orderID); err != nil {
		return Order{}, err
	}
	if err := tx.Commit(); err != nil {
		return Order{}, err
	}
	return s.GetOrder(orderID)
}

// syncPaid recalcula orders.paid a partir da soma dos pagamentos.
//
// A coluna é um resumo, não a verdade — a verdade são as linhas de
// order_payments. Ela é mantida em dia por dois motivos: o código antigo,
// caso esta versão seja revertida, lê só ela; e o backup exportado por uma
// versão antiga continua fazendo sentido.
//
// Pedido de total zero nunca conta como pago: não há o que receber, e marcar
// "pago" um pedido em que ninguém pagou nada confundiria a tela.
func syncPaid(tx *sql.Tx, orderID int64) error {
	_, err := tx.Exec(`
		UPDATE orders SET paid = (
		  SELECT CASE
		    WHEN COALESCE(t.total_cents, 0) <= 0 THEN 0
		    WHEN COALESCE((SELECT SUM(p.value_cents) FROM order_payments p
		                   WHERE p.order_id = orders.id), 0) >= t.total_cents THEN 1
		    ELSE 0
		  END
		  FROM order_totals t WHERE t.order_id = orders.id
		)
		WHERE id = ?`, orderID)
	return err
}
