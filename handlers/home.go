package handlers

import (
	// "encoding/json"
	"net/http"

	"github.com/edamsoft-sre/alpaca/server"
)

type HomeResponse struct {
	Message string `json:"message"`
	Status  bool   `json:"status"`
}

func HomeHandler(s server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		// send html for login tests
		http.ServeFile(w, r, "test.html")

		// json.NewEncoder(w).Encode(HomeResponse{
		// 	Message: "Welcome to Platzi Go",
		// 	Status:  true,
		// })
	}
}
