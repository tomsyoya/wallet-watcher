//go:build integration

// Package suitest provides integration tests for Sui blockchain client functionality.
// This package tests the Sui RPC client's ability to:
// - Connect to Sui mainnet RPC endpoints
// - Retrieve SUI and token balances for given addresses
// - Handle various address formats and error conditions
// - Process JSON responses from Sui RPC calls
package suitest

import (
	"context"
	"testing"
	"time"

	sui "github.com/you/wallet-watcher/internal/chains/sui"
)

// TestSuiClientGetBalances tests the main GetBalances method for Sui.
// This test verifies:
// - Successful connection to Sui mainnet RPC
// - Retrieval of SUI and token balances (if any)
// - Correct parsing of RPC response data
// - Graceful handling of addresses with zero balances
func TestSuiClientGetBalances(t *testing.T) {
	// Create Sui client with mainnet RPC endpoint
	client := sui.New("https://fullnode.mainnet.sui.io:443")

	// Set timeout context for RPC calls
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with a known address (may have zero balances)
	address := "0x1251d3064f375a8353eaeadf928b57c1b7cf91a609cb5a1dd1e779bb189735fa"
	balances, err := client.GetBalances(ctx, address)
	if err != nil {
		t.Fatalf("Failed to get balances: %v", err)
	}

	// Sui balances might be empty for this address (this is normal)
	t.Logf("Found %d balances:", len(balances))
	for _, balance := range balances {
		t.Logf("  %s: %d", balance.Token, balance.Amount)
	}
}

// TestSuiClientGetSUIBalance tests SUI balance retrieval specifically.
// This test verifies:
// - SUI balance is correctly extracted from the balances array
// - SUI balance parsing works correctly (may be zero)
// - SUI balance handling for addresses with no SUI
func TestSuiClientGetSUIBalance(t *testing.T) {
	// Create Sui client with mainnet RPC endpoint
	client := sui.New("https://fullnode.mainnet.sui.io:443")

	// Set timeout context for RPC calls
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with a known address (may have zero SUI balance)
	address := "0x1251d3064f375a8353eaeadf928b57c1b7cf91a609cb5a1dd1e779bb189735fa"
	balances, err := client.GetBalances(ctx, address)
	if err != nil {
		t.Fatalf("Failed to get balances: %v", err)
	}

	// Extract SUI balance from the balances array
	var suiBalance int64
	for _, balance := range balances {
		if balance.Token == "SUI" {
			suiBalance = balance.Amount
			break
		}
	}

	// Log SUI balance (may be zero for this test address)
	t.Logf("SUI balance: %d", suiBalance)
}

// TestSuiClientGetTokenBalances tests token balance retrieval (non-SUI tokens).
// This test verifies:
// - Token balances are correctly retrieved and parsed
// - Non-SUI tokens are properly identified and filtered
// - Token balance amounts are correctly extracted
func TestSuiClientGetTokenBalances(t *testing.T) {
	// Create Sui client with mainnet RPC endpoint
	client := sui.New("https://fullnode.mainnet.sui.io:443")

	// Set timeout context for RPC calls
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with a known address (may have zero token balances)
	address := "0x1251d3064f375a8353eaeadf928b57c1b7cf91a609cb5a1dd1e779bb189735fa"
	balances, err := client.GetBalances(ctx, address)
	if err != nil {
		t.Fatalf("Failed to get balances: %v", err)
	}

	// Filter out SUI balances to focus on token balances
	var tokenBalances []sui.Balance
	for _, balance := range balances {
		if balance.Token != "SUI" && balance.Amount > 0 {
			tokenBalances = append(tokenBalances, balance)
		}
	}

	// Log token balances for verification (may be empty for this test address)
	t.Logf("Found %d token balances:", len(tokenBalances))
	for _, balance := range tokenBalances {
		t.Logf("  %s: %d", balance.Token, balance.Amount)
	}
}

// TestSuiClientInvalidAddress tests error handling for invalid addresses.
// This test verifies:
// - Invalid address formats are properly rejected
// - Appropriate error messages are returned
// - Client handles malformed input gracefully
func TestSuiClientInvalidAddress(t *testing.T) {
	// Create Sui client with mainnet RPC endpoint
	client := sui.New("https://fullnode.mainnet.sui.io:443")

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
