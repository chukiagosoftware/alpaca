package handlers

import (
	"encoding/json"
	"github.com/edamsoft-sre/alpaca/server"
	"net/http"
)

type RootResponse struct {
	Message string `json:"message"`
	Status  bool   `json:"status"`
}

func RootHandler(s server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(RootResponse{
			Message: "Service started",
			Status:  true,
		})
	}
}
