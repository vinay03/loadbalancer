package src

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

type Target struct {
	Address     string
	proxy       *httputil.ReverseProxy
	Weight      int
	Connections int64
	Alive       bool
}

type TargetYAMLConfig struct {
	Address string `yaml:"address"`
	Weight  int    `yaml:"weight"`
}

func NewTarget(targetConfig *TargetYAMLConfig) *Target {
	serverUrl, err := url.Parse(targetConfig.Address)
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

	target := &Target{
		Address: targetConfig.Address,
		Weight:  targetConfig.Weight,
		proxy:   proxy,
	}

	if targetConfig.Weight > 0 {
		target.Weight = targetConfig.Weight
	} else {
		target.Weight = DEFAULT_TARGET_WEIGHT
	}

	return target
}

func (s *Target) IsAlive() bool {
	// TODO: Check whether s.Address is reachable
	return s.Alive
}
func (s *Target) MarkAsReachable() {
	s.Alive = true
}
func (s *Target) MarkAsUnreachable() {
	s.Alive = false
}
func (s *Target) Serve(rw http.ResponseWriter, req *http.Request) {
	s.Connections++
	crw := &CustomResponseWriter{ResponseWriter: rw}

	s.proxy.ServeHTTP(crw, req)

	s.Connections--

	if crw.Status == http.StatusBadGateway || crw.Status == http.StatusServiceUnavailable {
		log.Info().Str("address", s.Address).Int("status", crw.Status).Msg("Target is unreachable.\n")
		s.MarkAsUnreachable()
	}
}

type CustomResponseWriter struct {
	http.ResponseWriter
	Status int
}

func (scrw *CustomResponseWriter) WriteHeader(code int) {
	scrw.Status = code
	scrw.ResponseWriter.WriteHeader(code)
}
