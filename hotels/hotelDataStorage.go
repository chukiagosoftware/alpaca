package main

import (
	"github.com/chukiagosoftware/alpaca/database"
	"github.com/chukiagosoftware/alpaca/internal/hotelstorage"
)

type hotelStorage struct {
	*hotelstorage.Storage
}

func newHotelStorage(db *database.DB) *hotelStorage {
	return &hotelStorage{Storage: hotelstorage.NewStorage(db)}
}
