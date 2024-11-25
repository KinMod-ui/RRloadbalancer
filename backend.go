package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type Backend interface {
	SetAlive(bool)
	IsAlive() bool
	GetURL() *url.URL
	GetActiveConnections() int
	Serve(http.ResponseWriter, *http.Request)
}

type backend struct {
	url          *url.URL
	alive        bool
	mux          sync.RWMutex
	connections  int
	reverseProxy *httputil.ReverseProxy
}

func (b *backend) GetActiveConnections() int {
	b.mux.RLock()
	conns := b.connections
	defer b.mux.RUnlock()
	return conns
}

func (b *backend) SetAlive(val bool) {
	b.mux.Lock()
	b.alive = val
	b.mux.Unlock()
}

func (b *backend) IsAlive() bool {
	b.mux.RLock()
	alive := b.alive
	defer b.mux.RUnlock()
	return alive
}

func (b *backend) GetURL() *url.URL {
	return b.url
}

func (b *backend) Serve(rw http.ResponseWriter, req *http.Request) {
	defer func() {
		b.mux.Lock()
		b.connections--
		b.mux.Unlock()
	}()

	mylog.Println("Serving http")

	b.mux.Lock()
	b.connections++
	b.mux.Unlock()
	b.reverseProxy.ServeHTTP(rw, req)
}

func NewBackend(u *url.URL, rp *httputil.ReverseProxy) Backend {
	return &backend{
		url:          u,
		alive:        true,
		reverseProxy: rp,
	}
}

func isBackendAlive(ctx context.Context, aliveChannel chan bool, u *url.URL) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", u.Host)
	if err != nil {
		mylog.Println("Site unreachable")
		aliveChannel <- false
		return
	}

	_ = conn.Close()
	aliveChannel <- true
}
