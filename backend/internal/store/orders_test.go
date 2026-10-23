package store

import (
	"path/filepath"
	"testing"
)

func bancoDeTeste(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "atelie.db"))
	if err != nil {
		t.Fatalf("abrindo o banco de teste: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// TestHorarioDeEntregaSobreviveAoCicloCompleto segue a hora combinada com a
// cliente por todos os caminhos em que ela pode se perder: a gravação, a
// releitura, a edição do pedido e o backup. Era o campo novo da agenda, e um
// INSERT esquecido em qualquer um desses pontos devolveria o pedido com o
// horário em branco sem erro nenhum.
func TestHorarioDeEntregaSobreviveAoCicloCompleto(t *testing.T) {
	s := bancoDeTeste(t)

	cliente, err := s.CreateClient(Client{Name: "Dona Delisa"})
	if err != nil {
		t.Fatalf("criando cliente: %v", err)
	}

	pedido, err := s.CreateOrder(Order{
		ClientID: cliente.ID,
		Date:     "2026-10-22",
		Due:      "2026-10-24",
		DueTime:  "14:30",
		Status:   "Em produção",
		Items:    []OrderItem{{Name: "Bainha de calça", Qty: 2, PriceCents: 2500}},
	})
	if err != nil {
		t.Fatalf("criando pedido: %v", err)
	}
	if pedido.DueTime != "14:30" {
		t.Fatalf("o pedido criado veio com horário %q, esperava 14:30", pedido.DueTime)
	}

	lido, err := s.GetOrder(pedido.ID)
	if err != nil {
		t.Fatalf("relendo o pedido: %v", err)
	}
	if lido.DueTime != "14:30" {
		t.Errorf("relendo o pedido veio %q, esperava 14:30", lido.DueTime)
	}

	lido.DueTime = "09:00"
	atualizado, err := s.UpdateOrder(lido.ID, lido)
	if err != nil {
		t.Fatalf("atualizando o pedido: %v", err)
	}
	if atualizado.DueTime != "09:00" {
		t.Errorf("depois de editar veio %q, esperava 09:00", atualizado.DueTime)
	}

	backup, err := s.Export()
	if err != nil {
		t.Fatalf("exportando o backup: %v", err)
	}
	if err := s.Import(backup); err != nil {
		t.Fatalf("restaurando o backup: %v", err)
	}
	restaurado, err := s.GetOrder(pedido.ID)
	if err != nil {
		t.Fatalf("relendo o pedido restaurado: %v", err)
	}
	if restaurado.DueTime != "09:00" {
		t.Errorf("o backup devolveu o horário %q, esperava 09:00", restaurado.DueTime)
	}
}

// TestProximasEntregasOrdenamPelaHoraDoDia garante a ordem que a agenda e o
// Dashboard mostram: dentro do mesmo dia manda a hora, e quem não marcou hora
// fica no fim do dia — não na frente de quem tem horário combinado.
func TestProximasEntregasOrdenamPelaHoraDoDia(t *testing.T) {
	s := bancoDeTeste(t)

	cliente, err := s.CreateClient(Client{Name: "Dona Delisa"})
	if err != nil {
		t.Fatalf("criando cliente: %v", err)
	}

	criar := func(due, hora string) {
		t.Helper()
		_, err := s.CreateOrder(Order{
			ClientID: cliente.ID,
			Date:     "2026-10-22",
			Due:      due,
			DueTime:  hora,
			Status:   "Em produção",
			Items:    []OrderItem{{Name: "Bainha", Qty: 1, PriceCents: 2500}},
		})
		if err != nil {
			t.Fatalf("criando pedido de %s %s: %v", due, hora, err)
		}
	}

	criar("2026-10-24", "")
	criar("2026-10-24", "16:00")
	criar("2026-10-24", "08:15")
	criar("2026-10-23", "11:00")

	sum, err := s.Summary()
	if err != nil {
		t.Fatalf("calculando o resumo: %v", err)
	}

	esperado := []struct{ due, hora string }{
		{"2026-10-23", "11:00"},
		{"2026-10-24", "08:15"},
		{"2026-10-24", "16:00"},
		{"2026-10-24", ""},
	}
	if len(sum.UpcomingOrders) != len(esperado) {
		t.Fatalf("esperava %d entregas próximas, veio %d", len(esperado), len(sum.UpcomingOrders))
	}
	for i, e := range esperado {
		got := sum.UpcomingOrders[i]
		if got.Due != e.due || got.DueTime != e.hora {
			t.Errorf("posição %d: veio %s %q, esperava %s %q", i, got.Due, got.DueTime, e.due, e.hora)
		}
	}
}

// TestDescontoEUrgenciaFechamAConta fixa a conta do pedido, que é o número
// que a cliente paga e o que entra no financeiro. Cada caso traz o subtotal
// dos serviços e o que sobra depois do desconto e do acréscimo de urgência.
func TestDescontoEUrgenciaFechamAConta(t *testing.T) {
	s := bancoDeTeste(t)

	cliente, err := s.CreateClient(Client{Name: "Dona Delisa"})
	if err != nil {
		t.Fatalf("criando cliente: %v", err)
	}

	casos := []struct {
		nome                     string
		qty, preco               int64
		desconto, urgencia       int64
		itens, desc, rush, total int64
	}{
		{"sem desconto nem urgencia", 2, 5000, 0, 0, 10000, 0, 0, 10000},
		{"só desconto", 2, 5000, 10, 0, 10000, 1000, 0, 9000},
		{"só urgência", 2, 5000, 0, 20, 10000, 0, 2000, 12000},
		{"os dois juntos", 2, 5000, 10, 20, 10000, 1000, 2000, 11000},
		// Os percentuais incidem sobre o subtotal, não um sobre o outro: 10% e
		// 20% de 100,00 são 10,00 e 20,00, e não 10,00 e 18,00.
		{"meio centavo arredonda para cima", 1, 333, 5, 0, 333, 17, 0, 316},
		{"cortesia de 100%", 1, 4000, 100, 0, 4000, 4000, 0, 0},
	}

	for _, c := range casos {
		pedido, err := s.CreateOrder(Order{
			ClientID:        cliente.ID,
			Date:            "2026-10-22",
			Status:          "Em produção",
			DiscountPercent: c.desconto,
			RushPercent:     c.urgencia,
			Items:           []OrderItem{{Name: "Bainha", Qty: c.qty, PriceCents: c.preco}},
		})
		if err != nil {
			t.Fatalf("%s: criando pedido: %v", c.nome, err)
		}
		if pedido.ItemsCents != c.itens {
			t.Errorf("%s: subtotal %d, esperava %d", c.nome, pedido.ItemsCents, c.itens)
		}
		if pedido.DiscountCents != c.desc {
			t.Errorf("%s: desconto %d, esperava %d", c.nome, pedido.DiscountCents, c.desc)
		}
		if pedido.RushCents != c.rush {
			t.Errorf("%s: urgência %d, esperava %d", c.nome, pedido.RushCents, c.rush)
		}
		if pedido.TotalCents != c.total {
			t.Errorf("%s: total %d, esperava %d", c.nome, pedido.TotalCents, c.total)
		}
	}
}

// TestFinanceiroContaOTotalComDesconto: o valor a receber é o que a cliente
// vai pagar. Antes do desconto existir, bastava somar os itens; agora, somar
// os itens prometeria ao ateliê um dinheiro que não vai entrar.
func TestFinanceiroContaOTotalComDesconto(t *testing.T) {
	s := bancoDeTeste(t)

	cliente, err := s.CreateClient(Client{Name: "Dona Delisa"})
	if err != nil {
		t.Fatalf("criando cliente: %v", err)
	}

	pedido, err := s.CreateOrder(Order{
		ClientID:        cliente.ID,
		Date:            "2026-10-22",
		Status:          "Em produção",
		DiscountPercent: 10,
		RushPercent:     20,
		Items:           []OrderItem{{Name: "Vestido", Qty: 1, PriceCents: 20000}},
	})
	if err != nil {
		t.Fatalf("criando pedido: %v", err)
	}

	sum, err := s.Summary()
	if err != nil {
		t.Fatalf("calculando o resumo: %v", err)
	}
	if sum.ReceivableCents != 22000 {
		t.Errorf("a receber %d, esperava 22000 (200,00 - 10%% + 20%%)", sum.ReceivableCents)
	}

	// Pago, o mesmo valor líquido vira faturamento.
	pedido.Paid = true
	if _, err := s.UpdateOrder(pedido.ID, pedido); err != nil {
		t.Fatalf("marcando como pago: %v", err)
	}
	sum, err = s.Summary()
	if err != nil {
		t.Fatalf("calculando o resumo de novo: %v", err)
	}
	if sum.RevenueCents != 22000 {
		t.Errorf("faturamento %d, esperava 22000", sum.RevenueCents)
	}
	if sum.ReceivableCents != 0 {
		t.Errorf("a receber %d, esperava 0", sum.ReceivableCents)
	}
}

// TestBackupPreservaDescontoEUrgencia: os percentuais ficam no pedido, e o
// backup é a única cópia dos dados do ateliê. Se eles não forem exportados, a
// restauração devolve todos os pedidos pelo valor cheio.
func TestBackupPreservaDescontoEUrgencia(t *testing.T) {
	s := bancoDeTeste(t)

	cliente, err := s.CreateClient(Client{Name: "Dona Delisa"})
	if err != nil {
		t.Fatalf("criando cliente: %v", err)
	}
	pedido, err := s.CreateOrder(Order{
		ClientID:        cliente.ID,
		Date:            "2026-10-22",
		Status:          "Pronto",
		DiscountPercent: 15,
		RushPercent:     25,
		Items:           []OrderItem{{Name: "Calça", Qty: 1, PriceCents: 10000}},
	})
	if err != nil {
		t.Fatalf("criando pedido: %v", err)
	}

	backup, err := s.Export()
	if err != nil {
		t.Fatalf("exportando: %v", err)
	}
	if err := s.Import(backup); err != nil {
		t.Fatalf("restaurando: %v", err)
	}

	restaurado, err := s.GetOrder(pedido.ID)
	if err != nil {
		t.Fatalf("relendo o pedido: %v", err)
	}
	if restaurado.DiscountPercent != 15 || restaurado.RushPercent != 25 {
		t.Errorf("percentuais vieram %d%% e %d%%, esperava 15%% e 25%%",
			restaurado.DiscountPercent, restaurado.RushPercent)
	}
	if restaurado.TotalCents != 11000 {
		t.Errorf("total restaurado %d, esperava 11000", restaurado.TotalCents)
	}
}
