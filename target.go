package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/rs/zerolog/log"
)

type ProxyServer struct{}

func (ps *ProxyServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {

}

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func NewSimpleServer(addr string) *simpleServer {
	serverUrl, err := url.Parse(addr)
	if err != nil {
		log.Fatal().Msg("Error occured while parsing node url")
		os.Exit(1)
	}

	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

func (s *simpleServer) Address() string {
	return s.addr
}

func (s *simpleServer) IsAlive() bool {
	return true
}
func (s *simpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}
