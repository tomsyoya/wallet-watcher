package store

import (
	"context"
	"time"
)

func (s *Store) InsertTxEventSolana(
	ctx context.Context,
	txHash string,
	ts time.Time,
	sender, receiver *string,
	fee *int64,
	method *string,
	rawJSON []byte,
) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO tx_events_solana (tx_hash, ts, sender, receiver, fee, method, raw)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7::text,'')::jsonb)
		ON CONFLICT (tx_hash, ts) DO NOTHING;
	`, txHash, ts, sender, receiver, fee, method, string(rawJSON))
	return err
}

func (s *Store) InsertTxEventSui(
	ctx context.Context,
	txHash string,
	ts time.Time,
	sender, receiver *string,
	fee *int64,
	method *string,
	rawJSON []byte,
) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO tx_events_sui (tx_hash, ts, sender, receiver, fee, method, raw)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7::text,'')::jsonb)
		ON CONFLICT (tx_hash, ts) DO NOTHING;
	`, txHash, ts, sender, receiver, fee, method, string(rawJSON))
	return err
}
