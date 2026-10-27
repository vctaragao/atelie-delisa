package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// pedidoDeTeste cria uma cliente e um pedido de um valor conhecido.
func pedidoDeTeste(t *testing.T, s *Store, valorCents int64) Order {
	t.Helper()
	cliente, err := s.CreateClient(Client{Name: "Dona Delisa"})
	if err != nil {
		t.Fatalf("criando cliente: %v", err)
	}
	pedido, err := s.CreateOrder(Order{
		ClientID: cliente.ID,
		Date:     "2026-10-20",
		Status:   "Em produção",
		Items:    []OrderItem{{Name: "Vestido", Qty: 1, PriceCents: valorCents}},
	})
	if err != nil {
		t.Fatalf("criando pedido: %v", err)
	}
	return pedido
}

// TestPagamentoParcialDeixaOSaldoEmAberto é o caso que motivou os pagamentos:
// a cliente deixa a peça, paga um sinal e acerta o resto na entrega. Até aqui
// o pedido era pago ou não pago, e metade do dinheiro não tinha onde aparecer.
func TestPagamentoParcialDeixaOSaldoEmAberto(t *testing.T) {
	s := bancoDeTeste(t)
	pedido := pedidoDeTeste(t, s, 20000)

	pedido, err := s.CreatePayment(Payment{
		OrderID: pedido.ID, Date: "2026-10-20", ValueCents: 5000, Method: "Pix",
	})
	if err != nil {
		t.Fatalf("lançando o sinal: %v", err)
	}

	if pedido.PaidCents != 5000 {
		t.Errorf("pago %d, esperava 5000", pedido.PaidCents)
	}
	if pedido.BalanceCents != 15000 {
		t.Errorf("falta receber %d, esperava 15000", pedido.BalanceCents)
	}
	if pedido.Paid {
		t.Error("um sinal de 50,00 em 200,00 não quita o pedido")
	}
	if len(pedido.Payments) != 1 {
		t.Fatalf("esperava 1 pagamento no pedido, veio %d", len(pedido.Payments))
	}

	// O caixa do ateliê conta os 50,00 que entraram e cobra os 150,00 que
	// faltam — e não o pedido inteiro de um lado só.
	sum, err := s.Summary()
	if err != nil {
		t.Fatalf("calculando o resumo: %v", err)
	}
	if sum.RevenueCents != 5000 {
		t.Errorf("recebido %d, esperava 5000", sum.RevenueCents)
	}
	if sum.ReceivableCents != 15000 {
		t.Errorf("a receber %d, esperava 15000", sum.ReceivableCents)
	}

	// Quitado na entrega.
	pedido, err = s.CreatePayment(Payment{
		OrderID: pedido.ID, Date: "2026-10-24", ValueCents: 15000, Method: "Dinheiro",
	})
	if err != nil {
		t.Fatalf("lançando o resto: %v", err)
	}
	if !pedido.Paid || pedido.BalanceCents != 0 {
		t.Errorf("pedido quitado: paid=%v saldo=%d, esperava true e 0",
			pedido.Paid, pedido.BalanceCents)
	}

	sum, err = s.Summary()
	if err != nil {
		t.Fatalf("recalculando o resumo: %v", err)
	}
	if sum.RevenueCents != 20000 || sum.ReceivableCents != 0 {
		t.Errorf("recebido %d e a receber %d, esperava 20000 e 0",
			sum.RevenueCents, sum.ReceivableCents)
	}
}

// TestExcluirPagamentoReabreOSaldo: baixa lançada errado tem de poder ser
// desfeita, e o pedido volta a dever.
func TestExcluirPagamentoReabreOSaldo(t *testing.T) {
	s := bancoDeTeste(t)
	pedido := pedidoDeTeste(t, s, 10000)

	pedido, err := s.CreatePayment(Payment{
		OrderID: pedido.ID, Date: "2026-10-20", ValueCents: 10000, Method: "Pix",
	})
	if err != nil {
		t.Fatalf("dando baixa: %v", err)
	}
	if !pedido.Paid {
		t.Fatal("o pedido deveria estar quitado antes da exclusão")
	}

	pedido, err = s.DeletePayment(pedido.Payments[0].ID)
	if err != nil {
		t.Fatalf("excluindo o pagamento: %v", err)
	}
	if pedido.Paid || pedido.PaidCents != 0 || pedido.BalanceCents != 10000 {
		t.Errorf("depois de excluir: paid=%v pago=%d saldo=%d, esperava false, 0 e 10000",
			pedido.Paid, pedido.PaidCents, pedido.BalanceCents)
	}
}

// TestEncarecerPedidoQuitadoReabreOSaldo: a conta muda quando ela acrescenta
// um serviço depois. Se o pedido continuasse marcado como pago, a diferença
// sumiria do que o ateliê tem a receber.
func TestEncarecerPedidoQuitadoReabreOSaldo(t *testing.T) {
	s := bancoDeTeste(t)
	pedido := pedidoDeTeste(t, s, 10000)

	pedido, err := s.CreatePayment(Payment{
		OrderID: pedido.ID, Date: "2026-10-20", ValueCents: 10000, Method: "Pix",
	})
	if err != nil {
		t.Fatalf("dando baixa: %v", err)
	}

	pedido.Items = append(pedido.Items, OrderItem{Name: "Barra", Qty: 1, PriceCents: 3000})
	pedido, err = s.UpdateOrder(pedido.ID, pedido)
	if err != nil {
		t.Fatalf("acrescentando serviço: %v", err)
	}

	if pedido.Paid {
		t.Error("o pedido que encareceu não está mais quitado")
	}
	if pedido.BalanceCents != 3000 {
		t.Errorf("falta receber %d, esperava 3000", pedido.BalanceCents)
	}
}

// TestBackupAntigoPreservaOQueEstavaPago: o arquivo exportado antes dos
// pagamentos existirem só tem o sim/não de paid. Restaurar um desses não pode
// zerar o faturamento do ateliê.
func TestBackupAntigoPreservaOQueEstavaPago(t *testing.T) {
	s := bancoDeTeste(t)

	antigo := Backup{
		Version: 1,
		Clients: []Client{{ID: 1, Name: "Dona Delisa"}},
		Orders: []Order{{
			ID: 1, Number: "0001", ClientID: 1, Date: "2026-09-10",
			Status: "Entregue", Payment: "Pix", Paid: true, TotalCents: 7500,
			Items: []OrderItem{{Name: "Bainha", Qty: 1, PriceCents: 7500}},
			// Payments vem nulo: é o que distingue o arquivo antigo do novo.
		}},
	}
	if err := s.Import(antigo); err != nil {
		t.Fatalf("restaurando o backup antigo: %v", err)
	}

	pedido, err := s.GetOrder(1)
	if err != nil {
		t.Fatalf("lendo o pedido restaurado: %v", err)
	}
	if !pedido.Paid || pedido.PaidCents != 7500 {
		t.Errorf("pedido restaurado: paid=%v pago=%d, esperava true e 7500",
			pedido.Paid, pedido.PaidCents)
	}

	sum, err := s.Summary()
	if err != nil {
		t.Fatalf("calculando o resumo: %v", err)
	}
	if sum.RevenueCents != 7500 {
		t.Errorf("recebido %d, esperava 7500", sum.RevenueCents)
	}
}

// TestMigracaoDePagamentosHerdaOQueJaEstavaPago é o teste que protege o banco
// de produção no dia em que esta versão subir.
//
// Lá existem pedidos com paid = 1 e nenhuma linha de pagamento, porque a
// tabela acabou de ser criada. Sem a carga inicial da migração 0006, o ateliê
// abriria o sistema com o faturamento zerado e com todos os pedidos já pagos
// cobrando de novo. Em desenvolvimento o erro seria invisível: o banco de dev
// nasce vazio.
func TestMigracaoDePagamentosHerdaOQueJaEstavaPago(t *testing.T) {
	path := filepath.Join(t.TempDir(), "atelie.db")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("abrindo o banco: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}

	// O banco como estava antes desta versão: até a migração 5.
	if err := goose.UpTo(db, "migrations", 5); err != nil {
		t.Fatalf("migrando até a versão 5: %v", err)
	}

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("preparando o banco antigo: %v", err)
		}
	}
	exec(`INSERT INTO clients (id, name) VALUES (1, 'Ana'), (2, 'Bia')`)
	exec(`INSERT INTO orders (id, number, client_id, date, status, payment, paid)
	      VALUES (1, '0001', 1, '2026-09-10', 'Entregue', 'Pix', 1),
	             (2, '0002', 2, '2026-09-12', 'Pronto', 'Dinheiro', 0)`)
	exec(`INSERT INTO order_items (order_id, name, qty, price_cents)
	      VALUES (1, 'Bainha', 2, 5000), (2, 'Barra', 1, 3000)`)

	if err := goose.Up(db, "migrations"); err != nil {
		t.Fatalf("aplicando a migração dos pagamentos: %v", err)
	}

	// O pedido que estava pago ganhou um recebimento do valor cheio, na data
	// de entrada dele.
	var valor int64
	var data string
	err = db.QueryRow(
		`SELECT value_cents, date FROM order_payments WHERE order_id = 1`).Scan(&valor, &data)
	if err != nil {
		t.Fatalf("o pedido pago não herdou a baixa: %v", err)
	}
	if valor != 10000 {
		t.Errorf("baixa herdada de %d, esperava 10000", valor)
	}
	if data != "2026-09-10" {
		t.Errorf("data da baixa herdada %q, esperava a data de entrada do pedido", data)
	}

	// E o que não estava pago continua devendo.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM order_payments WHERE order_id = 2`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("o pedido em aberto ganhou %d pagamento(s), esperava nenhum", n)
	}
}
