package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/edamsoft-sre/alpaca/models"
	"github.com/edamsoft-sre/alpaca/server"
	"github.com/gorilla/mux"
)

type HotelListResponse struct {
	Hotels []*models.HotelAPIItem `json:"hotels"`
	Total  int64                  `json:"total"`
	Page   uint64                 `json:"page"`
	Limit  uint64                 `json:"limit"`
}

type HotelResponse struct {
	Hotel *models.HotelAPIItem `json:"hotel"`
}

func ListHotelsHandler(s server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")

		var page uint64 = 0
		var limit uint64 = 10

		if pageStr != "" {
			if p, err := strconv.ParseUint(pageStr, 10, 64); err == nil {
				page = p
			}
		}

		if limitStr != "" {
			if l, err := strconv.ParseUint(limitStr, 10, 64); err == nil {
				limit = l
			}
		}

		hotels, err := s.HotelService().List(r.Context(), page, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		total, err := s.HotelService().Count(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HotelListResponse{
			Hotels: hotels,
			Total:  total,
			Page:   page,
			Limit:  limit,
		})
	}
}

func GetHotelByIDHandler(s server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		hotelID := params["hotelId"]

		hotel, err := s.HotelService().GetByID(r.Context(), hotelID)
		if err != nil {
			http.Error(w, "Hotel not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HotelResponse{
			Hotel: hotel,
		})
	}
}

func GetHotelsByCityHandler(s server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		cityName := params["cityName"]

		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")

		var page uint64 = 0
		var limit uint64 = 10

		if pageStr != "" {
			if p, err := strconv.ParseUint(pageStr, 10, 64); err == nil {
				page = p
			}
		}

		if limitStr != "" {
			if l, err := strconv.ParseUint(limitStr, 10, 64); err == nil {
				limit = l
			}
		}

		hotels, err := s.HotelService().GetByCity(r.Context(), cityName, page, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HotelListResponse{
			Hotels: hotels,
			Page:   page,
			Limit:  limit,
		})
	}
}
