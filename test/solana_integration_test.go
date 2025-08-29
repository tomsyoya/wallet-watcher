//go:build integration

package test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	sol "github.com/you/wallet-watcher/internal/chains/solana"
	"github.com/you/wallet-watcher/internal/store"
)

// TestSolana_Integration_One は、Solanaチェーンとの統合テストです。
// 実際のSolanaネットワークに接続して、以下の統合的な動作をテストします：
// - 実際のSolana RPCエンドポイントへの接続
// - データベースへの接続とデータ保存
// - 実際のアドレスから署名履歴の取得
// - 実際のトランザクション詳細の取得
// - 取得したデータのデータベースへの保存
//
// このテストを実行するには、以下の環境変数が必要です：
// - SOLANA_RPC_URL: Solana RPCエンドポイントのURL
// - SOL_ADDR: テスト対象のSolanaアドレス
// - DATABASE_URL: データベース接続文字列

func TestSolana_Integration_One(t *testing.T) {
	// 統合テストに必要な環境変数をチェック
	rpc := os.Getenv("SOLANA_RPC_URL")
	addr := os.Getenv("SOL_ADDR")
	db  := os.Getenv("DATABASE_URL")
	if rpc == "" || addr == "" || db == "" {
		t.Skip("set SOLANA_RPC_URL, SOL_ADDR, DATABASE_URL to run this test")
	}

	ctx := context.Background()
	
	// データベースに接続
	st, err := store.New(ctx)
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}
	defer st.Close()

	// Solanaクライアントを実際のRPCエンドポイントで初期化
	cl := sol.New(rpc)
	cCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// ステップ1: 実際のSolanaアドレスから署名履歴を1件取得
	sigs, err := cl.GetSignaturesForAddress(cCtx, addr, 1)
	if err != nil || len(sigs) == 0 {
		t.Fatalf("GetSignaturesForAddress failed: %v", err)
	}
	sig := sigs[0].Signature

	// ステップ2: 取得した署名を使用してトランザクション詳細を取得
	tx, err := cl.GetTransaction(cCtx, sig)
	if err != nil {
		t.Fatalf("GetTransaction failed: %v", err)
	}

	// ステップ3: トランザクションのブロック時間を処理
	var ts time.Time
	if tx.BlockTime != nil {
		ts = time.Unix(*tx.BlockTime, 0).UTC()
	} else {
		ts = time.Now().UTC()
	}
	raw, _ := json.Marshal(tx)

	// ステップ4: 取得したトランザクションデータをデータベースに保存
	if err := InsertTxEventSolanaForTest(ctx, st, sig, ts, nil, nil, nil, nil, raw); err != nil {
	    t.Fatalf("insert failed: %v", err)
	}
}
