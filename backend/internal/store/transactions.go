package store

import (
	"database/sql"
	"errors"
)

func (s *Store) ListTransactions() ([]Transaction, error) {
	rows, err := s.db.Query(
		`SELECT id, date, type, description, value_cents, payment
		 FROM transactions ORDER BY date DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Transaction{}
	for rows.Next() {
		var t Transaction
		err := rows.Scan(&t.ID, &t.Date, &t.Type, &t.Description, &t.ValueCents, &t.Payment)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTransaction(id int64) (Transaction, error) {
	var t Transaction
	err := s.db.QueryRow(
		`SELECT id, date, type, description, value_cents, payment
		 FROM transactions WHERE id = ?`, id,
	).Scan(&t.ID, &t.Date, &t.Type, &t.Description, &t.ValueCents, &t.Payment)
	if errors.Is(err, sql.ErrNoRows) {
		return Transaction{}, ErrNotFound
	}
	return t, err
}

func (s *Store) CreateTransaction(t Transaction) (Transaction, error) {
	res, err := s.db.Exec(
		`INSERT INTO transactions (date, type, description, value_cents, payment)
		 VALUES (?, ?, ?, ?, ?)`,
		t.Date, t.Type, t.Description, t.ValueCents, t.Payment,
	)
	if err != nil {
		return Transaction{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Transaction{}, err
	}
	return s.GetTransaction(id)
}

func (s *Store) DeleteTransaction(id int64) error {
	res, err := s.db.Exec(`DELETE FROM transactions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
