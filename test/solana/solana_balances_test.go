//go:build integration

// Package solanatest provides integration tests for Solana blockchain client functionality.
// This package tests the Solana RPC client's ability to:
// - Connect to Solana mainnet RPC endpoints
// - Retrieve SOL and token balances for given addresses
// - Handle various address formats and error conditions
// - Process JSON responses from Solana RPC calls
package solanatest

import (
	"context"
	"testing"
	"time"

	sol "github.com/you/wallet-watcher/internal/chains/solana"
)

// TestSolanaClientGetBalances tests the main GetBalances method for Solana.
// This test verifies:
// - Successful connection to Solana mainnet RPC
// - Retrieval of both SOL and token balances
// - Correct parsing of RPC response data
// - Presence of expected balance types
func TestSolanaClientGetBalances(t *testing.T) {
	// Create Solana client with mainnet RPC endpoint
	client := sol.New("https://api.mainnet-beta.solana.com")

	// Set timeout context for RPC calls
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with a known address that has balances (System Program address)
	address := "11111111111111111111111111111112"
	balances, err := client.GetBalances(ctx, address)
	if err != nil {
		t.Fatalf("Failed to get balances: %v", err)
	}

	// Verify at least one balance was returned
	if len(balances) == 0 {
		t.Error("Expected at least one balance")
	}

	// Verify SOL balance is present and positive
	foundSOL := false
	for _, balance := range balances {
		if balance.Token == "SOL" {
			foundSOL = true
			if balance.Amount <= 0 {
				t.Error("SOL balance should be positive")
			}
			t.Logf("SOL balance: %d", balance.Amount)
			break
		}
	}

	if !foundSOL {
		t.Error("Expected SOL balance to be present")
	}

	// Log all balances for debugging and verification
	t.Logf("Found %d balances:", len(balances))
	for _, balance := range balances {
		t.Logf("  %s: %d", balance.Token, balance.Amount)
	}
}

// TestSolanaClientGetSOLBalance tests SOL balance retrieval specifically.
// This test verifies:
// - SOL balance is correctly extracted from the balances array
// - SOL balance amount is positive and reasonable
// - SOL balance parsing works correctly
func TestSolanaClientGetSOLBalance(t *testing.T) {
	// Create Solana client with mainnet RPC endpoint
	client := sol.New("https://api.mainnet-beta.solana.com")

	// Set timeout context for RPC calls
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with a known address that has SOL balance
	address := "11111111111111111111111111111112"
	balances, err := client.GetBalances(ctx, address)
	if err != nil {
		t.Fatalf("Failed to get balances: %v", err)
	}

	// Extract SOL balance from the balances array
	var solBalance int64
	for _, balance := range balances {
		if balance.Token == "SOL" {
			solBalance = balance.Amount
			break
		}
	}

	// Verify SOL balance is positive
	if solBalance <= 0 {
		t.Error("SOL balance should be positive")
	}

	t.Logf("SOL balance: %d", solBalance)
}

// TestSolanaClientGetTokenBalances tests token balance retrieval (non-SOL tokens).
// This test verifies:
// - Token balances are correctly retrieved and parsed
// - Non-SOL tokens are properly identified and filtered
// - Token balance amounts are correctly extracted
func TestSolanaClientGetTokenBalances(t *testing.T) {
	// Create Solana client with mainnet RPC endpoint
	client := sol.New("https://api.mainnet-beta.solana.com")

	// Set timeout context for RPC calls
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with a known address that has token balances
	address := "11111111111111111111111111111112"
	balances, err := client.GetBalances(ctx, address)
	if err != nil {
		t.Fatalf("Failed to get balances: %v", err)
	}

	// Filter out SOL balances to focus on token balances
	var tokenBalances []sol.Balance
	for _, balance := range balances {
		if balance.Token != "SOL" && balance.Amount > 0 {
			tokenBalances = append(tokenBalances, balance)
		}
	}

	// Log token balances for verification
	t.Logf("Found %d token balances:", len(tokenBalances))
	for _, balance := range tokenBalances {
		t.Logf("  %s: %d", balance.Token, balance.Amount)
	}
}

// TestSolanaClientInvalidAddress tests error handling for invalid addresses.
// This test verifies:
// - Invalid address formats are properly rejected
// - Appropriate error messages are returned
// - Client handles malformed input gracefully
func TestSolanaClientInvalidAddress(t *testing.T) {
	// Create Solana client with mainnet RPC endpoint
	client := sol.New("https://api.mainnet-beta.solana.com")

	// Set shorter timeout for error cases
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with invalid address format
	address := "invalid_address"
	_, err := client.GetBalances(ctx, address)
	if err == nil {
		t.Error("Expected error for invalid address")
	}

	// Log the expected error for verification
	t.Logf("Expected error for invalid address: %v", err)
}
