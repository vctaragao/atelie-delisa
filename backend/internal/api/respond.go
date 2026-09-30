package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"atelie/backend/internal/store"
)

// maxBody limita o corpo das requisições. O maior payload esperado é a
// restauração de backup, por isso 16 MB.
const maxBody = 16 << 20

type errorBody struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Cabeçalhos já foram enviados; só dá para registrar.
		log.Printf("erro ao escrever resposta JSON: %v", err)
	}
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Error: msg})
}

// writeStoreErr traduz os erros do store para status HTTP.
func writeStoreErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, "Registro não encontrado.")
	case errors.Is(err, store.ErrHasOrders):
		writeErr(w, http.StatusConflict,
			"Este cliente possui pedidos. Exclua ou altere os pedidos antes de excluir o cliente.")
	default:
		log.Printf("erro interno: %v", err)
		writeErr(w, http.StatusInternalServerError, "Erro interno no servidor.")
	}
}

// decodeJSON lê o corpo da requisição, recusando campos desconhecidos para
// que erros de digitação no cliente não passem silenciosamente.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeErr(w, http.StatusBadRequest, "Dados inválidos: "+err.Error())
		return false
	}
	return true
}

// pathID extrai e valida o {id} da rota.
func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, "Identificador inválido.")
		return 0, false
	}
	return id, true
}
