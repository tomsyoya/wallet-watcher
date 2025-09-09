package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	sui "github.com/you/wallet-watcher/internal/chains/sui"
	"github.com/you/wallet-watcher/internal/store"
)

type SuiWorker struct {
	st    *store.Store
	cl    *sui.Client
	batch int
}

func NewSui(st *store.Store, cl *sui.Client, batch int) *SuiWorker {
	if batch <= 0 { batch = 10 }
	return &SuiWorker{st: st, cl: cl, batch: batch}
}

// 1回分の処理：登録アドレスを列挙→各アドレスの新着Checkpointを取得→保存→カーソル更新
func (w *SuiWorker) Tick(ctx context.Context) error {
	addrs, err := w.st.ListWatchedSui(ctx, 200)
	if err != nil { return err }

	for _, a := range addrs {
		if err := w.processAddress(ctx, a.Address, a.LastCheckpoint); err != nil {
			log.Printf("[sui] address=%s err=%v", a.Address, err)
		}
	}
	return nil
}

func (w *SuiWorker) processAddress(ctx context.Context, address string, lastCheckpoint *int64) error {
	// 最新のCheckpointから取得開始
	var cursor *string
	if lastCheckpoint != nil {
		cursorStr := fmt.Sprintf("%d", *lastCheckpoint)
		cursor = &cursorStr
	}

	// Checkpointを取得
	checkpoints, err := w.cl.GetCheckpointSummary(ctx, cursor, w.batch)
	if err != nil { return err }

	var newestCheckpoint int64 = -1
	for _, cp := range checkpoints.Data {
		// SequenceNumberを取得
		var seqNum int64 = -1
		if cp.SequenceNumber != nil {
			if val, ok := cp.SequenceNumber.Value(); ok {
				seqNum = int64(val)
			}
		}
		
		// 既に処理済みのCheckpointはスキップ
		if lastCheckpoint != nil && seqNum <= *lastCheckpoint {
			continue
		}

		// このCheckpoint内のトランザクションを処理
		var timestampMs uint64
		if cp.TimestampMs != nil {
			if val, ok := cp.TimestampMs.Value(); ok {
				timestampMs = val
			}
		}
		for _, txDigest := range cp.Transactions {
			if err := w.processTransaction(ctx, address, txDigest, timestampMs); err != nil {
				log.Printf("[sui] process tx %s: %v", txDigest, err)
			}
		}

		if seqNum > newestCheckpoint {
			newestCheckpoint = seqNum
		}
	}

	// カーソル更新
	if newestCheckpoint >= 0 {
		return w.st.UpdateSuiCursor(ctx, address, newestCheckpoint)
	}
	return nil
}

func (w *SuiWorker) processTransaction(ctx context.Context, address string, txDigest string, timestampMs uint64) error {
	// トランザクションの詳細を取得
	tx, err := w.cl.GetTransactionBlockDetailed(ctx, txDigest)
	if err != nil {
		return err
	}

	// トランザクションが成功していない場合はスキップ
	if tx.Effects.Status.Status != "success" {
		return nil
	}

	// タイムスタンプを設定
	var ts time.Time
	if timestampMs > 0 {
		ts = time.Unix(int64(timestampMs/1000), int64((timestampMs%1000)*1000000)).UTC()
	} else {
		ts = time.Now().UTC()
	}

	// 送信者と受信者を抽出（簡易版）
	var sender, receiver *string
	if len(tx.Transaction.Data.Message.Inputs) > 0 {
		// 最初のInputから送信者を推定
		for _, input := range tx.Transaction.Data.Message.Inputs {
			if input.Type == "pure" && input.ValueType == "address" {
				s := input.Value
				sender = &s
				break
			}
		}
	}

	// ガス代を計算
	var fee *int64
	var totalGas uint64
	if tx.Effects.GasUsed.ComputationCost != nil {
		if val, ok := tx.Effects.GasUsed.ComputationCost.Value(); ok {
			totalGas += val
		}
	}
	if tx.Effects.GasUsed.StorageCost != nil {
		if val, ok := tx.Effects.GasUsed.StorageCost.Value(); ok {
			totalGas += val
		}
	}
	if totalGas > 0 {
		f := int64(totalGas)
		fee = &f
	}

	// メソッド名を抽出（簡易版）
	var method *string
	if len(tx.Transaction.Data.Message.Transactions) > 0 {
		kind := tx.Transaction.Data.Message.Transactions[0].Kind
		method = &kind
	}

	// 生データをJSON化
	raw, _ := json.Marshal(tx)

	// データベースに保存
	return w.st.InsertTxEventSui(ctx, txDigest, ts, sender, receiver, fee, method, raw)
}
