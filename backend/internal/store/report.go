package store

// Relatório de um período: o fechamento que a tela de Relatórios mostra.
//
// Tudo aqui sai de uma data inicial e uma data final, inclusive, comparadas
// com a data de entrada do pedido (orders.date) e com a data do lançamento
// (transactions.date). Pedido sem data de entrada fica de fora, porque não há
// como dizer a que período ele pertence.

// Report monta o fechamento do período de `from` até `to`, ambos no formato
// "2006-01-02" e ambos incluídos.
func (s *Store) Report(from, to string) (Report, error) {
	rep := Report{From: from, To: to}

	// Os valores vêm da view order_totals, que já aplica desconto e urgência.
	// Cancelado fica fora: o pedido existiu, mas não virou dinheiro nenhum.
	err := s.db.QueryRow(`
		SELECT COUNT(*),
		       COUNT(DISTINCT o.client_id),
		       COALESCE(SUM(t.total_cents), 0),
		       COALESCE(SUM(t.discount_cents), 0),
		       COALESCE(SUM(t.rush_cents), 0),
		       COALESCE(SUM(CASE WHEN o.paid = 1 THEN t.total_cents ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN o.paid = 0 THEN t.total_cents ELSE 0 END), 0)
		FROM orders o
		JOIN order_totals t ON t.order_id = o.id
		WHERE o.date BETWEEN ? AND ? AND o.status <> 'Cancelado'`,
		from, to,
	).Scan(
		&rep.Orders, &rep.ClientsServed, &rep.OrdersCents,
		&rep.DiscountCents, &rep.RushCents, &rep.PaidCents, &rep.ReceivableCents,
	)
	if err != nil {
		return Report{}, err
	}

	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(i.qty), 0)
		FROM order_items i
		JOIN orders o ON o.id = i.order_id
		WHERE o.date BETWEEN ? AND ? AND o.status <> 'Cancelado'`,
		from, to,
	).Scan(&rep.Pieces)
	if err != nil {
		return Report{}, err
	}

	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(CASE WHEN type = 'entrada' THEN value_cents ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN type = 'saida' THEN value_cents ELSE 0 END), 0)
		FROM transactions WHERE date BETWEEN ? AND ?`,
		from, to,
	).Scan(&rep.IncomeCents, &rep.ExpenseCents)
	if err != nil {
		return Report{}, err
	}

	if rep.Orders > 0 {
		rep.AverageCents = rep.OrdersCents / int64(rep.Orders)
	}
	rep.ResultCents = rep.PaidCents + rep.IncomeCents - rep.ExpenseCents

	if rep.TopServices, err = s.topServices(from, to); err != nil {
		return Report{}, err
	}
	if rep.TopClients, err = s.topClients(from, to); err != nil {
		return Report{}, err
	}
	if rep.ByStatus, err = s.ordersByStatus(from, to); err != nil {
		return Report{}, err
	}
	return rep, nil
}

// topServices agrupa pelo nome gravado no item, e não pelo serviço da tabela
// de preços: o item guarda o nome com que foi vendido, então um serviço
// renomeado depois continua somando no nome que a cliente viu no comprovante.
func (s *Store) topServices(from, to string) ([]ReportService, error) {
	rows, err := s.db.Query(`
		SELECT i.name,
		       SUM(i.qty),
		       COUNT(DISTINCT o.id),
		       SUM(i.qty * i.price_cents)
		FROM order_items i
		JOIN orders o ON o.id = i.order_id
		WHERE o.date BETWEEN ? AND ? AND o.status <> 'Cancelado'
		GROUP BY i.name
		ORDER BY SUM(i.qty * i.price_cents) DESC, SUM(i.qty) DESC, i.name
		LIMIT 8`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []ReportService{}
	for rows.Next() {
		var r ReportService
		if err := rows.Scan(&r.Name, &r.Qty, &r.Orders, &r.TotalCents); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (s *Store) topClients(from, to string) ([]ReportClient, error) {
	rows, err := s.db.Query(`
		SELECT o.client_id,
		       COALESCE(c.name, 'Cliente removido'),
		       COUNT(*),
		       COALESCE(SUM(t.total_cents), 0)
		FROM orders o
		LEFT JOIN clients c ON c.id = o.client_id
		JOIN order_totals t ON t.order_id = o.id
		WHERE o.date BETWEEN ? AND ? AND o.status <> 'Cancelado'
		GROUP BY o.client_id
		ORDER BY SUM(t.total_cents) DESC, COUNT(*) DESC
		LIMIT 5`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []ReportClient{}
	for rows.Next() {
		var r ReportClient
		if err := rows.Scan(&r.ID, &r.Name, &r.Orders, &r.TotalCents); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// ordersByStatus devolve todos os status na ordem em que a UI os mostra,
// inclusive os que ficaram em zero — o relatório de um mês fraco precisa
// poder dizer "nenhum entregue", e não omitir a linha.
func (s *Store) ordersByStatus(from, to string) ([]ReportStatus, error) {
	rows, err := s.db.Query(`
		SELECT status, COUNT(*) FROM orders
		WHERE date BETWEEN ? AND ?
		GROUP BY status`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contagem := map[string]int{}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		contagem[status] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	list := make([]ReportStatus, 0, len(OrderStatuses))
	for _, status := range OrderStatuses {
		list = append(list, ReportStatus{Status: status, Orders: contagem[status]})
	}
	return list, nil
}
