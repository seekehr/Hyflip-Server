package handlers

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/storage"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"strings"
	"sync"
)

const playerFilesUrl = "premium_players.txt"

type RegisteredPlayers struct {
	lock sync.RWMutex // for scalability. not rly needed rn lmao but who cares
}

type CreateAccountRequest struct {
	Username string `json:"username"`
}

func (p *RegisteredPlayers) Read() []string {
	p.lock.Lock()
	defer p.lock.Unlock()
	data, err := os.ReadFile(playerFilesUrl)
	if err != nil {
		panic("could not read premium player files: " + err.Error())
	}

	// convert bytes to string
	content := string(data)
	// split by newlines
	return strings.Split(strings.TrimSpace(content), "\n")
}

func CreateAccountPostHandler(p *RegisteredPlayers, data *RequiredStructs) echo.HandlerFunc {
	return func(c echo.Context) error {
		players := p.Read()
		var req CreateAccountRequest
		if c.Bind(&req) != nil {
			return c.JSON(http.StatusBadRequest, ResponseType{
				Success: false,
				Data:    nil,
				Message: "Invalid username (not provided)",
			})
		}
		username := req.Username

		for _, player := range players {
			if player == username {
				key := uuid.New().String()
				playerUuid, err := api.GetMojangUuid(data.Api, username)
				fmt.Println("nice")
				if err != nil {
					return c.JSON(http.StatusBadRequest, ResponseType{
						Success: false,
						Data:    nil,
						Message: "Invalid username (uuid not found). Error: " + err.Error(),
					})
				}

				err = data.UserDb.CreateUser(storage.GetHash(key, username), playerUuid, username)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, ResponseType{
						Success: false,
						Data:    nil,
						Message: "Error creating user. Error: " + err.Error(),
					})
				}

				return c.JSON(http.StatusOK, ResponseType{
					Success: true,
					Data:    map[string]string{"key": key}, // key will then be stored in the user's pc. and hashed for auth check
					Message: "",
				})
			}
		}

		return c.JSON(http.StatusUnauthorized, ResponseType{
			Success: false,
			Data:    nil,
			Message: "Player is not a premium player",
		})
	}
}
