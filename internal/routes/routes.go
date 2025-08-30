package routes

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/cache"
	"Hyflip-Server/internal/flippers"
	"Hyflip-Server/internal/handlers"
	"Hyflip-Server/internal/storage"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func RegisterRoutes(e *echo.Echo, userDb *storage.DatabaseClient, hypixelApi *api.HypixelApiClient, configTable *storage.ConfigTableClient, bzCache *cache.Cache[flippers.BazaarFoundFlip]) {
	reqStruct := &handlers.FlipperStructs{
		Api:         hypixelApi,
		UsersTable:  userDb,
		ConfigTable: configTable,
		BzCache:     bzCache,
	}

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:8080"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT},
	}))

	e.POST("/create_account", handlers.CreateAccountPostHandler(&handlers.RegisteredPlayers{}, reqStruct))
	protected := e.Group("/api/")
	protected.Use(handlers.AuthMiddleware(reqStruct))
	protected.GET("bzflips", handlers.GetBzFlipsHandler(reqStruct))
}
