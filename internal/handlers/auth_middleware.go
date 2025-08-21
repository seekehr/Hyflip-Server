package handlers

import (
	"Hyflip-Server/internal/storage"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
)

func AuthMiddleware(data *RequiredStructs) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			log.Println("uwu (auth here)")
			// userKeyHash
			token := c.Request().Header.Get("Authorization")
			username := c.Request().URL.Query().Get("username")

			if token == "" || username == "" {
				return c.JSON(http.StatusUnauthorized, ResponseType{
					Success: false,
					Message: "unauthorized (token/username not found)",
					Data:    nil,
				})
			}

			exists, err := data.UsersTable.ExistsUser(storage.GetHash(username, token))
			if err != nil || !exists {
				log.Println(err)
				return c.JSON(http.StatusUnauthorized, ResponseType{
					Success: false,
					Message: "invalid token",
					Data:    nil,
				})
			}

			// Continue. our next function (endpoint) will have access to userDb and hypixelApi
			return next(c)
		}
	}
}
