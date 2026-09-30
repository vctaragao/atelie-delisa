package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// TestMigracaoAlcancaBancoQueJaExiste cobre o caso que motivou adotar o
// goose. Antes, o schema.sql era aplicado inteiro a cada start com
// CREATE TABLE IF NOT EXISTS: uma coluna nova nunca chegava a um banco que
// já tinha a tabela. Em desenvolvimento passava despercebido, porque lá o
// banco nasce do zero; em produção, sobre os dados reais, quebrava.
func TestMigracaoAlcancaBancoQueJaExiste(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "atelie.db")

	// Banco criado antes da migração nova, e com dado dentro — como o de
	// produção no dia em que uma alteração de schema for pedida.
	s, err := Open(path)
	if err != nil {
		t.Fatalf("abrindo o banco: %v", err)
	}
	if _, err := s.db.Exec(`INSERT INTO clients (name) VALUES ('Dona Delisa')`); err != nil {
		t.Fatalf("inserindo cliente: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("fechando o banco: %v", err)
	}

	// Migração criada depois, adicionando uma coluna à tabela existente.
	migDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		t.Fatal(err)
	}
	nova := "-- +goose Up\nALTER TABLE clients ADD COLUMN apelido TEXT NOT NULL DEFAULT '';\n"
	if err := os.WriteFile(filepath.Join(migDir, "0002_adiciona_apelido.sql"), []byte(nova), 0o644); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("reabrindo o banco: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(nil)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Up(db, migDir); err != nil {
		t.Fatalf("aplicando a migração nova: %v", err)
	}

	// A coluna precisa existir no banco antigo, e o dado que já estava lá
	// precisa ter sobrevivido.
	var nome, apelido string
	err = db.QueryRow(`SELECT name, apelido FROM clients WHERE name = 'Dona Delisa'`).Scan(&nome, &apelido)
	if err != nil {
		t.Fatalf("a coluna nova nao chegou ao banco existente: %v", err)
	}
	if nome != "Dona Delisa" {
		t.Errorf("cliente esperado 'Dona Delisa', veio %q", nome)
	}
	if apelido != "" {
		t.Errorf("apelido deveria vir com o default vazio, veio %q", apelido)
	}
}

// TestMigracoesSaoIdempotentes garante que subir o backend várias vezes não
// reaplica o que já rodou — é o que acontece a cada restart do container.
func TestMigracoesSaoIdempotentes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "atelie.db")

	for i := 1; i <= 3; i++ {
		s, err := Open(path)
		if err != nil {
			t.Fatalf("abertura %d falhou: %v", i, err)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}

	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM goose_db_version WHERE version_id > 0`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("esperava 1 migracao registrada, veio %d", n)
	}

	// seedServices tambem nao pode duplicar a tabela de precos.
	var servicos int
	if err := db.QueryRow(`SELECT COUNT(*) FROM services`).Scan(&servicos); err != nil {
		t.Fatal(err)
	}
	if servicos != 6 {
		t.Errorf("esperava os 6 servicos padrao, veio %d", servicos)
	}
}
