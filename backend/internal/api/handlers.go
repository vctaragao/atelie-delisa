package api

import (
	"net/http"
	"strings"

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
	if len(o.Items) == 0 {
		writeErr(w, http.StatusUnprocessableEntity, "Adicione pelo menos um serviço ao pedido.")
		return false
	}
	for i := range o.Items {
		it := &o.Items[i]
		it.Name = strings.TrimSpace(it.Name)
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
