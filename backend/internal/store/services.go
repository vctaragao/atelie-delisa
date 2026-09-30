package store

import (
	"database/sql"
	"errors"
)

func (s *Store) ListServices() ([]Service, error) {
	rows, err := s.db.Query(
		`SELECT id, name, category, price_cents, time FROM services
		 ORDER BY category COLLATE NOCASE, name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Service{}
	for rows.Next() {
		var v Service
		if err := rows.Scan(&v.ID, &v.Name, &v.Category, &v.PriceCents, &v.Time); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) GetService(id int64) (Service, error) {
	var v Service
	err := s.db.QueryRow(
		`SELECT id, name, category, price_cents, time FROM services WHERE id = ?`, id,
	).Scan(&v.ID, &v.Name, &v.Category, &v.PriceCents, &v.Time)
	if errors.Is(err, sql.ErrNoRows) {
		return Service{}, ErrNotFound
	}
	return v, err
}

func (s *Store) CreateService(v Service) (Service, error) {
	res, err := s.db.Exec(
		`INSERT INTO services (name, category, price_cents, time) VALUES (?, ?, ?, ?)`,
		v.Name, v.Category, v.PriceCents, v.Time,
	)
	if err != nil {
		return Service{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Service{}, err
	}
	return s.GetService(id)
}

func (s *Store) UpdateService(id int64, v Service) (Service, error) {
	res, err := s.db.Exec(
		`UPDATE services SET name = ?, category = ?, price_cents = ?, time = ? WHERE id = ?`,
		v.Name, v.Category, v.PriceCents, v.Time, id,
	)
	if err != nil {
		return Service{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Service{}, ErrNotFound
	}
	return s.GetService(id)
}

// DeleteService remove o serviço da tabela de preços. Os itens de pedidos
// antigos guardam nome e preço próprios, então o histórico não muda — só o
// vínculo service_id vira NULL (ON DELETE SET NULL no schema).
func (s *Store) DeleteService(id int64) error {
	res, err := s.db.Exec(`DELETE FROM services WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
