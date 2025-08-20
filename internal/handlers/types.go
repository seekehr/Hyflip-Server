package handlers

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/storage"
)

type RequiredStructs struct {
	Api         *api.HypixelApiClient
	UserDb      *storage.UserDatabaseClient
	ConfigTable *storage.ConfigTableClient
}

type ResponseType struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
