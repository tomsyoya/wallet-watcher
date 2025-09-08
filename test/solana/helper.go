package solanatest

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/you/wallet-watcher/internal/store"
)


type txEventSolanaRow struct {
	TxHash   string
	TS       time.Time
	Sender   *string
	Receiver *string
	Token    *string
	Amount   *int64
	Fee      *int64
	Method   *string
	RawText  *string
}

func testTimeWindow() (start, end time.Time) {
	end = time.Now().UTC()
	h := 24
	if v := os.Getenv("TEST_HISTORY_WINDOW_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			h = n
		}
	}
	return end.Add(-time.Duration(h) * time.Hour), end
}

// 直近ウィンドウのバックアップ
func backupTxEventsSolana(ctx context.Context, st *store.Store) ([]txEventSolanaRow, error) {
	start, end := testTimeWindow()
	rows, err := st.Pool.Query(ctx, `
		SELECT tx_hash, ts, sender, receiver, token,
		       NULLIF(amount::text,'')::bigint AS amount,
		       NULLIF(fee::text,'')::bigint    AS fee,
		       method,
		       raw::text
		FROM tx_events_solana
		WHERE ts >= $1 AND ts < $2
	`, start, end)
	if err != nil { return nil, err }
	defer rows.Close()

	var out []txEventSolanaRow
	for rows.Next() {
		var r txEventSolanaRow
		if err := rows.Scan(&r.TxHash, &r.TS, &r.Sender, &r.Receiver, &r.Token, &r.Amount, &r.Fee, &r.Method, &r.RawText); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// 直近ウィンドウだけ削除（TRUNCATEしない）
func clearWindowTxEventsSolana(ctx context.Context, st *store.Store) error {
	start, end := testTimeWindow()
	_, err := st.Pool.Exec(ctx, `DELETE FROM tx_events_solana WHERE ts >= $1 AND ts < $2`, start, end)
	return err
}

func restoreTxEventsSolana(ctx context.Context, st *store.Store, backup []txEventSolanaRow) error {
	if len(backup) == 0 { return nil }
	batch := &pgx.Batch{}
	for _, r := range backup {
		batch.Queue(`
			INSERT INTO tx_events_solana
			  (tx_hash, ts, sender, receiver, token, amount, fee, method, raw)
			VALUES
			  ($1,$2,$3,$4,$5,$6,$7,$8, NULLIF($9::text,'')::jsonb)
			ON CONFLICT (tx_hash, ts) DO NOTHING
		`, r.TxHash, r.TS, r.Sender, r.Receiver, r.Token, r.Amount, r.Fee, r.Method, r.RawText)
	}
	br := st.Pool.SendBatch(ctx, batch)
	return br.Close()
}

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
