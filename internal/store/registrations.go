package store

import (
	"context"
	"fmt"
	"time"
)

// Chain はサポートするチェーン種別
type Chain string

const (
	ChainSolana Chain = "solana"
	ChainSui    Chain = "sui"
)

// WatchedAddress は監視対象アドレス + カーソルを表す統一ビュー
type WatchedAddress struct {
	Chain          Chain      `json:"chain"`
	Address        string     `json:"address"`
	LastSignature  *string    `json:"last_signature,omitempty"`  // Solana 用
	LastCheckpoint *int64     `json:"last_checkpoint,omitempty"` // Sui 用
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// UpsertWatchedAddress は (chain,address) を監視対象に追加（既存なら何もしない）
func (s *Store) UpsertWatchedAddress(ctx context.Context, chain string, address string) error {
	switch chain {
	case string(ChainSolana):
		_, err := s.Pool.Exec(ctx, `
			INSERT INTO watched_addresses_solana (address)
			VALUES ($1)
			ON CONFLICT (address) DO NOTHING;
		`, address)
		return err
	case string(ChainSui):
		_, err := s.Pool.Exec(ctx, `
			INSERT INTO watched_addresses_sui (address)
			VALUES ($1)
			ON CONFLICT (address) DO NOTHING;
		`, address)
		return err
	default:
		return fmt.Errorf("unsupported chain: %s", chain)
	}
}

// RemoveWatchedAddress は監視を解除（該当行を削除）し、削除件数を返す
func (s *Store) RemoveWatchedAddress(ctx context.Context, chain string, address string) (int64, error) {
	switch chain {
	case string(ChainSolana):
		ct, err := s.Pool.Exec(ctx, `DELETE FROM watched_addresses_solana WHERE address = $1;`, address)
		return ct.RowsAffected(), err
	case string(ChainSui):
		ct, err := s.Pool.Exec(ctx, `DELETE FROM watched_addresses_sui WHERE address = $1;`, address)
		return ct.RowsAffected(), err
	default:
		return 0, fmt.Errorf("unsupported chain: %s", chain)
	}
}

// GetWatchedAddress は単一アドレスの情報を取得
func (s *Store) GetWatchedAddress(ctx context.Context, chain string, address string) (*WatchedAddress, error) {
	switch chain {
	case string(ChainSolana):
		var w WatchedAddress
		var lastSig *string
		err := s.Pool.QueryRow(ctx, `
			SELECT address, last_signature, created_at, updated_at
			FROM watched_addresses_solana
			WHERE address = $1
		`, address).Scan(&w.Address, &lastSig, &w.CreatedAt, &w.UpdatedAt)
		if err != nil {
			return nil, err
		}
		w.Chain = ChainSolana
		w.LastSignature = lastSig
		return &w, nil

	case string(ChainSui):
		var w WatchedAddress
		var lastCp *int64
		err := s.Pool.QueryRow(ctx, `
			SELECT address, last_checkpoint, created_at, updated_at
			FROM watched_addresses_sui
			WHERE address = $1
		`, address).Scan(&w.Address, &lastCp, &w.CreatedAt, &w.UpdatedAt)
		if err != nil {
			return nil, err
		}
		w.Chain = ChainSui
		w.LastCheckpoint = lastCp
		return &w, nil

	default:
		return nil, fmt.Errorf("unsupported chain: %s", chain)
	}
}

// ListWatchedAddresses は監視中アドレスをページングして取得
func (s *Store) ListWatchedAddresses(ctx context.Context, chain string, limit, offset int) ([]WatchedAddress, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	switch chain {
	case string(ChainSolana):
		rows, err := s.Pool.Query(ctx, `
			SELECT address, last_signature, created_at, updated_at
			FROM watched_addresses_solana
			ORDER BY id DESC
			LIMIT $1 OFFSET $2
		`, limit, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var out []WatchedAddress
		for rows.Next() {
			var w WatchedAddress
			var lastSig *string
			if err := rows.Scan(&w.Address, &lastSig, &w.CreatedAt, &w.UpdatedAt); err != nil {
				return nil, err
			}
			w.Chain = ChainSolana
			w.LastSignature = lastSig
			out = append(out, w)
		}
		return out, rows.Err()

	case string(ChainSui):
		rows, err := s.Pool.Query(ctx, `
			SELECT address, last_checkpoint, created_at, updated_at
			FROM watched_addresses_sui
			ORDER BY id DESC
			LIMIT $1 OFFSET $2
		`, limit, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var out []WatchedAddress
		for rows.Next() {
			var w WatchedAddress
			var lastCp *int64
			if err := rows.Scan(&w.Address, &lastCp, &w.CreatedAt, &w.UpdatedAt); err != nil {
				return nil, err
			}
			w.Chain = ChainSui
			w.LastCheckpoint = lastCp
			out = append(out, w)
		}
		return out, rows.Err()

	default:
		return nil, fmt.Errorf("unsupported chain: %s", chain)
	}
}

// UpdateCursor はチェーンごとのカーソル（進捗）を更新
func (s *Store) UpdateCursor(ctx context.Context, chain string, address string, cursor any) error {
	switch chain {
	case string(ChainSolana):
		// cursor は signature(string) を想定
		sig, ok := cursor.(string)
		if !ok {
			return fmt.Errorf("solana cursor must be string(signature)")
		}
		_, err := s.Pool.Exec(ctx, `
			UPDATE watched_addresses_solana
			SET last_signature = $2, updated_at = NOW()
			WHERE address = $1
		`, address, sig)
		return err

	case string(ChainSui):
		// cursor は checkpoint(int64) を想定
		cp, ok := toInt64Ptr(cursor)
		if !ok {
			return fmt.Errorf("sui cursor must be int64(checkpoint) or *int64")
		}
		_, err := s.Pool.Exec(ctx, `
			UPDATE watched_addresses_sui
			SET last_checkpoint = $2, updated_at = NOW()
			WHERE address = $1
		`, address, cp)
		return err

	default:
		return fmt.Errorf("unsupported chain: %s", chain)
	}
}

func toInt64Ptr(v any) (*int64, bool) {
	switch t := v.(type) {
	case int64:
		return &t, true
	case *int64:
		return t, true
	case int:
		x := int64(t)
		return &x, true
	default:
		return nil, false
	}
}
