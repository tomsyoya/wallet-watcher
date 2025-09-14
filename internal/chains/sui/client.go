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

func (e *rpcError) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
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

/* -------- getCheckpointSummary -------- */

type CheckpointSummary struct {
	SequenceNumber    *Uint64Flex `json:"sequenceNumber"`
	Digest            string `json:"digest"`
	PreviousDigest    string `json:"previousDigest"`
	Epoch             *Uint64Flex `json:"epoch"`
	TimestampMs       *Uint64Flex `json:"timestampMs"`
	Transactions      []string `json:"transactions"`
	CheckpointCommitments []string `json:"checkpointCommitments"`
	ValidatorSignature    string `json:"validatorSignature"`
}

type GetCheckpointSummaryResult struct {
	Data []CheckpointSummary `json:"data"`
	NextCursor *string `json:"nextCursor"`
	HasNextPage bool `json:"hasNextPage"`
}

func (c *Client) GetCheckpointSummary(ctx context.Context, cursor *string, limit int) (*GetCheckpointSummaryResult, error) {
	if limit <= 0 { limit = 10 }
	
	// Sui RPC getCheckpoints の正しいパラメータ形式
	params := []any{
		cursor,  // cursor (string or null)
		limit,   // limit (number)
		true,    // descending order
	}
	
	var result GetCheckpointSummaryResult
	if err := c.call(ctx, "sui_getCheckpoints", params, &result); err != nil {
		return nil, fmt.Errorf("rpc error %d: %s", err.Code, err.Message)
	}
	return &result, nil
}

/* -------- getTransactionBlock (詳細版) -------- */

type TransactionBlockDetailed struct {
	Digest      string `json:"digest"`
	TimestampMs *Uint64Flex `json:"timestampMs"`
	Transaction struct {
		Data struct {
			Message struct {
				Inputs []struct {
					Type string `json:"type"`
					ValueType string `json:"valueType"`
					Value string `json:"value"`
				} `json:"inputs"`
				Transactions []struct {
					Kind string `json:"kind"`
					Data map[string]any `json:"data"`
				} `json:"transactions"`
			} `json:"message"`
		} `json:"data"`
	} `json:"transaction"`
	Effects struct {
		Status struct {
			Status string `json:"status"`
		} `json:"status"`
		GasUsed struct {
			ComputationCost *Uint64Flex `json:"computationCost"`
			StorageCost     *Uint64Flex `json:"storageCost"`
			StorageRebate   *Uint64Flex `json:"storageRebate"`
			NonRefundableStorageFee *Uint64Flex `json:"nonRefundableStorageFee"`
		} `json:"gasUsed"`
		TransactionDigest string `json:"transactionDigest"`
	} `json:"effects"`
}

func (c *Client) GetTransactionBlockDetailed(ctx context.Context, digest string) (*TransactionBlockDetailed, error) {
	params := []any{
		digest,
		map[string]any{
			"showInput": true,
			"showEffects": true,
			"showEvents": true,
			"showObjectChanges": true,
			"showBalanceChanges": true,
		},
	}
	
	var result TransactionBlockDetailed
	if err := c.call(ctx, "sui_getTransactionBlock", params, &result); err != nil {
		return nil, fmt.Errorf("rpc error %d: %s", err.Code, err.Message)
	}
	return &result, nil
}

/* -------- GetBalances -------- */

type Balance struct {
	Token  string `json:"token"`
	Amount int64  `json:"amount"`
}

type getBalanceResp struct {
	Result string `json:"result"`
	Error  *rpcError `json:"error,omitempty"`
}

type getCoinsResp struct {
	Result struct {
		Data []struct {
			CoinType string `json:"coinType"`
			Balance  string `json:"balance"`
		} `json:"data"`
		NextCursor *string `json:"nextCursor"`
		HasNextPage bool `json:"hasNextPage"`
	} `json:"result"`
	Error *rpcError `json:"error,omitempty"`
}

type getOwnedObjectsResp struct {
	Result struct {
		Data []struct {
			Data *struct {
				Fields map[string]interface{} `json:"fields"`
			} `json:"data"`
		} `json:"data"`
		NextCursor *string `json:"nextCursor"`
		HasNextPage bool `json:"hasNextPage"`
	} `json:"result"`
	Error *rpcError `json:"error,omitempty"`
}

func (c *Client) GetBalances(ctx context.Context, address string) ([]Balance, error) {
	var balances []Balance

	// Get SUI balance
	suiBalance, err := c.getSUIBalance(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to get SUI balance: %v", err)
	}
	if suiBalance > 0 {
		balances = append(balances, Balance{
			Token:  "SUI",
			Amount: suiBalance,
		})
	}

	// Get token balances
	tokenBalances, err := c.getTokenBalances(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to get token balances: %v", err)
	}
	balances = append(balances, tokenBalances...)

	return balances, nil
}

func (c *Client) getSUIBalance(ctx context.Context, address string) (int64, error) {
	params := []any{
		address,
		map[string]any{
			"filter": map[string]any{
				"StructType": "0x2::coin::Coin<0x2::sui::SUI>",
			},
			"options": map[string]any{
				"showType":    true,
				"showContent": true,
			},
		},
	}
	
	var result getOwnedObjectsResp
	if err := c.call(ctx, "suix_getOwnedObjects", params, &result); err != nil {
		return 0, err
	}

	var totalBalance int64
	for _, obj := range result.Result.Data {
		if obj.Data != nil && obj.Data.Fields != nil {
			if balance, ok := obj.Data.Fields["balance"].(string); ok {
				if bal, err := strconv.ParseInt(balance, 10, 64); err == nil {
					totalBalance += bal
				}
			}
		}
	}

	return totalBalance, nil
}

func (c *Client) getTokenBalances(ctx context.Context, address string) ([]Balance, error) {
	// For now, return empty balances as Sui token balance retrieval is complex
	// This would require implementing suix_getOwnedObjects with different filters
	// for each token type, which is beyond the scope of this basic implementation
	return []Balance{}, nil
}
