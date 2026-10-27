package store

import (
	"database/sql"
	"time"
)

// Summary calcula as métricas do Dashboard e do Financeiro.
//
// Regras:
//   - faturamento recebido = pagamentos dos pedidos + lançamentos de entrada
//   - a receber = soma do que falta em cada pedido não cancelado
//   - despesas = lançamentos de saída
//
// Antes dos pagamentos parciais isto era "pedido pago" ou "pedido não pago",
// o pedido inteiro de um lado ou do outro. Agora um pedido de 200,00 com
// 50,00 de sinal entra com 50,00 no recebido e 150,00 no a receber, que é o
// que o caixa do ateliê realmente tem.
func (s *Store) Summary() (Summary, error) {
	var sum Summary

	q := func(dest *int64, query string) error {
		return s.db.QueryRow(query).Scan(dest)
	}

	var paidOrders, manualIn int64
	queries := []struct {
		dest  *int64
		query string
	}{
		{&paidOrders, `
			SELECT COALESCE(SUM(p.value_cents), 0) FROM order_payments p
			JOIN orders o ON o.id = p.order_id
			WHERE o.status <> 'Cancelado'`},
		{&manualIn, `
			SELECT COALESCE(SUM(value_cents), 0) FROM transactions WHERE type = 'entrada'`},
		// O MAX com zero mantém o pedido pago a mais fora da conta: ele não
		// tem saldo negativo a abater do que as outras clientes devem.
		{&sum.ReceivableCents, `
			SELECT COALESCE(SUM(MAX(t.total_cents - COALESCE(
			         (SELECT SUM(p.value_cents) FROM order_payments p
			           WHERE p.order_id = o.id), 0), 0)), 0)
			FROM orders o
			JOIN order_totals t ON t.order_id = o.id
			WHERE o.status <> 'Cancelado'`},
		{&sum.ExpenseCents, `
			SELECT COALESCE(SUM(value_cents), 0) FROM transactions WHERE type = 'saida'`},
	}
	for _, it := range queries {
		if err := q(it.dest, it.query); err != nil {
			return Summary{}, err
		}
	}

	sum.RevenueCents = paidOrders + manualIn
	sum.BalanceCents = sum.RevenueCents - sum.ExpenseCents

	counts := []struct {
		dest  *int
		query string
	}{
		{&sum.InProgress, `SELECT COUNT(*) FROM orders WHERE status NOT IN ('Entregue', 'Cancelado')`},
		{&sum.Ready, `SELECT COUNT(*) FROM orders WHERE status = 'Pronto'`},
	}
	for _, it := range counts {
		if err := s.db.QueryRow(it.query).Scan(it.dest); err != nil {
			return Summary{}, err
		}
	}

	recent, err := s.queryOrders(`ORDER BY o.date DESC, o.id DESC LIMIT 7`)
	if err != nil {
		return Summary{}, err
	}
	sum.RecentOrders = recent

	// Dentro do mesmo dia vale a hora combinada; o pedido sem hora marcada
	// cai no fim do dia, e não antes de quem tem horário.
	upcoming, err := s.queryOrders(
		`WHERE o.due <> '' AND o.status NOT IN ('Entregue', 'Cancelado')
		 ORDER BY o.due ASC, (o.due_time = '') ASC, o.due_time ASC LIMIT 5`)
	if err != nil {
		return Summary{}, err
	}
	sum.UpcomingOrders = upcoming

	return sum, nil
}

// Export monta o backup completo em memória.
func (s *Store) Export() (Backup, error) {
	clients, err := s.ListClients()
	if err != nil {
		return Backup{}, err
	}
	services, err := s.ListServices()
	if err != nil {
		return Backup{}, err
	}
	orders, err := s.ListOrders()
	if err != nil {
		return Backup{}, err
	}
	txs, err := s.ListTransactions()
	if err != nil {
		return Backup{}, err
	}
	messages, err := s.ListMessages()
	if err != nil {
		return Backup{}, err
	}

	// Versão 2: cada pedido leva junto a sua lista de pagamentos. A 1 tinha
	// só o sim/não de paid, e a restauração sabe ler as duas.
	return Backup{
		Version:      2,
		ExportedAt:   time.Now().Format(time.RFC3339),
		Clients:      clients,
		Services:     services,
		Orders:       orders,
		Transactions: txs,
		Messages:     messages,
	}, nil
}

// Import substitui todo o conteúdo do banco pelo backup, numa única
// transação: se qualquer linha falhar, nada é alterado.
//
// Os IDs do arquivo são preservados para que os pedidos continuem ligados aos
// clientes e serviços corretos.
func (s *Store) Import(b Backup) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range []string{"order_payments", "order_items", "orders", "transactions", "services", "clients"} {
		if _, err := tx.Exec(`DELETE FROM ` + table); err != nil {
			return err
		}
	}

	for _, c := range b.Clients {
		_, err := tx.Exec(
			`INSERT INTO clients (id, name, phone, address, notes) VALUES (?, ?, ?, ?, ?)`,
			c.ID, c.Name, c.Phone, c.Address, c.Notes,
		)
		if err != nil {
			return err
		}
	}
	for _, v := range b.Services {
		_, err := tx.Exec(
			`INSERT INTO services (id, name, category, price_cents, time) VALUES (?, ?, ?, ?, ?)`,
			v.ID, v.Name, v.Category, v.PriceCents, v.Time,
		)
		if err != nil {
			return err
		}
	}
	for _, o := range b.Orders {
		_, err := tx.Exec(
			`INSERT INTO orders (id, number, client_id, date, due, due_time, status,
			                     payment, notes, discount_percent, rush_percent)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			o.ID, o.Number, o.ClientID, o.Date, o.Due, o.DueTime, o.Status, o.Payment,
			o.Notes, o.DiscountPercent, o.RushPercent,
		)
		if err != nil {
			return err
		}
		if err := replaceItems(tx, o.ID, o.Items); err != nil {
			return err
		}
		if err := restorePayments(tx, o); err != nil {
			return err
		}
	}
	for _, t := range b.Transactions {
		_, err := tx.Exec(
			`INSERT INTO transactions (id, date, type, description, value_cents, payment)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			t.ID, t.Date, t.Type, t.Description, t.ValueCents, t.Payment,
		)
		if err != nil {
			return err
		}
	}

	// Arquivo gerado antes das mensagens existirem não traz a lista. Apagar as
	// mensagens atuais nesse caso seria perder o texto dela por causa da idade
	// do backup.
	if b.Messages != nil {
		if _, err := tx.Exec(`DELETE FROM message_templates`); err != nil {
			return err
		}
		for _, m := range b.Messages {
			_, err := tx.Exec(
				`INSERT INTO message_templates (id, name, body, suggest_status, position)
				 VALUES (?, ?, ?, ?, ?)`,
				m.ID, m.Name, m.Body, m.SuggestStatus, m.Position,
			)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// restorePayments devolve ao pedido os pagamentos que vieram do backup.
//
// Arquivo gerado antes do controle de pagamentos não traz a lista: traz só o
// sim/não de paid. Nesse caso o pedido pago vira um recebimento do valor
// cheio, do mesmo jeito que a migração 0006 fez com o banco. Sem isso,
// restaurar um backup antigo zeraria o faturamento do ateliê.
func restorePayments(tx *sql.Tx, o Order) error {
	pagamentos := o.Payments
	if pagamentos == nil && o.Paid && o.TotalCents > 0 {
		data := o.Date
		if data == "" {
			data = time.Now().Format("2006-01-02")
		}
		pagamentos = []Payment{{
			Date:       data,
			ValueCents: o.TotalCents,
			Method:     o.Payment,
			Notes:      "Baixa que constava no pedido do backup restaurado",
		}}
	}
	for _, p := range pagamentos {
		_, err := tx.Exec(
			`INSERT INTO order_payments (order_id, date, value_cents, method, notes)
			 VALUES (?, ?, ?, ?, ?)`,
			o.ID, p.Date, p.ValueCents, p.Method, p.Notes,
		)
		if err != nil {
			return err
		}
	}
	return syncPaid(tx, o.ID)
}

// Reset apaga todos os dados e recria a tabela de preços padrão.
func (s *Store) Reset() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, table := range []string{"order_payments", "order_items", "orders", "transactions", "services", "clients"} {
		if _, err := tx.Exec(`DELETE FROM ` + table); err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return s.seedServices()
}
