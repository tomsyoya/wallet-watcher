package suitest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sui "github.com/you/wallet-watcher/internal/chains/sui"
)

// TestSuiClient_Mock は、Suiクライアントのモックテストです。
// 実際のSuiネットワークに接続せずに、モックサーバーを使用して以下の動作をテストします：
// - Sui RPCメソッドのモックレスポンス
// - トランザクションディジェストの取得機能
// - トランザクションブロックの取得機能
// - クライアントの基本的な動作確認
//
// このテストは外部ネットワークに依存せず、高速に実行されます。
func TestSuiClient_Mock(t *testing.T) {
	// モックHTTPサーバーのセットアップ
	// Sui RPCエンドポイントを模擬するHTTPサーバーを作成
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// リクエストボディからRPCメソッド名を抽出
		var q struct{ Method string `json:"method"` }
		_ = json.NewDecoder(r.Body).Decode(&q)

		// RPCメソッドに応じてモックレスポンスを返す
		switch q.Method {
		case "suix_queryTransactionBlocks":
			// トランザクションブロッククエリのモックレスポンス
			// テスト用のダミーディジェストを返す
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "id": 1,
				"result": map[string]any{
					"data": []map[string]any{
						{"digest": "MOCK_DIGEST"},
					},
				},
			})
		case "suix_getTransactionBlock":
			// トランザクションブロック取得のモックレスポンス
			// 現在時刻をミリ秒で返す
			nowMs := uint64(time.Now().UnixMilli())
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "id": 1,
				"result": map[string]any{
					"digest":      "MOCK_DIGEST",
					"timestampMs": nowMs,
				},
			})
		default:
			// 未知のメソッドの場合はエラーを返す
			http.Error(w, "unknown method", 400)
		}
	})
	
	// テスト用HTTPサーバーを起動
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// SuiクライアントをモックサーバーのURLで初期化
	cl := sui.New(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// テスト1: トランザクションディジェストの取得
	// 指定されたアドレスに関連するトランザクションディジェストを1件取得
	digest, err := cl.QueryOneTxDigestByToAddress(ctx, "0xabc")
	if err != nil || digest == "" {
		t.Fatalf("query digest err=%v digest=%q", err, digest)
	}
	
	// テスト2: トランザクションブロックの取得
	// 取得したディジェストを使用してトランザクションブロックの詳細を取得
	tb, err := cl.GetTransactionBlock(ctx, digest)
	if err != nil || tb == nil || tb.Digest == "" {
		t.Fatalf("get tx block failed: %v, %+v", err, tb)
	}
}
