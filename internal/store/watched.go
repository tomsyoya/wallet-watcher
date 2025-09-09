package store

import (
	"context"
)

type WatchedSolana struct {
	Address  string
	LastSlot *int64
}

func (s *Store) ListWatchedSolana(ctx context.Context, limit int) ([]WatchedSolana, error) {
	if limit <= 0 || limit > 1000 { limit = 200 }
	rows, err := s.Pool.Query(ctx, `
		SELECT address, last_slot
		FROM watched_addresses_solana
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil { return nil, err }
	defer rows.Close()

	var out []WatchedSolana
	for rows.Next() {
		var w WatchedSolana
		if err := rows.Scan(&w.Address, &w.LastSlot); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) UpdateSolanaCursor(ctx context.Context, address string, lastSlot int64) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE watched_addresses_solana
		SET last_slot = GREATEST(COALESCE(last_slot,0), $2), updated_at = NOW()
		WHERE address = $1
	`, address, lastSlot)
	return err
}

type WatchedSui struct {
	Address        string
	LastCheckpoint *int64
}

func (s *Store) ListWatchedSui(ctx context.Context, limit int) ([]WatchedSui, error) {
	if limit <= 0 || limit > 1000 { limit = 200 }
	rows, err := s.Pool.Query(ctx, `
		SELECT address, last_checkpoint
		FROM watched_addresses_sui
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil { return nil, err }
	defer rows.Close()

	var out []WatchedSui
	for rows.Next() {
		var w WatchedSui
		if err := rows.Scan(&w.Address, &w.LastCheckpoint); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) UpdateSuiCursor(ctx context.Context, address string, lastCheckpoint int64) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE watched_addresses_sui
		SET last_checkpoint = GREATEST(COALESCE(last_checkpoint,0), $2), updated_at = NOW()
		WHERE address = $1
	`, address, lastCheckpoint)
	return err
}
