package store

import "time"

// Summary calcula as métricas do Dashboard e do Financeiro.
//
// Regras herdadas do sistema original:
//   - faturamento recebido = pedidos pagos + lançamentos de entrada
//   - a receber = pedidos não pagos que não estejam cancelados
//   - despesas = lançamentos de saída
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
			SELECT COALESCE(SUM(t.total_cents), 0) FROM orders o
			JOIN order_totals t ON t.order_id = o.id
			WHERE o.paid = 1 AND o.status <> 'Cancelado'`},
		{&manualIn, `
			SELECT COALESCE(SUM(value_cents), 0) FROM transactions WHERE type = 'entrada'`},
		{&sum.ReceivableCents, `
			SELECT COALESCE(SUM(t.total_cents), 0) FROM orders o
			JOIN order_totals t ON t.order_id = o.id
			WHERE o.paid = 0 AND o.status <> 'Cancelado'`},
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

	upcoming, err := s.queryOrders(
		`WHERE o.due <> '' AND o.status NOT IN ('Entregue', 'Cancelado')
		 ORDER BY o.due ASC LIMIT 5`)
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

	return Backup{
		Version:      1,
		ExportedAt:   time.Now().Format(time.RFC3339),
		Clients:      clients,
		Services:     services,
		Orders:       orders,
		Transactions: txs,
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

	for _, table := range []string{"order_items", "orders", "transactions", "services", "clients"} {
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
			`INSERT INTO orders (id, number, client_id, date, due, status, payment, paid, notes)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			o.ID, o.Number, o.ClientID, o.Date, o.Due, o.Status, o.Payment,
			boolToInt(o.Paid), o.Notes,
		)
		if err != nil {
			return err
		}
		if err := replaceItems(tx, o.ID, o.Items); err != nil {
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

	return tx.Commit()
}

// Reset apaga todos os dados e recria a tabela de preços padrão.
func (s *Store) Reset() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, table := range []string{"order_items", "orders", "transactions", "services", "clients"} {
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
