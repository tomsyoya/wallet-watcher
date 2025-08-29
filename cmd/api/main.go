package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	api "github.com/you/wallet-watcher/internal/api"
	"github.com/you/wallet-watcher/internal/store"
)

func main() {
	_ = godotenv.Load()

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	// DB 接続
	ctx := context.Background()
	st, err := store.New(ctx)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer st.Close()

	// ルーティング
	srv := &api.Server{Store: st}
	r := api.Routes(srv)

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
