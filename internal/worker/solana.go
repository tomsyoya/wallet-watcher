package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	sol "github.com/you/wallet-watcher/internal/chains/solana"
	"github.com/you/wallet-watcher/internal/store"
)

type SolanaWorker struct {
	st    *store.Store
	cl    *sol.Client
	batch int
}

func NewSolana(st *store.Store, cl *sol.Client, batch int) *SolanaWorker {
	if batch <= 0 { batch = 10 }
	return &SolanaWorker{st: st, cl: cl, batch: batch}
}

// 1回分の処理：登録アドレスを列挙→各アドレスの新着Txを取得→保存→カーソル更新
func (w *SolanaWorker) Tick(ctx context.Context) error {
	addrs, err := w.st.ListWatchedSolana(ctx, 200)
	if err != nil { return err }

	for _, a := range addrs {
		if err := w.processAddress(ctx, a.Address, a.LastSlot); err != nil {
			log.Printf("[solana] address=%s err=%v", a.Address, err)
		}
	}
	return nil
}

func (w *SolanaWorker) processAddress(ctx context.Context, address string, lastSlot *int64) error {
	// 新しいものから最大batch件を取得し、lastSlotより新しいものを抽出
	sigs, err := w.cl.GetSignaturesForAddress(ctx, address, w.batch)
	if err != nil { return err }

	var newestSlot int64 = -1
	for _, s := range sigs {
		if lastSlot != nil && int64(s.Slot) <= *lastSlot {
			// ここより古い分は無視
			continue
		}
		// 詳細取得
		tx, err := w.cl.GetTransaction(ctx, s.Signature)
		if err != nil {
			log.Printf("[solana] get tx %s: %v", s.Signature, err)
			continue
		}

		// 正規化して保存（最小）
		var ts time.Time
		if tx.BlockTime != nil && *tx.BlockTime > 0 {
			ts = time.Unix(*tx.BlockTime, 0).UTC()
		} else {
			ts = time.Now().UTC()
		}
		var sender, receiver *string
		if len(tx.Transaction.Message.AccountKeys) > 0 {
			s := tx.Transaction.Message.AccountKeys[0]; sender = &s
		}
		if len(tx.Transaction.Message.AccountKeys) > 1 {
			r := tx.Transaction.Message.AccountKeys[1]; receiver = &r
		}
		var fee *int64
		if tx.Meta != nil {
			f := int64(tx.Meta.Fee); fee = &f
		}
		raw, _ := json.Marshal(tx)
		if err := w.st.InsertTxEventSolana(ctx, s.Signature, ts, sender, receiver, fee, nil, raw); err != nil {
			log.Printf("[solana] insert tx %s: %v", s.Signature, err)
		}
		if int64(s.Slot) > newestSlot {
			newestSlot = int64(s.Slot)
		}
	}

	// カーソル更新
	if newestSlot >= 0 {
		return w.st.UpdateSolanaCursor(ctx, address, newestSlot)
	}
	return nil
}
