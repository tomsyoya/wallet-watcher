package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type TxEvent struct {
	TxHash   string     `json:"tx_hash"`
	TS       time.Time  `json:"ts"`
	Sender   *string    `json:"sender,omitempty"`
	Receiver *string    `json:"receiver,omitempty"`
	Token    *string    `json:"token,omitempty"`
	Amount   *int64     `json:"amount,omitempty"`
	Fee      *int64     `json:"fee,omitempty"`
	Method   *string    `json:"method,omitempty"`
}

func (s *Store) ListTxEvents(ctx context.Context, chain string, address *string, limit int, before *time.Time) ([]TxEvent, error) {
    if limit <= 0 || limit > 200 { limit = 50 }

    var q string
    args := []any{}

    switch chain {
    case "solana":
        q = `
          SELECT tx_hash, ts, sender, receiver, token,
                 NULLIF(amount::text,'')::bigint AS amount,
                 NULLIF(fee::text,'')::bigint    AS fee,
                 method
          FROM tx_events_solana
          WHERE 1=1
        `
    case "sui":
        q = `
          SELECT tx_hash, ts, sender, receiver, token,
                 NULLIF(amount::text,'')::bigint AS amount,
                 NULLIF(fee::text,'')::bigint    AS fee,
                 method
          FROM tx_events_sui
          WHERE 1=1
        `
    default:
        return nil, fmt.Errorf("unsupported chain: %s", chain)
    }

    if address != nil && *address != "" {
        switch chain {
        case "solana":
            // 厳密一致 + raw にも含まれていればヒット
            args = append(args, *address, *address, "%"+*address+"%")
            q += fmt.Sprintf(`
              AND (
                sender = $%d OR receiver = $%d OR raw::text ILIKE $%d
              )`, len(args)-2, len(args)-1, len(args))
        case "sui":
            // 0xを外して小文字へ（DB側の sender/receiver も同様に比較）
			      addrNorm := strings.ToLower(strings.TrimPrefix(*address, "0x"))
			      args = append(args, addrNorm, addrNorm, "%"+addrNorm+"%")
			      q += fmt.Sprintf(`
			      	AND (
			      		lower(regexp_replace(COALESCE(sender,''), '^0x', '')) = $%d
			      		OR lower(regexp_replace(COALESCE(receiver,''), '^0x', '')) = $%d
			      		OR lower(raw::text) ILIKE $%d
			      	)
			      `, len(args)-2, len(args)-1, len(args))
        }
    }

    if before != nil {
        args = append(args, *before)
        q += fmt.Sprintf(" AND ts < $%d", len(args))
    }

    args = append(args, limit)
    q += fmt.Sprintf(" ORDER BY ts DESC LIMIT $%d", len(args))

    rows, err := s.Pool.Query(ctx, q, args...)
    if err != nil { return nil, err }
    defer rows.Close()

    out := make([]TxEvent, 0, limit)
    for rows.Next() {
        var e TxEvent
        if err := rows.Scan(&e.TxHash, &e.TS, &e.Sender, &e.Receiver, &e.Token, &e.Amount, &e.Fee, &e.Method); err != nil {
            return nil, err
        }
        out = append(out, e)
    }
    return out, rows.Err()
}
