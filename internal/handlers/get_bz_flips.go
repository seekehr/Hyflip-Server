package handlers

import (
	"Hyflip-Server/internal/flippers"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
)

func GetBzFlipsHandler(data *FlipperStructs) echo.HandlerFunc {
	return func(c echo.Context) error {
		userKeyHash := c.Get("user_key_hash")
		if userKeyHash == nil {
			return c.JSON(http.StatusUnauthorized, ResponseType{
				Success: false,
				Message: "Invalid user_key_hash provided (nil)",
				Data:    nil,
			})
		}

		log.Println("Received request.")
		// todo: cache this data.
		conf, err := data.ConfigTable.GetConfig(userKeyHash.(string))
		if err != nil {
			log.Println("Error loading config. Error: " + err.Error())
			return c.JSON(http.StatusUnauthorized, ResponseType{
				Success: false,
				Message: "Request error (Loading Config). Error: " + err.Error(),
				Data:    nil,
			})
		}

		flips, err := flippers.BzFlip(data.Api, &conf.BzConfig)
		if err != nil {
			return c.JSON(500, ResponseType{
				Success: false,
				Message: "Flipper error. Error: " + err.Error(),
				Data:    nil,
			})
		}

		flusher, err := GetSSEFlusher(c)
		if err != nil {
			log.Println("Error getting SSEFlusher. Error: " + err.Error())
			return nil
		}

		for {
			select {
			case <-c.Request().Context().Done(): // client ended the stream
				return nil
			case flip, ok := <-flips:
				if !ok {
					log.Println("Flip stream completed.")
					return nil
				}

				jsonFlip, err := json.Marshal(flip)
				if err != nil {
					log.Printf("Error marshalling JSON: %v", err)
					continue // Log the error and skip this flip, but keep the stream alive
				}

				// Write the data and flush
				fmt.Fprintf(c.Response(), "data: %s\n\n", jsonFlip)
				flusher.Flush()
			}
		}
	}
}

// GetSSEFlusher - Sets the headers to allow server-side events, and gives us the flusher to immediately push data
func GetSSEFlusher(c echo.Context) (http.Flusher, error) {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	/* the flusher is needed because http buffers our responses because an HTTP request for every small request would cause some
	performance issues */
	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("http req doesnt support sse")
	}

	return flusher, nil
}
