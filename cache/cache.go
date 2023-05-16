package cache

import (
	"errors"
	"sync"
	"time"
)

type cachedData struct {
	value             interface{}
	expireAtTimestamp int64
}

type localCache struct {
	stop chan struct{}
	wg   sync.WaitGroup
	mu   sync.RWMutex

	data map[string]cachedData
}

func NewLocalCache(cleanupInterval time.Duration) *localCache {
	lc := &localCache{
		data: make(map[string]cachedData),
		stop: make(chan struct{}),
	}

	lc.wg.Add(1)
	go func(cleanupInterval time.Duration) {
		defer lc.wg.Done()
		lc.cleanupLoop(cleanupInterval)
	}(cleanupInterval)

	return lc
}

func (lc *localCache) cleanupLoop(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-lc.stop:
			return
		case <-t.C:
			lc.mu.Lock()
			for uid, cache := range lc.data {
				if cache.expireAtTimestamp <= time.Now().Unix() {
					delete(lc.data, uid)
				}
			}
			lc.mu.Unlock()
		}
	}
}

func (lc *localCache) StopCleanup() {
	close(lc.stop)
	lc.wg.Wait()
}

func (lc *localCache) Set(key string, v interface{}, expireAtTimestamp int64) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.data[key] = cachedData{
		value:             v,
		expireAtTimestamp: expireAtTimestamp,
	}
}

var (
	errValNotInCache = errors.New("the value isn't in cache")
)

func (lc *localCache) Get(key string) (interface{}, error) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	cache, ok := lc.data[key]
	if !ok {
		return nil, errValNotInCache
	}

	return cache.value, nil
}

func (lc *localCache) Delete(key string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	delete(lc.data, key)
}
