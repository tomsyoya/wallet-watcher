package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/you/wallet-watcher/internal/store"
)

type Server struct{ Store *store.Store }

func Routes(s *Server) http.Handler {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	r.Post("/register", s.handleRegister)
	return r
}

type registerReq struct {
	Chain   string `json:"chain"`
	Address string `json:"address"`
}
type registerResp struct{ OK bool `json:"ok"` }

// 簡易バリデーション（PoC用）
var (
	reB58 = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]+$`)  // Solana: Base58 文字集合
	reHex = regexp.MustCompile(`^(0x)?[0-9a-fA-F]{40,64}$`) // Sui: 0x + hex（緩め）
)

func validateChainAndAddress(chain, addr string) error {
	switch strings.ToLower(chain) {
	case "solana":
		if len(addr) < 32 || len(addr) > 64 || !reB58.MatchString(addr) {
			return errors.New("invalid solana address")
		}
	case "sui":
		if !reHex.MatchString(strings.ToLower(addr)) {
			return errors.New("invalid sui address")
		}
	default:
		return errors.New("chain must be 'solana' or 'sui'")
	}
	return nil
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	req.Chain = strings.ToLower(strings.TrimSpace(req.Chain))
	req.Address = strings.TrimSpace(req.Address)

	if err := validateChainAndAddress(req.Chain, req.Address); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := s.Store.UpsertWatchedAddress(ctx, req.Chain, req.Address); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(registerResp{OK: true})
}
