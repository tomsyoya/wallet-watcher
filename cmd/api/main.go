package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/you/wallet-watcher/internal/api"
	"github.com/you/wallet-watcher/internal/store"
)

func main() {
    _ = godotenv.Load()
    port := os.Getenv("APP_PORT")
    if port == "" { port = "8080" }

    dbUrl := os.Getenv("DATABASE_URL")
    if dbUrl == "" {
        log.Fatal("DATABASE_URL is empty")
    }


    ctx := context.Background()
    pool, err := pgxpool.New(ctx, dbUrl)
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    s := &store.Store{DB: pool}

    r := chi.NewRouter()
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })

    r.Post("/register", (&api.RegisterHandler{Store: s}).ServeHTTP)

    log.Printf("listening on :%s", port)
    if err := http.ListenAndServe(":"+port, r); err != nil {
        log.Fatal(err)
    }
}

