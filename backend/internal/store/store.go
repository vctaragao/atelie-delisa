package store

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Store struct {
	db *sql.DB
}

// Open abre (ou cria) o banco em path e aplica as migrações pendentes.
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
	if err := migrate(db); err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.seedServices(); err != nil {
		return nil, fmt.Errorf("populando serviços iniciais: %w", err)
	}
	return s, nil
}

// migrate aplica as migrações de internal/store/migrations que ainda não
// rodaram neste banco, em ordem de versão. O goose registra cada uma na
// tabela goose_db_version, então um banco existente recebe só o que falta —
// é isso que faz uma coluna nova chegar à produção, e não apenas aos bancos
// criados do zero em desenvolvimento.
//
// Nunca edite uma migração já aplicada: ela não roda de novo, e a alteração
// passaria a existir só em bancos novos. Para mudar algo, crie a próxima.
func migrate(db *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("configurando o goose: %w", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("aplicando migrações: %w", err)
	}
	return nil
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
