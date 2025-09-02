package cache

import (
	"Hyflip-Server/internal/api"
	"Hyflip-Server/internal/config"
	"Hyflip-Server/internal/flippers"
	"log"
	"sync/atomic"
	"time"
)

// subscriberList just a struct wrapper for our subscribers slice so we can compare them in CAS
type subscriberList struct {
	subscribers []chan flippers.BazaarFoundFlip
}

type BazaarCache struct {
	// flipsSnapshot slice of flips updated every new flip found and reset every update. CAS'd so we don't have to lock for performance reasons
	flipsSnapshot atomic.Value
	// subscribers slice of subscriber channels. CAS'd so we don't have to lock for performance reasons
	subscribers atomic.Value

	api        *api.HypixelApiClient
	expiryTime time.Duration
	lastUpdate atomic.Int64
	isUpdating atomic.Bool
}

// NewBazaarCache returns a new BazaarCache. Keep only one of these per program lifecycle. It also starts the update goroutine automatically
func NewBazaarCache(apiClient *api.HypixelApiClient, expiryTime time.Duration) *BazaarCache {
	bzCache := &BazaarCache{
		api:        apiClient,
		expiryTime: expiryTime,
	}

	// empty collections to prevent nil panics. yes, we do not handle nils like real alphas
	bzCache.flipsSnapshot.Store(make(map[int]flippers.BazaarFoundFlip))
	bzCache.subscribers.Store(&subscriberList{
		subscribers: make([]chan flippers.BazaarFoundFlip, 0),
	})

	go bzCache.startUpdateGoroutine()
	return bzCache
}

// Get returns a snapshot of the most recent bazaar flips. Resets every new update
func (c *BazaarCache) Get() map[int]flippers.BazaarFoundFlip {
	return c.flipsSnapshot.Load().(map[int]flippers.BazaarFoundFlip)
}

// Subscribe adds your channel to the subscribers list so you can receive live updates. compare-and-swap loop
func (c *BazaarCache) Subscribe() chan flippers.BazaarFoundFlip {
	newSubChan := make(chan flippers.BazaarFoundFlip, 100)
	for {
		// read our current slice
		oldListPtr := c.subscribers.Load().(*subscriberList)
		oldSlice := oldListPtr.subscribers

		// make a copy, modify the copy with our new channel/subscriber
		newSlice := make([]chan flippers.BazaarFoundFlip, len(oldSlice)+1)
		copy(newSlice, oldSlice)
		newSlice[len(oldSlice)] = newSubChan

		// CAS the newslice. retry in case some other goroutine also does this. using struct so we can CAS as you cannot compare slices.
		newListPtr := &subscriberList{subscribers: newSlice}
		if c.subscribers.CompareAndSwap(oldListPtr, newListPtr) {
			log.Println("New subscriber added. Total subscribers:", len(newSlice))
			return newSubChan
		}
	}
}

// Unsubscribe removes a subscriber's channel (if it exists).  compare-and-swap loop. for ref: we can directly look for the channels as they are reference types and point to distinct objects in memory
func (c *BazaarCache) Unsubscribe(subChan chan flippers.BazaarFoundFlip) {
	for {
		oldListPtr := c.subscribers.Load().(*subscriberList)
		oldSlice := oldListPtr.subscribers
		foundIndex := -1
		for i, ch := range oldSlice {
			if ch == subChan {
				foundIndex = i
				break
			}
		}

		// channel is not in the list
		if foundIndex == -1 {
			return
		}

		// create a new slice excluding the removed channel
		newSlice := make([]chan flippers.BazaarFoundFlip, 0, len(oldSlice)-1)
		newSlice = append(newSlice, oldSlice[:foundIndex]...)
		newSlice = append(newSlice, oldSlice[foundIndex+1:]...)

		// CAS the newslice. if swap fails, that means another CAS happened at the same time. so we retry (hopefully not forever). i should prob add a 'retries' mechanism lmao
		newListPtr := &subscriberList{subscribers: newSlice}
		if c.subscribers.CompareAndSwap(oldListPtr, newListPtr) {
			log.Println("Subscriber removed. Total subscribers:", len(newSlice))
			return
		}
	}
}

// startUpdateGoroutine is the background goroutine to keep updating our cache. "BUT ISNT THIS AGAINST THE PHILOSOPHY OF CACHE??" I DONT CARE
func (c *BazaarCache) startUpdateGoroutine() {
	ticker := time.NewTicker(c.expiryTime / 4)
	defer ticker.Stop()

	for range ticker.C {
		if c.isExpired() {
			if !c.isUpdating.CompareAndSwap(false, true) {
				log.Println("Skipping bazaar cache update: an update is already in progress.")
				continue
			}

			func() {
				log.Println("Starting bazaar cache update...")
				defer c.isUpdating.Store(false)
				c.lastUpdate.Store(time.Now().Unix())

				// get the current list of subscribers and reset it. atomic swap makes it efficient asf, better than locking
				newEmptyList := &subscriberList{subscribers: make([]chan flippers.BazaarFoundFlip, 0)}
				oldList := c.subscribers.Swap(newEmptyList).(*subscriberList)
				subscribersForThisUpdate := oldList.subscribers

				chn, err := flippers.BzFlip(c.api, config.GenerateDefaultBZConfig())
				if err != nil {
					log.Println("Error updating bazaar cache. Err: " + err.Error())
					// Ensure channels are closed even on failure.
					for _, subChan := range subscribersForThisUpdate {
						close(subChan)
					}
					return
				}

				newSnapshot := make(map[int]flippers.BazaarFoundFlip)
				counter := 0
				for foundFlip := range chn {
					// live broadcast the updates to our subscribers
					for _, subChan := range subscribersForThisUpdate {
						select {
						case subChan <- foundFlip:
						default:
							// subscriber channel buffer is full so we just drop it.
						}
					}

					newSnapshot[counter] = foundFlip
					counter++
				}

				c.flipsSnapshot.Store(newSnapshot) // we store the snapshot in case the next update is delayed (like the 1-4s delay of bzflip)
				log.Printf("Bazaar cache update complete. Stored %d new flips.", len(newSnapshot))

				// end of flips. close channels for subscribers
				for _, subChan := range subscribersForThisUpdate {
					close(subChan)
				}

				log.Printf("Closed %d subscriber channels for this update cycle.", len(subscribersForThisUpdate))
			}()
		}
	}
}

// isExpired checks if the cache is expired based on the configured expiry time.
func (c *BazaarCache) isExpired() bool {
	last := c.lastUpdate.Load()
	if last == 0 {
		return true // Never updated, so it's expired.
	}
	return time.Since(time.Unix(last, 0)) >= c.expiryTime
}
