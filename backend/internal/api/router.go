package api

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// Routes monta o roteador. Usa o ServeMux da stdlib (Go 1.22+), que já
// entende método e parâmetros de caminho — nenhuma dependência externa.
func (s *Server) Routes(allowedOrigins []string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/summary", s.summary)

	mux.HandleFunc("GET /api/clients", s.listClients)
	mux.HandleFunc("POST /api/clients", s.createClient)
	mux.HandleFunc("PUT /api/clients/{id}", s.updateClient)
	mux.HandleFunc("DELETE /api/clients/{id}", s.deleteClient)

	mux.HandleFunc("GET /api/services", s.listServices)
	mux.HandleFunc("POST /api/services", s.createService)
	mux.HandleFunc("PUT /api/services/{id}", s.updateService)
	mux.HandleFunc("DELETE /api/services/{id}", s.deleteService)

	mux.HandleFunc("GET /api/orders", s.listOrders)
	mux.HandleFunc("POST /api/orders", s.createOrder)
	mux.HandleFunc("GET /api/orders/{id}", s.getOrder)
	mux.HandleFunc("PUT /api/orders/{id}", s.updateOrder)
	mux.HandleFunc("DELETE /api/orders/{id}", s.deleteOrder)

	mux.HandleFunc("GET /api/transactions", s.listTransactions)
	mux.HandleFunc("POST /api/transactions", s.createTransaction)
	mux.HandleFunc("DELETE /api/transactions/{id}", s.deleteTransaction)

	mux.HandleFunc("GET /api/backup", s.exportBackup)
	mux.HandleFunc("POST /api/backup/restore", s.importBackup)
	mux.HandleFunc("DELETE /api/data", s.resetData)

	return logging(cors(allowedOrigins)(mux))
}

// cors libera as origens configuradas. Em desenvolvimento o front roda em
// :5173 e o back em :8080, então o navegador exige esses cabeçalhos.
func cors(allowed []string) func(http.Handler) http.Handler {
	allowAll := len(allowed) == 1 && allowed[0] == "*"

	permitted := func(origin string) bool {
		for _, a := range allowed {
			if strings.EqualFold(strings.TrimSpace(a), origin) {
				return true
			}
		}
		return false
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowAll || permitted(origin)) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Set("Access-Control-Max-Age", "300")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Millisecond))
	})
}
