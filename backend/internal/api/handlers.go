package api

import (
	"net/http"
	"strings"
	"time"

	"atelie/backend/internal/store"
)

type Server struct {
	st *store.Store
}

func New(st *store.Store) *Server { return &Server{st: st} }

// ---------- clientes ----------

func (s *Server) listClients(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListClients()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) createClient(w http.ResponseWriter, r *http.Request) {
	var in store.Client
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validateClient(w, &in) {
		return
	}
	out, err := s.st.CreateClient(in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) updateClient(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in store.Client
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validateClient(w, &in) {
		return
	}
	out, err := s.st.UpdateClient(id, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) deleteClient(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.st.DeleteClient(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func validateClient(w http.ResponseWriter, c *store.Client) bool {
	c.Name = strings.TrimSpace(c.Name)
	c.Phone = strings.TrimSpace(c.Phone)
	c.Address = strings.TrimSpace(c.Address)
	c.Notes = strings.TrimSpace(c.Notes)
	if c.Name == "" {
		writeErr(w, http.StatusUnprocessableEntity, "O nome do cliente é obrigatório.")
		return false
	}
	return true
}

// ---------- serviços ----------

func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListServices()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) createService(w http.ResponseWriter, r *http.Request) {
	var in store.Service
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validateService(w, &in) {
		return
	}
	out, err := s.st.CreateService(in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) updateService(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in store.Service
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validateService(w, &in) {
		return
	}
	out, err := s.st.UpdateService(id, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) deleteService(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.st.DeleteService(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func validateService(w http.ResponseWriter, v *store.Service) bool {
	v.Name = strings.TrimSpace(v.Name)
	v.Category = strings.TrimSpace(v.Category)
	v.Time = strings.TrimSpace(v.Time)
	if v.Name == "" {
		writeErr(w, http.StatusUnprocessableEntity, "O nome do serviço é obrigatório.")
		return false
	}
	if v.PriceCents < 0 {
		writeErr(w, http.StatusUnprocessableEntity, "O preço não pode ser negativo.")
		return false
	}
	return true
}

// ---------- pedidos ----------

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListOrders()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	out, err := s.st.GetOrder(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	var in store.Order
	if !decodeJSON(w, r, &in) {
		return
	}
	if !s.validateOrder(w, &in) {
		return
	}
	out, err := s.st.CreateOrder(in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) updateOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in store.Order
	if !decodeJSON(w, r, &in) {
		return
	}
	if !s.validateOrder(w, &in) {
		return
	}
	out, err := s.st.UpdateOrder(id, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) deleteOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.st.DeleteOrder(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) validateOrder(w http.ResponseWriter, o *store.Order) bool {
	o.Notes = strings.TrimSpace(o.Notes)
	o.Due = strings.TrimSpace(o.Due)
	o.DueTime = strings.TrimSpace(o.DueTime)

	// Hora sem dia não tem onde aparecer na agenda, que é organizada por data.
	// Em vez de recusar o pedido, descarta a hora solta.
	if o.Due == "" {
		o.DueTime = ""
	}
	if o.DueTime != "" {
		if _, err := time.Parse("15:04", o.DueTime); err != nil {
			writeErr(w, http.StatusUnprocessableEntity,
				"O horário de entrega deve estar no formato HH:MM, como 14:30.")
			return false
		}
	}

	if _, err := s.st.GetClient(o.ClientID); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "Selecione um cliente válido.")
		return false
	}
	if o.Status == "" {
		o.Status = store.OrderStatuses[0]
	}
	if !store.ValidStatus(o.Status) {
		writeErr(w, http.StatusUnprocessableEntity,
			"Status inválido. Use um destes: "+strings.Join(store.OrderStatuses, ", ")+".")
		return false
	}
	if o.Payment == "" {
		o.Payment = "Não informado"
	}
	// Desconto não passa de 100%: acima disso o ateliê estaria pagando para
	// trabalhar. A urgência vai até 100%, que é cobrar o dobro — o teto existe
	// para um dígito a mais na pressa não virar uma conta absurda no
	// comprovante da cliente.
	if o.DiscountPercent < 0 || o.DiscountPercent > 100 {
		writeErr(w, http.StatusUnprocessableEntity,
			"O desconto deve ser de 0% a 100%.")
		return false
	}
	if o.RushPercent < 0 || o.RushPercent > 100 {
		writeErr(w, http.StatusUnprocessableEntity,
			"O acréscimo de urgência deve ser de 0% a 100%.")
		return false
	}
	if len(o.Items) == 0 {
		writeErr(w, http.StatusUnprocessableEntity, "Adicione pelo menos um serviço ao pedido.")
		return false
	}
	for i := range o.Items {
		it := &o.Items[i]
		it.Name = strings.TrimSpace(it.Name)
		it.Piece = strings.TrimSpace(it.Piece)
		if it.Name == "" {
			it.Name = "Serviço personalizado"
		}
		if it.Qty < 1 {
			writeErr(w, http.StatusUnprocessableEntity, "A quantidade de cada item deve ser 1 ou mais.")
			return false
		}
		if it.PriceCents < 0 {
			writeErr(w, http.StatusUnprocessableEntity, "O preço de um item não pode ser negativo.")
			return false
		}
	}
	return true
}

// ---------- etapa e pagamentos do pedido ----------

// statusBody é o corpo de PUT /api/orders/{id}/status. Só a etapa: mover um
// pedido para produção pela lista de ordens não deve mexer em mais nada, e
// mandar o pedido inteiro de volta só para isso reescreveria os itens à toa.
type statusBody struct {
	Status string `json:"status"`
}

func (s *Server) setOrderStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in statusBody
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Status = strings.TrimSpace(in.Status)
	if !store.ValidStatus(in.Status) {
		writeErr(w, http.StatusUnprocessableEntity,
			"Status inválido. Use um destes: "+strings.Join(store.OrderStatuses, ", ")+".")
		return
	}
	out, err := s.st.SetOrderStatus(id, in.Status)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// createPayment dá baixa de um valor recebido. Devolve o pedido inteiro, e
// não o pagamento: quem está na tela precisa ver na hora quanto ainda falta.
func (s *Server) createPayment(w http.ResponseWriter, r *http.Request) {
	orderID, ok := pathID(w, r)
	if !ok {
		return
	}
	var in store.Payment
	if !decodeJSON(w, r, &in) {
		return
	}

	// Quem manda no pedido é a rota, não o corpo.
	in.OrderID = orderID
	in.Date = strings.TrimSpace(in.Date)
	in.Method = strings.TrimSpace(in.Method)
	in.Notes = strings.TrimSpace(in.Notes)

	if in.Date == "" {
		writeErr(w, http.StatusUnprocessableEntity,
			"Informe a data em que o pagamento foi recebido.")
		return
	}
	if _, err := time.Parse("2006-01-02", in.Date); err != nil {
		writeErr(w, http.StatusUnprocessableEntity,
			"A data do pagamento deve estar no formato AAAA-MM-DD.")
		return
	}
	// Zero não é baixa nenhuma. Valor negativo seria um estorno, e o estorno
	// aqui é excluir o pagamento lançado errado — some do histórico em vez de
	// virar uma linha negativa que ninguém entende depois.
	if in.ValueCents <= 0 {
		writeErr(w, http.StatusUnprocessableEntity,
			"O valor recebido precisa ser maior que zero.")
		return
	}
	if in.Method == "" {
		in.Method = "Não informado"
	}

	out, err := s.st.CreatePayment(in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) deletePayment(w http.ResponseWriter, r *http.Request) {
	orderID, ok := pathID(w, r)
	if !ok {
		return
	}
	paymentID, ok := pathInt64(w, r, "paymentId")
	if !ok {
		return
	}

	// O pagamento tem de ser mesmo deste pedido: sem esta conferência, um
	// endereço montado errado apagaria a baixa de outra cliente.
	pg, err := s.st.GetPayment(paymentID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	if pg.OrderID != orderID {
		writeErr(w, http.StatusNotFound, "Este pagamento não é deste pedido.")
		return
	}

	out, err := s.st.DeletePayment(paymentID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// ---------- mensagens do WhatsApp ----------

func (s *Server) listMessages(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListMessages()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) createMessage(w http.ResponseWriter, r *http.Request) {
	var in store.MessageTemplate
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validateMessage(w, &in) {
		return
	}
	out, err := s.st.CreateMessage(in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) updateMessage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in store.MessageTemplate
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validateMessage(w, &in) {
		return
	}
	out, err := s.st.UpdateMessage(id, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) deleteMessage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.st.DeleteMessage(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) restoreMessages(w http.ResponseWriter, r *http.Request) {
	if err := s.st.RestoreMessages(); err != nil {
		writeStoreErr(w, err)
		return
	}
	list, err := s.st.ListMessages()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func validateMessage(w http.ResponseWriter, m *store.MessageTemplate) bool {
	m.Name = strings.TrimSpace(m.Name)
	m.Body = strings.TrimSpace(m.Body)
	m.SuggestStatus = strings.TrimSpace(m.SuggestStatus)

	if m.Name == "" {
		writeErr(w, http.StatusUnprocessableEntity, "Dê um nome à mensagem.")
		return false
	}
	if m.Body == "" {
		writeErr(w, http.StatusUnprocessableEntity, "Escreva o texto da mensagem.")
		return false
	}
	// Vazio é válido: a mensagem existe na lista, mas não é sugerida sozinha.
	if m.SuggestStatus != "" && !store.ValidStatus(m.SuggestStatus) {
		writeErr(w, http.StatusUnprocessableEntity,
			"Situação inválida. Use uma destas: "+strings.Join(store.OrderStatuses, ", ")+".")
		return false
	}
	return true
}

// ---------- lançamentos financeiros ----------

func (s *Server) listTransactions(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListTransactions()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) createTransaction(w http.ResponseWriter, r *http.Request) {
	var in store.Transaction
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Description = strings.TrimSpace(in.Description)
	in.Date = strings.TrimSpace(in.Date)
	if in.Date == "" {
		writeErr(w, http.StatusUnprocessableEntity, "A data é obrigatória.")
		return
	}
	if in.Description == "" {
		writeErr(w, http.StatusUnprocessableEntity, "A descrição é obrigatória.")
		return
	}
	if in.Type != "entrada" && in.Type != "saida" {
		writeErr(w, http.StatusUnprocessableEntity, `O tipo deve ser "entrada" ou "saida".`)
		return
	}
	if in.ValueCents < 0 {
		writeErr(w, http.StatusUnprocessableEntity, "O valor não pode ser negativo.")
		return
	}

	out, err := s.st.CreateTransaction(in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.st.DeleteTransaction(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ---------- resumo, backup ----------

func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	out, err := s.st.Summary()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// report fecha um período. Sem datas na URL, devolve os últimos 30 dias, que
// é o que a tela abre mostrando.
func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	const dia = "2006-01-02"

	hoje := time.Now()
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	if from == "" {
		from = hoje.AddDate(0, 0, -29).Format(dia)
	}
	if to == "" {
		to = hoje.Format(dia)
	}

	for _, d := range []string{from, to} {
		if _, err := time.Parse(dia, d); err != nil {
			writeErr(w, http.StatusUnprocessableEntity,
				"Data inválida no período do relatório.")
			return
		}
	}
	if from > to {
		writeErr(w, http.StatusUnprocessableEntity,
			"A data inicial do período vem depois da final.")
		return
	}

	out, err := s.st.Report(from, to)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) exportBackup(w http.ResponseWriter, r *http.Request) {
	out, err := s.st.Export()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	w.Header().Set("Content-Disposition",
		`attachment; filename="backup-atelie-delisa.json"`)
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) importBackup(w http.ResponseWriter, r *http.Request) {
	var in store.Backup
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Clients == nil || in.Services == nil || in.Orders == nil {
		writeErr(w, http.StatusUnprocessableEntity, "Arquivo de backup inválido.")
		return
	}
	if err := s.st.Import(in); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restaurado"})
}

func (s *Server) resetData(w http.ResponseWriter, r *http.Request) {
	if err := s.st.Reset(); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "apagado"})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
