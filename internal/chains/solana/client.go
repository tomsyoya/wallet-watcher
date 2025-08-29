package solana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	URL    string
	client *http.Client
}

func New(url string) *Client {
	return &Client{
		URL: url,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ---- JSON-RPC payload ----
type rpcRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type SigInfo struct {
	Signature string  `json:"signature"`
	Slot      uint64  `json:"slot"`
	ERR       *any    `json:"err"`
	Memo      *string `json:"memo"`
	BlockTime *int64  `json:"blockTime"`
}

type getSigsResp struct {
	Result []SigInfo     `json:"result"`
	Error  *rpcErrorBody `json:"error,omitempty"`
}
type rpcErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *Client) GetSignaturesForAddress(ctx context.Context, address string, limit int) ([]SigInfo, error) {
	if limit <= 0 {
		limit = 1
	}
	req := rpcRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "getSignaturesForAddress",
		Params: []interface{}{
			address,
			map[string]int{"limit": limit},
		},
	}
	b, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(b))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out getSigsResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", out.Error.Code, out.Error.Message)
	}
	return out.Result, nil
}

// ---- getTransaction ----

type getTxResp struct {
	Result *TransactionWithMeta `json:"result"`
	Error  *rpcErrorBody        `json:"error,omitempty"`
}

type TransactionWithMeta struct {
	Slot        uint64         `json:"slot"`
	BlockTime   *int64         `json:"blockTime"`
	Meta        *TxMeta        `json:"meta"`
	Transaction EncodedTx      `json:"transaction"`
	Version     interface{}    `json:"version"`
}
type TxMeta struct {
	Fee         uint64      `json:"fee"`
	Err         interface{} `json:"err"`
	PreBalances []uint64    `json:"preBalances"`
	PostBalances []uint64   `json:"postBalances"`
}
type EncodedTx struct {
	Message struct {
		AccountKeys []string `json:"accountKeys"`
	} `json:"message"`
	Signatures []string `json:"signatures"`
}

func (c *Client) GetTransaction(ctx context.Context, signature string) (*TransactionWithMeta, error) {
	req := rpcRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "getTransaction",
		Params: []interface{}{
			signature,
			map[string]interface{}{
				"encoding": "json",        // human friendly
				"maxSupportedTransactionVersion": 0,
			},
		},
	}
	b, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(b))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out getTxResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", out.Error.Code, out.Error.Message)
	}
	return out.Result, nil
}
