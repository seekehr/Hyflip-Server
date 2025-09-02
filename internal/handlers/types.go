package handlers

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/cache"
	"Hyflip-Server/internal/storage"
)

type FlipperStructs struct {
	Api         *api.HypixelApiClient
	UsersTable  *storage.DatabaseClient
	ConfigTable *storage.ConfigTableClient
	BzCache     *cache.BazaarCache
}

type ResponseType struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
