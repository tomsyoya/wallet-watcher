package api


import (
"context"
"encoding/json"
"net/http"


"github.com/you/wallet-watcher/internal/store"
)


type RegisterHandler struct {
Store *store.Store
}


type registerRequest struct {
Chain string `json:"chain"`
Address string `json:"address"`
}


type registerResponse struct {
Status string `json:"status"`
Chain string `json:"chain"`
Address string `json:"address"`
}


func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
var req registerRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
http.Error(w, "invalid json", http.StatusBadRequest)
return
}


// 簡易バリデーション
if req.Chain != "solana" && req.Chain != "sui" {
http.Error(w, "invalid chain", http.StatusBadRequest)
return
}
if len(req.Address) < 10 { // 最小の簡易チェック
http.Error(w, "invalid address", http.StatusBadRequest)
return
}


err := h.Store.UpsertRegistration(context.Background(), store.Registration{
Chain: req.Chain,
Address: req.Address,
})
if err != nil {
http.Error(w, err.Error(), http.StatusInternalServerError)
return
}


w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(registerResponse{
Status: "ok",
Chain: req.Chain,
Address: req.Address,
})
}

