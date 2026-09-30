package store

import (
	"database/sql"
	"errors"
)

// ErrNotFound indica que o registro pedido não existe.
var ErrNotFound = errors.New("registro não encontrado")

// ErrHasOrders impede apagar um cliente que ainda tem pedidos, para não
// deixar pedidos órfãos no histórico.
var ErrHasOrders = errors.New("cliente possui pedidos")

const clientCols = `
  c.id, c.name, c.phone, c.address, c.notes,
  (SELECT COUNT(*) FROM orders o WHERE o.client_id = c.id)`

func scanClient(row interface{ Scan(...any) error }) (Client, error) {
	var c Client
	err := row.Scan(&c.ID, &c.Name, &c.Phone, &c.Address, &c.Notes, &c.OrderCount)
	return c, err
}

func (s *Store) ListClients() ([]Client, error) {
	rows, err := s.db.Query(`SELECT ` + clientCols + ` FROM clients c ORDER BY c.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Client{}
	for rows.Next() {
		c, err := scanClient(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetClient(id int64) (Client, error) {
	row := s.db.QueryRow(`SELECT `+clientCols+` FROM clients c WHERE c.id = ?`, id)
	c, err := scanClient(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Client{}, ErrNotFound
	}
	return c, err
}

func (s *Store) CreateClient(c Client) (Client, error) {
	res, err := s.db.Exec(
		`INSERT INTO clients (name, phone, address, notes) VALUES (?, ?, ?, ?)`,
		c.Name, c.Phone, c.Address, c.Notes,
	)
	if err != nil {
		return Client{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Client{}, err
	}
	return s.GetClient(id)
}

func (s *Store) UpdateClient(id int64, c Client) (Client, error) {
	res, err := s.db.Exec(
		`UPDATE clients SET name = ?, phone = ?, address = ?, notes = ? WHERE id = ?`,
		c.Name, c.Phone, c.Address, c.Notes, id,
	)
	if err != nil {
		return Client{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Client{}, ErrNotFound
	}
	return s.GetClient(id)
}

func (s *Store) DeleteClient(id int64) error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM orders WHERE client_id = ?`, id).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrHasOrders
	}

	res, err := s.db.Exec(`DELETE FROM clients WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}
