package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/rs/zerolog/log"
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
	return &Target{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
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
