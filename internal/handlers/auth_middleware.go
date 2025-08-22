package handlers

import (
	"Hyflip-Server/internal/storage"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
)

func AuthMiddleware(data *RequiredStructs) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			fmt.Println("Going through auth.")
			// userKeyHash
			authHeader := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, ResponseType{
					Success: false,
					Message: "unauthorized (token header malformed)",
					Data:    nil,
				})
			}
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			username := strings.TrimSpace(c.Request().URL.Query().Get("username"))

			if token == "" || username == "" {
				return c.JSON(http.StatusUnauthorized, ResponseType{
					Success: false,
					Message: "unauthorized (token/username not found)",
					Data:    nil,
				})
			}

			hash := storage.GetHash(token, username)
			exists, err := data.UsersTable.ExistsUser(hash)
			if err != nil || !exists {
				return c.JSON(http.StatusUnauthorized, ResponseType{
					Success: false,
					Message: "invalid token",
					Data:    nil,
				})
			}

			fmt.Println("Succeeded auth.")
			c.Set("user_key_hash", hash)
			// Continue. our next function (endpoint) will have access to userDb and hypixelApi
			return next(c)
		}
	}
}
