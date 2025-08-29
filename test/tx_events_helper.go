package test

import (
	"context"
	"time"

	"github.com/you/wallet-watcher/internal/store"
)

// Solana （最小カラムのみ） 登録用ヘルパー
func InsertTxEventSolanaForTest(
    ctx context.Context, st *store.Store,
    txHash string, ts time.Time,
    sender, receiver *string,
    fee *int64, method *string,
    rawJSON []byte,
) error {
    _, err := st.Pool.Exec(ctx, `
        INSERT INTO tx_events_solana (tx_hash, ts, sender, receiver, fee, method, raw)
        VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7::text, '')::jsonb)
        ON CONFLICT (tx_hash, ts) DO NOTHING;
    `, txHash, ts, sender, receiver, fee, method, string(rawJSON))
    return err
}

// Sui （最小カラムのみ） 登録用ヘルパー
func InsertTxEventSuiForTest(
    ctx context.Context, st *store.Store,
    txHash string, ts time.Time, rawJSON []byte,
) error {
    _, err := st.Pool.Exec(ctx, `
        INSERT INTO tx_events_sui (tx_hash, ts, raw)
        VALUES ($1, $2, NULLIF($3::text, '')::jsonb)
        ON CONFLICT (tx_hash, ts) DO NOTHING;
    `, txHash, ts, string(rawJSON))
    return err
}