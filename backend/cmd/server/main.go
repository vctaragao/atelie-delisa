// Command server sobe a API HTTP do Ateliê Delisa.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"atelie/backend/internal/api"
	"atelie/backend/internal/store"
)

func main() {
	log.SetFlags(log.LstdFlags)

	addr := env("ADDR", ":8080")
	dbPath := env("DB_PATH", "/data/atelie.db")
	origins := strings.Split(env("CORS_ORIGINS", "http://localhost:5173,http://localhost:8081"), ",")

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatalf("criando diretório do banco: %v", err)
	}

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("abrindo banco: %v", err)
	}
	defer st.Close()
	log.Printf("banco pronto em %s", dbPath)

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.New(st).Routes(origins),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Sobe o servidor em outra goroutine para poder aguardar o sinal de
	// encerramento na principal.
	errc := make(chan error, 1)
	go func() {
		log.Printf("API ouvindo em %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errc:
		log.Fatalf("servidor falhou: %v", err)
	case <-stop:
		log.Println("encerrando...")
	}

	// Dá tempo para as requisições em andamento terminarem antes de fechar
	// o banco (o defer st.Close() roda depois disso).
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("encerramento forçado: %v", err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
