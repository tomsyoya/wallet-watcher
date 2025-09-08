package solanatest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sol "github.com/you/wallet-watcher/internal/chains/solana"
)

// TestSolanaClient_Mock は、Solanaチェーンクライアントのモックテストです。
// 実際のSolanaネットワークに接続せずに、HTTPモックサーバーを使用して
// 以下の機能をテストします：
// - GetSignaturesForAddress: アドレスの署名履歴取得
// - GetTransaction: トランザクション詳細の取得
// このテストにより、Solanaクライアントの実装が正しく動作することを確認できます。

func TestSolanaClient_Mock(t *testing.T) {
	// HTTPモックサーバーをセットアップ
	// Solana RPC APIのレスポンスをシミュレート
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		type req struct {
			Method string `json:"method"`
		}
		var q req
		_ = json.NewDecoder(r.Body).Decode(&q)

		switch q.Method {
		case "getSignaturesForAddress":
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "id": 1,
				"result": []map[string]any{{
					"signature": "MOCK_SIG",
					"slot":      123,
					"blockTime": time.Now().Unix(),
				}},
			})
		case "getTransaction":
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "id": 1,
				"result": map[string]any{
					"slot":      123,
					"blockTime": time.Now().Unix(),
					"meta": map[string]any{
						"fee": 1000,
					},
					"transaction": map[string]any{
						"message": map[string]any{
							"accountKeys": []string{"Sender111", "Receiver222"},
						},
						"signatures": []string{"MOCK_SIG"},
					},
					"version": 0,
				},
			})
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// SolanaクライアントをモックサーバーのURLで初期化
	cl := sol.New(srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// テスト1: アドレスの署名履歴取得をテスト
	sigs, err := cl.GetSignaturesForAddress(ctx, "dummy", 1)
	if err != nil || len(sigs) == 0 {
		t.Fatalf("GetSignaturesForAddress failed: %v", err)
	}

	// テスト2: トランザクション詳細の取得をテスト
	tx, err := cl.GetTransaction(ctx, sigs[0].Signature)
	if err != nil {
		t.Fatalf("GetTransaction failed: %v", err)
	}
	if len(tx.Transaction.Message.AccountKeys) < 2 {
		t.Fatalf("unexpected tx: %+v", tx)
	}
}
