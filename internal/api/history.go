package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	chain := strings.ToLower(strings.TrimSpace(q.Get("chain")))
	if chain != "solana" && chain != "sui" {
		http.Error(w, "chain must be 'solana' or 'sui'", http.StatusBadRequest)
		return
	}

	addr := strings.TrimSpace(q.Get("address"))
	var addrPtr *string
	if addr != "" {
		addrPtr = &addr
	}

	limit := 50
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	var beforePtr *time.Time
	if v := q.Get("before"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			http.Error(w, "invalid 'before' format (use RFC3339)", http.StatusBadRequest)
			return
		}
		beforePtr = &t
	}

	events, err := s.Store.ListTxEvents(r.Context(), chain, addrPtr, limit, beforePtr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]any{"events": events}
	if len(events) == limit {
		// 次ページ用カーソル（最終行の ts を返す）
		nextBefore := events[len(events)-1].TS.UTC().Format(time.RFC3339)
		resp["next_before"] = nextBefore
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	_ = enc.Encode(resp)
}
