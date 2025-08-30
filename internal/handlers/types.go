package handlers

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/cache"
	"Hyflip-Server/internal/flippers"
	"Hyflip-Server/internal/storage"
)

type FlipperStructs struct {
	Api         *api.HypixelApiClient
	UsersTable  *storage.DatabaseClient
	ConfigTable *storage.ConfigTableClient
	BzCache     *cache.Cache[<-chan flippers.BazaarFoundFlip]
}

type ResponseType struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
