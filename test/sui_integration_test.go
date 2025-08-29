//go:build integration

package test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	sui "github.com/you/wallet-watcher/internal/chains/sui"
	"github.com/you/wallet-watcher/internal/store"
)

func TestSui_Integration_One(t *testing.T) {
	rpc := os.Getenv("SUI_RPC_URL")
	addr := os.Getenv("SUI_ADDR")
	db := os.Getenv("DATABASE_URL")
	if rpc == "" || addr == "" || db == "" {
		t.Skip("set SUI_RPC_URL, SUI_ADDR, DATABASE_URL to run this test")
	}

	ctx := context.Background()
	st, err := store.New(ctx)
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}
	defer st.Close()

	cl := sui.New(rpc)

	// 1) 対象アドレス宛のTxを 1 件だけ取得（digest）
	cCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	digest, err := cl.QueryOneTxDigestByToAddress(cCtx, addr)
	if err != nil {
		t.Fatalf("query tx blocks: %v", err)
	}
	if digest == "" {
		t.Skip("no transactions found for given address; try another SUI_ADDR")
	}

	// 2) そのトランザクションブロックの詳細
	tb, err := cl.GetTransactionBlock(cCtx, digest)
	if err != nil {
		t.Fatalf("get tx block: %v", err)
	}
	// 3) DB に 1 件だけ INSERT（最小）
	var ts time.Time
	if ms, ok := tb.TimestampMs.Value(); ok && ms > 0 {
	    ts = time.UnixMilli(int64(ms)).UTC()
	} else {
	    ts = time.Now().UTC()
	}
	raw, _ := json.Marshal(tb)
	if err := InsertTxEventSuiForTest(ctx, st, digest, ts, raw); err != nil {
		t.Fatalf("insert tx_events_sui failed: %v", err)
	}
}
