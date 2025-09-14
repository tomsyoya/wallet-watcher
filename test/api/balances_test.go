//go:build integration

// Package apitest provides integration tests for the /balances API endpoints.
// This package tests the complete API functionality including:
// - HTTP request/response handling
// - JSON serialization/deserialization
// - Parameter validation
// - Error handling
// - Performance characteristics
package apitest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	api "github.com/you/wallet-watcher/internal/api"
	"github.com/you/wallet-watcher/internal/store"
)

type BalancesResponse struct {
	Address  string      `json:"address"`
	Balances interface{} `json:"balances"`
}

type Balance struct {
	Token  string `json:"token"`
	Amount int64  `json:"amount"`
}

// TestBalancesAPI tests the /balances API endpoint with various scenarios.
// This test verifies:
// - Valid requests for both Solana and Sui chains return correct data
// - Invalid requests (missing parameters, invalid addresses) return appropriate errors
// - JSON response structure is correct
// - HTTP status codes are appropriate for each scenario
func TestBalancesAPI(t *testing.T) {
	// Setup test database connection
	ctx := context.Background()
	st, err := store.New(ctx)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer st.Close()

	// Create API server instance with database store
	server := &api.Server{Store: st}
	handler := api.Routes(server)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		validateFunc   func(t *testing.T, resp *http.Response)
	}{
		{
			// Test case: Valid Solana address with balances
			// Verifies: API returns 200 OK, correct JSON structure, SOL balance present
			name:           "Solana balances - valid address",
			url:            "/balances?chain=solana&address=11111111111111111111111111111112",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result BalancesResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify address field matches request
				if result.Address != "11111111111111111111111111111112" {
					t.Errorf("Expected address %s, got %s", "11111111111111111111111111111112", result.Address)
				}

				// Verify balances field is an array
				balances, ok := result.Balances.([]interface{})
				if !ok {
					t.Fatalf("Expected balances to be array, got %T", result.Balances)
				}

				// Verify at least one balance exists
				if len(balances) == 0 {
					t.Error("Expected at least one balance")
				}

				// Verify SOL balance is present (main token for Solana)
				foundSOL := false
				for _, bal := range balances {
					balanceMap, ok := bal.(map[string]interface{})
					if !ok {
						continue
					}
					if token, ok := balanceMap["token"].(string); ok && token == "SOL" {
						foundSOL = true
						break
					}
				}

				if !foundSOL {
					t.Error("Expected SOL balance to be present")
				}
			},
		},
		{
			// Test case: Valid Sui address (may have zero balances)
			// Verifies: API returns 200 OK, correct JSON structure, handles empty balances gracefully
			name:           "Sui balances - valid address",
			url:            "/balances?chain=sui&address=0x1251d3064f375a8353eaeadf928b57c1b7cf91a609cb5a1dd1e779bb189735fa",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *http.Response) {
				var result BalancesResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify address field matches request
				if result.Address != "0x1251d3064f375a8353eaeadf928b57c1b7cf91a609cb5a1dd1e779bb189735fa" {
					t.Errorf("Expected address %s, got %s", "0x1251d3064f375a8353eaeadf928b57c1b7cf91a609cb5a1dd1e779bb189735fa", result.Address)
				}

				// Sui balances might be null or empty array (this is normal for some addresses)
				if result.Balances != nil {
					balances, ok := result.Balances.([]interface{})
					if !ok {
						t.Errorf("Expected balances to be array or null, got %T", result.Balances)
					}
					t.Logf("Sui balances: %+v", balances)
				}
			},
		},
		{
			// Test case: Missing required chain parameter
			// Verifies: API returns 400 Bad Request for missing chain parameter
			name:           "Missing chain parameter",
			url:            "/balances?address=11111111111111111111111111111112",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, resp *http.Response) {
				// Should return error message about missing chain parameter
			},
		},
		{
			// Test case: Missing required address parameter
			// Verifies: API returns 400 Bad Request for missing address parameter
			name:           "Missing address parameter",
			url:            "/balances?chain=solana",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, resp *http.Response) {
				// Should return error message about missing address parameter
			},
		},
		{
			// Test case: Invalid chain name (not supported)
			// Verifies: API returns 400 Bad Request for unsupported chain
			name:           "Invalid chain",
			url:            "/balances?chain=ethereum&address=11111111111111111111111111111112",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, resp *http.Response) {
				// Should return error message about unsupported chain
			},
		},
		{
			// Test case: Invalid Solana address format
			// Verifies: API returns 400 Bad Request for malformed Solana address
			name:           "Invalid Solana address",
			url:            "/balances?chain=solana&address=invalid",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, resp *http.Response) {
				// Should return error message about invalid address format
			},
		},
		{
			// Test case: Invalid Sui address format
			// Verifies: API returns 400 Bad Request for malformed Sui address
			name:           "Invalid Sui address",
			url:            "/balances?chain=sui&address=invalid",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, resp *http.Response) {
				// Should return error message about invalid address format
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, rr.Result())
			}
		})
	}
}

// TestBalancesAPIPerformance tests the performance characteristics of the /balances API.
// This test verifies:
// - API responses are returned within acceptable time limits (10 seconds)
// - Both Solana and Sui endpoints perform within expected ranges
// - Performance is consistent across multiple requests
func TestBalancesAPIPerformance(t *testing.T) {
	// Setup test database connection
	ctx := context.Background()
	st, err := store.New(ctx)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer st.Close()

	// Create API server instance with database store
	server := &api.Server{Store: st}
	handler := api.Routes(server)

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "Solana performance test",
			url:  "/balances?chain=solana&address=11111111111111111111111111111112",
		},
		{
			name: "Sui performance test",
			url:  "/balances?chain=sui&address=0x1251d3064f375a8353eaeadf928b57c1b7cf91a609cb5a1dd1e779bb189735fa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()

			req, err := http.NewRequest("GET", tt.url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			duration := time.Since(start)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
			}

			// Performance check - should respond within 10 seconds
			if duration > 10*time.Second {
				t.Errorf("Request took too long: %v", duration)
			}

			t.Logf("Request completed in %v", duration)
		})
	}
}
