package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"atelie/backend/internal/store"
)

func servidorComCliente(t *testing.T) (*Server, int64) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "atelie.db"))
	if err != nil {
		t.Fatalf("abrindo o banco de teste: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	cliente, err := st.CreateClient(store.Client{Name: "Dona Delisa"})
	if err != nil {
		t.Fatalf("criando cliente: %v", err)
	}
	return New(st), cliente.ID
}

func postPedido(t *testing.T, s *Server, corpo string) (*httptest.ResponseRecorder, store.Order) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Routes([]string{"*"}).ServeHTTP(rec, req)

	var pedido store.Order
	if rec.Code == http.StatusCreated {
		if err := json.Unmarshal(rec.Body.Bytes(), &pedido); err != nil {
			t.Fatalf("resposta ilegível: %v — corpo: %s", err, rec.Body.String())
		}
	}
	return rec, pedido
}

// TestPedidoAceitaDescontoEUrgencia confere que os percentuais chegam ao banco
// e voltam com a conta fechada — é o que a tela mostra e o comprovante imprime.
func TestPedidoAceitaDescontoEUrgencia(t *testing.T) {
	s, cliente := servidorComCliente(t)

	rec, pedido := postPedido(t, s, `{
		"clientId": `+itoa(cliente)+`,
		"date": "2026-10-22",
		"due": "2026-10-22",
		"dueTime": "17:00",
		"discountPercent": 10,
		"rushPercent": 20,
		"items": [{"name": "Bainha", "qty": 2, "priceCents": 5000}]
	}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("esperava 201, veio %d: %s", rec.Code, rec.Body.String())
	}
	if pedido.ItemsCents != 10000 || pedido.DiscountCents != 1000 ||
		pedido.RushCents != 2000 || pedido.TotalCents != 11000 {
		t.Errorf("conta errada: serviços %d, desconto %d, urgência %d, total %d",
			pedido.ItemsCents, pedido.DiscountCents, pedido.RushCents, pedido.TotalCents)
	}
}

// TestPedidoRecusaPercentualForaDaFaixa: um dígito a mais na pressa não pode
// virar um desconto de 1000% e um pedido de valor negativo.
func TestPedidoRecusaPercentualForaDaFaixa(t *testing.T) {
	s, cliente := servidorComCliente(t)

	casos := []struct {
		nome     string
		percents string
	}{
		{"desconto acima de 100", `"discountPercent": 150`},
		{"desconto negativo", `"discountPercent": -5`},
		{"urgência acima de 100", `"rushPercent": 300`},
		{"urgência negativa", `"rushPercent": -1`},
	}

	for _, c := range casos {
		rec, _ := postPedido(t, s, `{
			"clientId": `+itoa(cliente)+`,
			"date": "2026-10-22",
			`+c.percents+`,
			"items": [{"name": "Bainha", "qty": 1, "priceCents": 5000}]
		}`)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: esperava 422, veio %d: %s", c.nome, rec.Code, rec.Body.String())
		}
	}
}

// TestPedidoRecusaHorarioInvalido fecha o outro campo novo da agenda: hora
// digitada errada tem de voltar como erro, não ser gravada como texto solto.
func TestPedidoRecusaHorarioInvalido(t *testing.T) {
	s, cliente := servidorComCliente(t)

	rec, _ := postPedido(t, s, `{
		"clientId": `+itoa(cliente)+`,
		"date": "2026-10-22",
		"due": "2026-10-23",
		"dueTime": "25:90",
		"items": [{"name": "Bainha", "qty": 1, "priceCents": 5000}]
	}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("esperava 422, veio %d: %s", rec.Code, rec.Body.String())
	}

	// Hora sem dia marcado não é erro: a agenda é organizada por data, então a
	// hora solta é descartada e o pedido entra sem horário.
	rec, pedido := postPedido(t, s, `{
		"clientId": `+itoa(cliente)+`,
		"date": "2026-10-22",
		"due": "",
		"dueTime": "14:30",
		"items": [{"name": "Bainha", "qty": 1, "priceCents": 5000}]
	}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("esperava 201, veio %d: %s", rec.Code, rec.Body.String())
	}
	if pedido.DueTime != "" {
		t.Errorf("o horário solto deveria ter sido descartado, veio %q", pedido.DueTime)
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
