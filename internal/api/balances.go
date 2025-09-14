package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	solana "github.com/you/wallet-watcher/internal/chains/solana"
	sui "github.com/you/wallet-watcher/internal/chains/sui"
)

// BalancesResponse represents the response for /balances endpoint
type BalancesResponse struct {
	Address  string      `json:"address"`
	Balances interface{} `json:"balances"`
}

// SolanaBalanceHandler handles Solana balance requests
func (s *Server) handleSolanaBalances(w http.ResponseWriter, r *http.Request) {
	address := chi.URLParam(r, "address")
	if address == "" {
		http.Error(w, "address parameter is required", http.StatusBadRequest)
		return
	}

	// Validate Solana address
	if err := validateChainAndAddress("solana", address); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Get Solana RPC URL from environment
	rpcURL := r.URL.Query().Get("rpc_url")
	if rpcURL == "" {
		rpcURL = "https://api.mainnet-beta.solana.com"
	}

	client := solana.New(rpcURL)
	balances, err := client.GetBalances(ctx, address)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get balances: %v", err), http.StatusInternalServerError)
		return
	}

	response := BalancesResponse{
		Address:  address,
		Balances: balances,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// SuiBalanceHandler handles Sui balance requests
func (s *Server) handleSuiBalances(w http.ResponseWriter, r *http.Request) {
	address := chi.URLParam(r, "address")
	if address == "" {
		http.Error(w, "address parameter is required", http.StatusBadRequest)
		return
	}

	// Validate Sui address
	if err := validateChainAndAddress("sui", address); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Get Sui RPC URL from environment
	rpcURL := r.URL.Query().Get("rpc_url")
	if rpcURL == "" {
		rpcURL = "https://fullnode.mainnet.sui.io:443"
	}

	client := sui.New(rpcURL)
	balances, err := client.GetBalances(ctx, address)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get balances: %v", err), http.StatusInternalServerError)
		return
	}

	response := BalancesResponse{
		Address:  address,
		Balances: balances,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GenericBalanceHandler handles balance requests for both chains
func (s *Server) handleBalances(w http.ResponseWriter, r *http.Request) {
	chain := strings.ToLower(r.URL.Query().Get("chain"))
	address := r.URL.Query().Get("address")

	if chain == "" || address == "" {
		http.Error(w, "chain and address parameters are required", http.StatusBadRequest)
		return
	}

	// Validate chain and address
	if err := validateChainAndAddress(chain, address); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var balances interface{}
	var err error

	switch chain {
	case "solana":
		// Get Solana RPC URL from environment
		rpcURL := r.URL.Query().Get("rpc_url")
		if rpcURL == "" {
			rpcURL = "https://api.mainnet-beta.solana.com"
		}
		client := solana.New(rpcURL)
		balances, err = client.GetBalances(ctx, address)
	case "sui":
		// Get Sui RPC URL from environment
		rpcURL := r.URL.Query().Get("rpc_url")
		if rpcURL == "" {
			rpcURL = "https://fullnode.mainnet.sui.io:443"
		}
		client := sui.New(rpcURL)
		balances, err = client.GetBalances(ctx, address)
	default:
		http.Error(w, "chain must be 'solana' or 'sui'", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get balances: %v", err), http.StatusInternalServerError)
		return
	}

	response := BalancesResponse{
		Address:  address,
		Balances: balances,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
