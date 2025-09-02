package handlers

import (
	"Hyflip-Server/internal/config"
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

		flusher, err := GetSSEFlusher(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, ResponseType{
				Success: false,
				Message: "Invalid flusher provided. Err: " + err.Error(),
				Data:    nil,
			})
		}
		flusher.Flush()

		log.Println("Received request.")
		// previous flips in the update in case we joined mid-update
		snapshot := data.BzCache.Get()
		// subscribe to get live updates now that we've joined the update stream
		liveUpdatesChan := data.BzCache.Subscribe()
		defer func() {
			data.BzCache.Unsubscribe(liveUpdatesChan)
			log.Println("SSE client disconnected. Unsubscribed from live updates.")
		}()

		// send the previous flips first
		for _, flip := range snapshot {
			FilterAndSendFlip(c, flusher, &flip, &conf.BzConfig) // check for user config filter too
		}

		log.Printf("Sent %d flips in initial snapshot.", len(snapshot))

		// waits for: either cache manager to say "updates done" by closing channel, client to disconnect or for a new flip to arrive via the channel we get when we subscribe
		for {
			select {
			// client connection closed
			case <-c.Request().Context().Done():
				return nil

			// new flip OR channel closed
			case flip, ok := <-liveUpdatesChan:
				// channel closed
				if !ok {
					log.Println("Flip stream completed for this update cycle.")
					return nil
				}

				// new flip
				FilterAndSendFlip(c, flusher, &flip, &conf.BzConfig) // check for user config filter too
			}
		}
	}
}

func FilterAndSendFlip(c echo.Context, flusher http.Flusher, f *flippers.BazaarFoundFlip, conf *config.BZConfig) {
	filteredProduct := flippers.Filter(nil, f, conf)
	if filteredProduct == nil {
		return
	}
	jsonFlip, err := json.Marshal(f)
	if err != nil {
		log.Printf("Error marshalling snapshot flip: %v", err)
		return
	}

	fmt.Fprintf(c.Response(), "data: %s\n\n", jsonFlip)
	flusher.Flush()
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
