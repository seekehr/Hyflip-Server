package handlers

import (
	"Hyflip-Server/internal/storage"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"strings"
)

func AuthMiddleware(data *RequiredStructs) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// userKeyHash
			authHeader := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return c.String(http.StatusUnauthorized, "Malformed Authorization header: missing 'Bearer ' prefix")
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			username := c.Request().URL.Query().Get("username")

			if token == "" || username == "" {
				return c.JSON(http.StatusUnauthorized, ResponseType{
					Success: false,
					Message: "unauthorized (token/username not found)",
					Data:    nil,
				})
			}

			hash := storage.GetHash(username, token)
			exists, err := data.UsersTable.ExistsUser(hash)
			if err != nil || !exists {
				log.Println(err)
				return c.JSON(http.StatusUnauthorized, ResponseType{
					Success: false,
					Message: "invalid token",
					Data:    nil,
				})
			}

			c.Set("user_key_hash", hash)
			// Continue. our next function (endpoint) will have access to userDb and hypixelApi
			return next(c)
		}
	}
}
