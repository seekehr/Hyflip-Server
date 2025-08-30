package cache_templates

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/cache"
	"Hyflip-Server/internal/config"
	"Hyflip-Server/internal/flippers"
	"log"
	"time"
)

func NewBazaarCache(api *api.HypixelApiClient) (*cache.Cache[<-chan flippers.BazaarFoundFlip], error) {
	defaultConfig := config.GenerateDefaultBZConfig() // NOTE: look at bz_flipper.go. We use a default config here because user config is applied TO the flips returned from cache
	updateFunc := func() (<-chan flippers.BazaarFoundFlip, error) {
		return flippers.BzFlip(api, defaultConfig)
	}
	errorFunc := func(err error) {
		log.Println("ERROR UPDATING BAZAAR CACHE. Error: " + err.Error())
	}

	newCache, err := cache.New[<-chan flippers.BazaarFoundFlip](20*time.Second, updateFunc, errorFunc)
	if err != nil {
		log.Fatal("Error creating new cache. " + err.Error())
		return nil, err
	}
	return newCache, nil
}
