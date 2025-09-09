package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	sol "github.com/you/wallet-watcher/internal/chains/solana"
	"github.com/you/wallet-watcher/internal/store"
	"github.com/you/wallet-watcher/internal/worker"
)

func main() {
	ctx := context.Background()

	st, err := store.New(ctx)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	rpc := os.Getenv("SOLANA_RPC_URL")
	if rpc == "" {
		log.Fatalf("SOLANA_RPC_URL is required")
	}
	cl := sol.New(rpc)

	interval := 5 * time.Second
	if v := os.Getenv("POLL_INTERVAL_SEC"); v != "" {
		if d, err := time.ParseDuration(v + "s"); err == nil {
			interval = d
		}
	}

	batch := 10 // 1アドレスあたり一度に取得する最大件数
	if v := os.Getenv("BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			batch = n
		}
	}

	w := worker.NewSolana(st, cl, batch)
	log.Printf("worker started: interval=%v batch=%d", interval, batch)

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		if err := w.Tick(ctx); err != nil {
			log.Printf("tick error: %v", err)
		}
		<-t.C
	}
}