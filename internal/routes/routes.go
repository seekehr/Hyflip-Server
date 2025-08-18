package routes

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/handlers"
	"Hyflip-Server/internal/storage"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func RegisterRoutes(e *echo.Echo, userDb *storage.UserDatabaseClient, hypixelApi *api.HypixelApiClient) {
	e.Use(middleware.CORSWithConfig(
		middleware.CORSConfig{
			AllowOrigins: []string{"*"},
		}))
	e.POST("/create_account", handlers.CreateAccountPostHandler(&handlers.RegisteredPlayers{}, userDb, hypixelApi))
	protected := e.Group("/api/")
	protected.Use(handlers.AuthMiddleware(userDb, hypixelApi))
}
