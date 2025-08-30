package cache

import (
	"sync/atomic"
	"time"
)

// Cache is a thread-safe cache storage for storing data. Creating it using `New` automatically updates the data.
type Cache[T any] struct {
	data           atomic.Value      // thread-safe data
	expiryTime     time.Duration     // time between updates
	updateDataFunc func() (T, error) // function used to update the cache
	errorFunc      func(error)       // function called when error occurs during updating the cache
	lastUpdate     atomic.Int64      // thread-safe way to store last updates; used to track when to update again alongside expiry time
}

// New creates a new cache and automatically calls the first update. Returns an error if update function fails. Create ONLY one cache per data-set; cache updater goroutine cannot be cancelled.
func New[T any](expiry time.Duration, updateFunc func() (T, error), errorFunc func(error)) (*Cache[T], error) {

	cache := &Cache[T]{
		expiryTime:     expiry,
		updateDataFunc: updateFunc,
		errorFunc:      errorFunc,
	}

	// perform the initial data load immediately upon creation
	initialData, err := cache.updateDataFunc()
	if err != nil {
		return nil, err
	}

	cache.data.Store(initialData)
	cache.lastUpdate.Store(time.Now().Unix())

	// background updates goroutine. we won't ever need to close this as only one cache storage should be used per data-set and cache storage last the entirety of the program lifetime.
	go cache.manager()

	return cache, nil
}

// Get automatically returns the data type-casted
func (c *Cache[T]) Get() T {
	return c.data.Load().(T)
}

// manager updates the cache data when it expires.
func (c *Cache[T]) manager() {
	// checks every e.g 5s if the expiry time is 20s to see if the expiry time is reached. arbitrary number, but lower will mostly always be better ig
	ticker := time.NewTicker(c.expiryTime / 4)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if c.isExpired() {
				newData, err := c.updateDataFunc()
				c.lastUpdate.Store(time.Now().Unix()) // we want to update regardless of if the function was successful
				if err != nil {
					c.errorFunc(err)
					return
				}

				c.data.Store(newData)
			}
		}
	}
}

// isExpired checks if the cache is expired, used by cache manager to automatically update expired cache.
func (c *Cache[T]) isExpired() bool {
	last := c.lastUpdate.Load()
	return time.Since(time.Unix(last, 0)) >= c.expiryTime
}
