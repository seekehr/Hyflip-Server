package handlers

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/storage"
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
				if err != nil {
					return c.JSON(http.StatusBadRequest, ResponseType{
						Success: false,
						Data:    nil,
						Message: "Invalid username (uuid not found). Error: " + err.Error(),
					})
				}

				hash := storage.GetHash(key, username)
				respChan := make(chan *ResponseType, 2)
				// Save user in the user table
				go func() {
					err = data.UsersTable.CreateUser(hash, playerUuid, username)
					if err != nil {
						respChan <- &ResponseType{
							Success: false,
							Data:    nil,
							Message: "Retry. Error creating user. Error: " + err.Error(),
						}
						return
					}
					respChan <- nil
				}()

				// Save user default config
				go func() {
					err := data.ConfigTable.SaveDefaultConfig(hash)
					if err != nil {
						respChan <- &ResponseType{
							Success: false,
							Data:    nil,
							Message: "Retry. Error saving config: " + err.Error(),
						}
						return
					}
					respChan <- nil
				}()

				// collect both results
				for i := 0; i < 2; i++ {
					if response := <-respChan; response != nil {
						return c.JSON(http.StatusInternalServerError, response)
					}
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
