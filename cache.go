package main

import (
	"errors"
	"sync"
	"time"
)

var ErrNotFound error = errors.New("msg not found in cache")

type (
	msgCache struct {
		Response
		expireAtTimestamp int64
	}

	localCache struct {
		stop chan struct{}

		wg   sync.WaitGroup
		mu   sync.Mutex
		msgs map[string]msgCache
	}
)

func NewLocalCache(cleanupInterval time.Duration) *localCache {
	lc := localCache{
		msgs: make(map[string]msgCache),
		stop: make(chan struct{}),
	}

	lc.wg.Add(1)
	go func(interval time.Duration) {
		defer lc.wg.Done()
		lc.cleanupLoop(interval)
	}(cleanupInterval)

	return &lc
}

func (l *localCache) cleanupLoop(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-l.stop:
			return
		case <-t.C:
			l.mu.Lock()
			defer l.mu.Unlock()
			for host, msg := range l.msgs {
				if msg.expireAtTimestamp <= time.Now().Unix() {
					delete(l.msgs, host)
				}
			}
		}
	}
}

func (l *localCache) StopCleanup() {
	close(l.stop)
	l.wg.Wait()
}

func (l *localCache) Update(msg Response, expireTime int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.msgs[msg.Host] = msgCache{
		expireAtTimestamp: expireTime,
		Response:               msg,
	}
}

func (l *localCache) Read(host string) (Response, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	m, ok := l.msgs[host]
	if !ok {
		return Response{}, ErrNotFound
	}
	return m.Response, nil
}
