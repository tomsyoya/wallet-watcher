package sui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	URL    string
	client *http.Client
}

func New(url string) *Client {
	return &Client{
		URL: url,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

type rpcRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type rpcEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error,omitempty"`
}

func (c *Client) call(ctx context.Context, method string, params any, out any) *rpcError {
	b, _ := json.Marshal(rpcRequest{Jsonrpc: "2.0", ID: 1, Method: method, Params: params})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return &rpcError{Code: -1, Message: err.Error()}
	}
	defer res.Body.Close()

	var env rpcEnvelope
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return &rpcError{Code: -2, Message: err.Error()}
	}
	if env.Error != nil {
		return env.Error
	}
	if out != nil && len(env.Result) > 0 {
		if err := json.Unmarshal(env.Result, out); err != nil {
			return &rpcError{Code: -3, Message: err.Error()}
		}
	}
	return nil
}

/* -------- queryTransactionBlocks (suix -> sui フォールバック) -------- */

type queryTxBlocksResult struct {
	Data []struct {
		Digest string `json:"digest"`
	} `json:"data"`
	// nextCursor等は省略
}

func (c *Client) QueryOneTxDigestByToAddress(ctx context.Context, address string) (string, error) {
	// suix_* 形の引数
	suixParams := []any{
		map[string]any{
			"filter": map[string]any{"ToAddress": address},
			"options": map[string]any{"showInput": false, "showDigest": true},
			"limit":   1,
			"cursor":  nil,
			"order":   "descending",
		},
	}
	var out queryTxBlocksResult
	if err := c.call(ctx, "suix_queryTransactionBlocks", suixParams, &out); err == nil {
		if len(out.Data) > 0 {
			return out.Data[0].Digest, nil
		}
		return "", nil
	} else if err.Code != -32601 {
		return "", fmt.Errorf("rpc error %d: %s", err.Code, err.Message)
	}

	// フォールバック: sui_queryTransactionBlocks は引数仕様が異なる場合があるので最小で
	suiParams := []any{
		map[string]any{
			"ToAddress": address,
		},
		nil, // cursor
		1,   // limit
		true, // descending
	}
	out = queryTxBlocksResult{}
	if err := c.call(ctx, "sui_queryTransactionBlocks", suiParams, &out); err != nil {
		return "", fmt.Errorf("rpc error %d: %s", err.Code, err.Message)
	}
	if len(out.Data) == 0 {
		return "", nil
	}
	return out.Data[0].Digest, nil
}

/* -------- getTransactionBlock (suix -> sui フォールバック) -------- */

type TransactionBlock struct {
	TimestampMs *Uint64Flex `json:"timestampMs"`
	Digest      string  `json:"digest"`
}

func (c *Client) GetTransactionBlock(ctx context.Context, digest string) (*TransactionBlock, error) {
	suixParams := []any{
		digest,
		map[string]any{
			"showDigest": true,
		},
	}
	var tb TransactionBlock
	if err := c.call(ctx, "suix_getTransactionBlock", suixParams, &tb); err == nil {
		return &tb, nil
	} else if err.Code != -32601 {
		return nil, fmt.Errorf("rpc error %d: %s", err.Code, err.Message)
	}

	// フォールバック: sui_getTransactionBlock は params 形が近い
	suiParams := suixParams
	tb = TransactionBlock{}
	if err := c.call(ctx, "sui_getTransactionBlock", suiParams, &tb); err != nil {
		return nil, fmt.Errorf("rpc error %d: %s", err.Code, err.Message)
	}
	return &tb, nil
}

// 数値/文字列/NULL を許容する柔軟型
type Uint64Flex uint64
func (u *Uint64Flex) UnmarshalJSON(b []byte) error {
    // "null"
    if bytes.Equal(b, []byte("null")) { return nil }
    // 数値
    var n uint64
    if err := json.Unmarshal(b, &n); err == nil {
        *u = Uint64Flex(n)
        return nil
    }
    // 文字列
    var s string
    if err := json.Unmarshal(b, &s); err == nil {
        if s == "" { return nil }
        v, err := strconv.ParseUint(s, 10, 64)
        if err != nil { return err }
        *u = Uint64Flex(v)
        return nil
    }
    return fmt.Errorf("invalid Uint64Flex: %s", string(b))
}
func (u *Uint64Flex) Value() (uint64, bool) {
    if u == nil { return 0, false }
    return uint64(*u), true
}
