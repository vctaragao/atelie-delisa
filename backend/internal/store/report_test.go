package store

import "testing"

// TestRelatorioFechaOPeriodo monta um mês pequeno e confere cada número do
// fechamento. É o relatório que a Maria vai olhar para decidir preço e
// compra de material, então os critérios ficam fixados aqui: o pedido entra
// pela data de entrada, o cancelado não vira dinheiro, e o que está fora do
// período não entra em conta nenhuma.
func TestRelatorioFechaOPeriodo(t *testing.T) {
	s := bancoDeTeste(t)

	ana, err := s.CreateClient(Client{Name: "Ana"})
	if err != nil {
		t.Fatalf("criando cliente: %v", err)
	}
	bia, err := s.CreateClient(Client{Name: "Bia"})
	if err != nil {
		t.Fatalf("criando cliente: %v", err)
	}

	// pago aqui significa quitado: o pedido recebe uma baixa do valor cheio,
	// na data de entrada. É o que a Maria faz quando a cliente paga tudo de
	// uma vez.
	criar := func(cliente int64, date, status string, pago bool, desconto, urgencia int64, itens ...OrderItem) {
		t.Helper()
		pedido, err := s.CreateOrder(Order{
			ClientID:        cliente,
			Date:            date,
			Status:          status,
			DiscountPercent: desconto,
			RushPercent:     urgencia,
			Items:           itens,
		})
		if err != nil {
			t.Fatalf("criando pedido de %s: %v", date, err)
		}
		if !pago {
			return
		}
		if _, err := s.CreatePayment(Payment{
			OrderID: pedido.ID, Date: date, ValueCents: pedido.TotalCents, Method: "Pix",
		}); err != nil {
			t.Fatalf("dando baixa no pedido de %s: %v", date, err)
		}
	}

	// Dentro do período.
	criar(ana.ID, "2026-03-10", "Entregue", true, 10, 20,
		OrderItem{Name: "Bainha", Qty: 2, PriceCents: 5000})
	criar(bia.ID, "2026-03-15", "Pronto", false, 0, 0,
		OrderItem{Name: "Barra", Qty: 1, PriceCents: 3000})
	criar(bia.ID, "2026-03-20", "Cancelado", false, 0, 0,
		OrderItem{Name: "Zíper", Qty: 3, PriceCents: 2000})
	// Fora do período, para provar o recorte.
	criar(ana.ID, "2026-04-02", "Entregue", true, 0, 0,
		OrderItem{Name: "Bainha", Qty: 9, PriceCents: 9900})

	if _, err := s.CreateTransaction(Transaction{
		Date: "2026-03-12", Type: "entrada", Description: "Retalhos", ValueCents: 2500,
	}); err != nil {
		t.Fatalf("lançando entrada: %v", err)
	}
	if _, err := s.CreateTransaction(Transaction{
		Date: "2026-03-18", Type: "saida", Description: "Linha", ValueCents: 4000,
	}); err != nil {
		t.Fatalf("lançando saída: %v", err)
	}
	if _, err := s.CreateTransaction(Transaction{
		Date: "2026-04-05", Type: "saida", Description: "Fora do mês", ValueCents: 90000,
	}); err != nil {
		t.Fatalf("lançando saída fora do período: %v", err)
	}

	rep, err := s.Report("2026-03-01", "2026-03-31")
	if err != nil {
		t.Fatalf("montando o relatório: %v", err)
	}

	esperado := []struct {
		nome       string
		got, quero int64
	}{
		{"pedidos", int64(rep.Orders), 2},
		{"peças", rep.Pieces, 3},
		{"clientes atendidas", int64(rep.ClientsServed), 2},
		{"valor dos pedidos", rep.OrdersCents, 14000},
		{"valor médio", rep.AverageCents, 7000},
		{"já pago", rep.PaidCents, 11000},
		{"a receber", rep.ReceivableCents, 3000},
		{"desconto concedido", rep.DiscountCents, 1000},
		{"urgência cobrada", rep.RushCents, 2000},
		{"outras entradas", rep.IncomeCents, 2500},
		{"despesas", rep.ExpenseCents, 4000},
		{"resultado", rep.ResultCents, 9500}, // 110,00 pagos + 25,00 − 40,00
	}
	for _, e := range esperado {
		if e.got != e.quero {
			t.Errorf("%s: veio %d, esperava %d", e.nome, e.got, e.quero)
		}
	}

	// O serviço cancelado não aparece, e a ordem é por valor.
	if len(rep.TopServices) != 2 {
		t.Fatalf("esperava 2 serviços no ranking, veio %d: %+v", len(rep.TopServices), rep.TopServices)
	}
	if rep.TopServices[0].Name != "Bainha" || rep.TopServices[0].Qty != 2 ||
		rep.TopServices[0].TotalCents != 10000 || rep.TopServices[0].Orders != 1 {
		t.Errorf("primeiro serviço veio %+v", rep.TopServices[0])
	}
	if rep.TopServices[1].Name != "Barra" {
		t.Errorf("segundo serviço veio %q, esperava Barra", rep.TopServices[1].Name)
	}

	if len(rep.TopClients) != 2 {
		t.Fatalf("esperava 2 clientes no ranking, veio %d", len(rep.TopClients))
	}
	if rep.TopClients[0].Name != "Ana" || rep.TopClients[0].TotalCents != 11000 {
		t.Errorf("primeira cliente veio %+v", rep.TopClients[0])
	}

	// Todos os status aparecem, inclusive os zerados e o cancelado.
	if len(rep.ByStatus) != len(OrderStatuses) {
		t.Fatalf("esperava %d situações, veio %d", len(OrderStatuses), len(rep.ByStatus))
	}
	porStatus := map[string]int{}
	for _, st := range rep.ByStatus {
		porStatus[st.Status] = st.Orders
	}
	if porStatus["Pronto"] != 1 || porStatus["Entregue"] != 1 ||
		porStatus["Cancelado"] != 1 || porStatus["Orçamento"] != 0 {
		t.Errorf("contagem por situação errada: %+v", rep.ByStatus)
	}
}

// TestRelatorioDePeriodoVazio: um período sem nada precisa devolver zeros e
// listas vazias, não erro — é o que a tela mostra quando ela abre o mês que
// vem ou um mês em que o ateliê ficou fechado.
func TestRelatorioDePeriodoVazio(t *testing.T) {
	s := bancoDeTeste(t)

	rep, err := s.Report("2026-01-01", "2026-01-31")
	if err != nil {
		t.Fatalf("montando o relatório: %v", err)
	}
	if rep.Orders != 0 || rep.OrdersCents != 0 || rep.AverageCents != 0 || rep.ResultCents != 0 {
		t.Errorf("esperava tudo zerado, veio %+v", rep)
	}
	if len(rep.TopServices) != 0 || len(rep.TopClients) != 0 {
		t.Errorf("esperava rankings vazios, veio %+v e %+v", rep.TopServices, rep.TopClients)
	}
	if len(rep.ByStatus) != len(OrderStatuses) {
		t.Errorf("as situações deveriam vir todas, zeradas: %+v", rep.ByStatus)
	}
}
