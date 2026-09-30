package store

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

type Store struct {
	db *sql.DB
}

// Open abre (ou cria) o banco em path e aplica o schema.
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("abrindo banco: %w", err)
	}

	// SQLite é single-writer. Limitar o pool a uma conexão evita
	// "database is locked" sob escrita concorrente.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("conectando ao banco: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("aplicando schema: %w", err)
	}

	s := &Store{db: db}
	if err := s.seedServices(); err != nil {
		return nil, fmt.Errorf("populando serviços iniciais: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// seedServices insere a tabela de preços padrão apenas se ainda não houver
// nenhum serviço cadastrado.
func (s *Store) seedServices() error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM services`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	defaults := []Service{
		{Name: "Bainha de calça", Category: "Ajustes", PriceCents: 2500, Time: "30 min"},
		{Name: "Bainha de vestido", Category: "Ajustes", PriceCents: 3500, Time: "40 min"},
		{Name: "Ajuste de cintura", Category: "Ajustes", PriceCents: 3500, Time: "45 min"},
		{Name: "Ajuste lateral", Category: "Ajustes", PriceCents: 3500, Time: "45 min"},
		{Name: "Troca de zíper", Category: "Consertos", PriceCents: 3500, Time: "40 min"},
		{Name: "Reparo simples", Category: "Consertos", PriceCents: 2000, Time: "20 min"},
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, d := range defaults {
		_, err := tx.Exec(
			`INSERT INTO services (name, category, price_cents, time) VALUES (?, ?, ?, ?)`,
			d.Name, d.Category, d.PriceCents, d.Time,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
