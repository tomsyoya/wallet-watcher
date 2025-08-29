package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sui "github.com/you/wallet-watcher/internal/chains/sui"
)

func TestSuiClient_Mock(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var q struct{ Method string `json:"method"` }
		_ = json.NewDecoder(r.Body).Decode(&q)

		switch q.Method {
		case "suix_queryTransactionBlocks":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "id": 1,
				"result": map[string]any{
					"data": []map[string]any{
						{"digest": "MOCK_DIGEST"},
					},
				},
			})
		case "suix_getTransactionBlock":
			nowMs := uint64(time.Now().UnixMilli())
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "id": 1,
				"result": map[string]any{
					"digest":      "MOCK_DIGEST",
					"timestampMs": nowMs,
				},
			})
		default:
			http.Error(w, "unknown method", 400)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cl := sui.New(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	digest, err := cl.QueryOneTxDigestByToAddress(ctx, "0xabc")
	if err != nil || digest == "" {
		t.Fatalf("query digest err=%v digest=%q", err, digest)
	}
	tb, err := cl.GetTransactionBlock(ctx, digest)
	if err != nil || tb == nil || tb.Digest == "" {
		t.Fatalf("get tx block failed: %v, %+v", err, tb)
	}
}
