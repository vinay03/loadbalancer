package main

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

// Configuration
const (
	TARGET_CONNECTION_TIMEOUT   = 30 * time.Second
	TARGET_CONNECTION_KEEPALIVE = 30 * time.Second
)

type Target struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func NewTarget(addr string) *Target {
	serverUrl, err := url.Parse(addr)
	if err != nil {
		log.Fatal().Msg("Error occured while parsing node url")
		os.Exit(1)
	}
	proxy := httputil.NewSingleHostReverseProxy(serverUrl)

	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   TARGET_CONNECTION_TIMEOUT,
			KeepAlive: TARGET_CONNECTION_KEEPALIVE,
		}).Dial,
		TLSHandshakeTimeout: 180 * time.Second,
	}

	return &Target{
		addr:  addr,
		proxy: proxy,
	}
}

func (s *Target) Address() string {
	return s.addr
}

func (s *Target) IsAlive() bool {
	return true
}
func (s *Target) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}
