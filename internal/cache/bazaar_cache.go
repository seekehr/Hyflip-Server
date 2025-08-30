package cache

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/config"
	"Hyflip-Server/internal/flippers"
	"log"
	"sync/atomic"
	"time"
)

type BazaarCache struct {
	DataChannel chan flippers.BazaarFoundFlip
	api         *api.HypixelApiClient
	expiryTime  time.Duration
	lastUpdate  atomic.Int64
	isUpdating  atomic.Bool
}

func NewBazaarCache(apiClient *api.HypixelApiClient, expiryTime time.Duration) *BazaarCache {
	bzCache := &BazaarCache{
		DataChannel: make(chan flippers.BazaarFoundFlip, 700), // prob dont need that many lol but who caressss
		api:         apiClient,
		expiryTime:  expiryTime,
		lastUpdate:  atomic.Int64{},
		isUpdating:  atomic.Bool{},
	}
	return bzCache
}

func (c *BazaarCache) startUpdateGoroutine() {
	// checks every e.g 5s if the expiry time is 20s to see if the expiry time is reached. arbitrary number, but lower will mostly always be better ig
	ticker := time.NewTicker(c.expiryTime / 4)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if c.isExpired() {
				if c.isUpdating.CompareAndSwap(false, true) {
					log.Println("Cannot update bazaar cache. Under update rn...")
					continue
				}

				c.lastUpdate.Store(time.Now().Unix()) // we want to update regardless of if the function was successful
				// generating default bz config because
				chn, err := flippers.BzFlip(c.api, config.GenerateDefaultBZConfig())
				c.isUpdating.Store(false)
				if err != nil {
					log.Println("Error updating bazaar cache. Err: " + err.Error())
					continue
				}
				for foundFlip := range chn {
					c.DataChannel <- foundFlip
				}
			}
		}
	}
}

// isExpired checks if the cache is expired, used by cache updating goroutine to automatically update expired cache.
func (c *BazaarCache) isExpired() bool {
	last := c.lastUpdate.Load()
	return time.Since(time.Unix(last, 0)) >= c.expiryTime
}
