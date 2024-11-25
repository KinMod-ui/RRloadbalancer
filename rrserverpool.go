package main

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type ServerPool interface {
	GetBackends() []Backend
	GetNextValidPeer() Backend
	AddBackend(Backend)
	GetServerPoolSize() int
	Serve(http.ResponseWriter, *http.Request)
}

type roundRobinServerPool struct {
	backends []Backend
	mux      sync.RWMutex
	current  int
}

func (s *roundRobinServerPool) Rotate() Backend {
	s.mux.Lock()
	s.current = (s.current + 1) % s.GetServerPoolSize()
	s.mux.Unlock()
	return s.backends[s.current]
}

func (s *roundRobinServerPool) GetNextValidPeer() Backend {
	for i := 0; i < s.GetServerPoolSize(); i++ {
		nextPeer := s.Rotate()
		if nextPeer.IsAlive() {
			return nextPeer
		}
	}
	return nil
}

func (s *roundRobinServerPool) GetBackends() []Backend {
	return s.backends
}

func (s *roundRobinServerPool) AddBackend(b Backend) {
	s.mux.Lock()
	s.backends = append(s.backends, b)
	s.mux.Unlock()
}

func (s *roundRobinServerPool) GetServerPoolSize() int {
	return len(s.backends)
}

func (sp *roundRobinServerPool) Serve(w http.ResponseWriter, r *http.Request) {
	mylog.Println(r.RemoteAddr)
	peer := sp.GetNextValidPeer()
	if peer != nil {
		peer.Serve(w, r)
		return
	}

	http.Error(w, "Service Not Available", http.StatusServiceUnavailable)
}

func HealthCheck(ctx context.Context, s ServerPool) {
	aliveChannel := make(chan bool, 1)

	for _, b := range s.GetBackends() {
		b := b
		requestCtx, stop := context.WithTimeout(ctx, 10*time.Second)
		defer stop()
		status := "up"
		go isBackendAlive(requestCtx, aliveChannel, b.GetURL())

		select {
		case <-ctx.Done():
			mylog.Println("Gracefully shutting down health check")
			return
		case alive := <-aliveChannel:
			b.SetAlive(alive)
			if !alive {
				status = "down"
			}

			mylog.Println("Url status", b.GetURL(), status)
		}
	}
}

func NewServerPool() (ServerPool, error) {
	return &roundRobinServerPool{
		backends: make([]Backend, 0),
		current:  0,
	}, nil
}
