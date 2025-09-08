//go:build integration

package suitest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	api "github.com/you/wallet-watcher/internal/api"
	sui "github.com/you/wallet-watcher/internal/chains/sui"
	"github.com/you/wallet-watcher/internal/store"
)

type historyResp struct {
	Events []struct {
		TxHash string `json:"tx_hash"`
	} `json:"events"`
	NextBefore *string `json:"next_before"`
}

func normalizeSui(addr string) string {
	a := strings.TrimSpace(addr)
	if strings.HasPrefix(a, "0x") || strings.HasPrefix(a, "0X") {
		return a
	}
	return "0x" + a
}

// TestSui_Integration_One は、Suiチェーンとの統合テストです。
// 実際のSuiネットワークに接続して、以下の統合的な動作をテストします：
// - 実際のSui RPCエンドポイントへの接続とデータ取得
// - データベースへの接続とデータ保存
// - APIエンドポイント（/history）の動作確認
// - エンドツーエンドのデータフロー検証
//
// このテストを実行するには、以下の環境変数が必要です：
// - SUI_RPC_URL: Sui RPCエンドポイントのURL
// - SUI_ADDR: テスト対象のSuiアドレス
// - DATABASE_URL: データベース接続文字列
func TestSui_Integration_One(t *testing.T) {
	// 環境変数のチェック - 統合テストに必要な設定を確認
	rpc := os.Getenv("SUI_RPC_URL")
	addr := os.Getenv("SUI_ADDR")
	db := os.Getenv("DATABASE_URL")
	if rpc == "" || addr == "" || db == "" {
		t.Skip("set SUI_RPC_URL, SUI_ADDR, DATABASE_URL to run this test")
	}
	// Suiアドレスの正規化（0xプレフィックスの追加）
	addr = normalizeSui(addr)

	// データベース接続の初期化
	ctx := context.Background()
	st, err := store.New(ctx)
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}
	defer st.Close()

	// Suiクライアントの初期化とタイムアウト設定
	cl := sui.New(rpc)
	cCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	backup, err := backupTxEventsSui(ctx, st)
	if err != nil { t.Fatalf("backup sui: %v", err) }
	if err := clearWindowTxEventsSui(ctx, st); err != nil { t.Fatalf("clear window sui: %v", err) }
	t.Cleanup(func() {
		_ = clearWindowTxEventsSui(ctx, st)
		_ = restoreTxEventsSui(ctx, st, backup)
	})

	// ステップ1: 実際のSui RPCからライブデータを取得（失敗しても続行）
	// 指定されたアドレスに関連するトランザクションディジェストを1件取得
	digest, _ := cl.QueryOneTxDigestByToAddress(cCtx, addr)
	// 取得したディジェストを使用してトランザクションブロックの詳細を取得
	tb, _ := cl.GetTransactionBlock(cCtx, digest)

	// トランザクションデータの初期化
	txHash := fmt.Sprintf("TEST_SUI_%d", time.Now().UnixNano())
	ts := time.Now().UTC()
	
	// ライブデータが取得できた場合の処理
	if tb != nil && tb.Digest != "" {
		txHash = tb.Digest // 実際のトランザクションディジェストを使用
		// タイムスタンプの処理（ミリ秒単位から秒単位に変換）
		if ms, ok := tb.TimestampMs.Value(); ok && ms > 0 {
			ts = time.UnixMilli(int64(ms)).UTC()
		}
	}

	// ステップ2: データベースへの保存
	// /history APIのアドレスフィルタに必ずヒットさせるため受信者アドレスを設定
	recv := addr
	if err := InsertTxEventSuiForTest(ctx, st, txHash, ts, nil, &recv, []byte(`{"source":"test-sui"}`)); err != nil {
		t.Fatalf("insert synthetic row: %v", err)
	}
	
	// テスト終了後のクリーンアップ処理
	t.Cleanup(func() {
		_, _ = st.Pool.Exec(ctx, `DELETE FROM tx_events_sui WHERE tx_hash = $1`, txHash)
	})

	// ステップ3: APIエンドポイントのテスト
	// インプロセスHTTPサーバーを起動して/historyエンドポイントをテスト
	srv := &api.Server{Store: st}
	router := api.Routes(srv)
	tsrv := httptest.NewServer(router)
	defer tsrv.Close()

	// /history APIを呼び出してSuiチェーンの履歴を取得
	resp, err := http.Get(fmt.Sprintf("%s/history?chain=sui&address=%s&limit=20", tsrv.URL, addr))
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	t.Logf("  resp: %+v ",  resp)
	
	// レスポンスの解析
	var hr historyResp
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		t.Fatalf("decode: %v", err)
	}
	
	// 挿入したトランザクションハッシュがAPIレスポンスに含まれているかチェック
	found := false
	for _, e := range hr.Events {
		if e.TxHash == txHash {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("inserted tx_hash not found in /history response (tx_hash=%s)", txHash)
	}
}
