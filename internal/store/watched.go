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
