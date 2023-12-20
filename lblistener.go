package main

import (
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

type Listener struct {
	Srv               http.Server
	Port              string
	Protocol          string
	SSLCertificate    string
	SSLCertificateKey string
	Balancers         []*Balancer
	IsRunning         bool
}

// var LoadBalancerListenersPool map[string]*LoadBalancerListener

func (lbs *Listener) Start() (err error) {
	if lbs.IsRunning {
		err = errors.New("LoadBalancer server is already running")
		return
	}

	go func(lbs *Listener) {
		log.Info().
			Str("port", lbs.Port).Str("protocol", lbs.Protocol).
			Msg("Starting Load Balancer Server")
		lbs.IsRunning = true
		err := lbs.Srv.ListenAndServe()
		if err == http.ErrServerClosed {
			lbs.IsRunning = false
			log.Info().Str("port", lbs.Port).Str("protocol", lbs.Protocol).Msg("Load Balancer server stopped")
		} else if err != nil {
			lbs.IsRunning = false
			log.Info().Str("port", lbs.Port).Err(err).Str("protocol", lbs.Protocol).Msg("Load Balancer server failed to start.")
		}
	}(lbs)

	return nil
}

func (lbs *Listener) Shutdown(serversSync *sync.WaitGroup) {
	log.Debug().
		Str("port", lbs.Port).
		Str("protocol", lbs.Protocol).
		Msg("Stopping listener at :" + lbs.Port)

	// Count Balancers
	serversSync.Add(len(lbs.Balancers))

	for _, balancer := range lbs.Balancers {
		go func(serversSync *sync.WaitGroup, balancer *Balancer) {
			log.Debug().Str("id", balancer.Id).Msg("Closing Load Balancer")
			balancer.State = LB_STATE_CLOSING
			balancer.liveConnections.Wait()
			balancer.State = LB_STATE_CLOSED
			serversSync.Done()
			log.Debug().Str("id", balancer.Id).Msg("Load Balancer Closed")
		}(serversSync, balancer)
	}
	serversSync.Wait()

	lbs.IsRunning = false

	log.Info().
		Str("port", lbs.Port).
		Str("protocol", lbs.Protocol).
		Msg("Listener stopped")
}

func (lbs *Listener) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if !lbs.IsRunning {
		log.Info().Msg("Request rejected due to inactive listener")
		return
	}
	requestURL := req.URL.RequestURI()

	candidateBalancer := struct {
		balancer *Balancer
		weight   int
	}{}

	found := false
	for _, balancer := range lbs.Balancers {
		if strings.Index(requestURL, balancer.RoutePrefix) == 0 {
			found = true
			balancerMatchWeight := len(balancer.RoutePrefix)
			if candidateBalancer.weight < balancerMatchWeight {
				candidateBalancer.balancer = balancer
				candidateBalancer.weight = len(balancer.RoutePrefix)
			}
		}
	}
	if found {
		log.Debug().
			Str("lister", lbs.Protocol+":"+lbs.Port).
			Str("balancer", candidateBalancer.balancer.Id).
			Str("uri", req.RequestURI).
			Str("method", req.Method).
			Msg("Request received")
		candidateBalancer.balancer.serveProxy(rw, req)
	} else {
		log.Info().Msgf("Balancer could not redirect request received at '%v'", requestURL)
	}
}

type ListenerHandler struct {
	Listener *Listener
}

func (lh ListenerHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	lh.Listener.ServeHTTP(rw, req)
}

func (lbs *Listener) GetListenerHandler() ListenerHandler {
	handler := ListenerHandler{
		Listener: lbs,
	}
	return handler
}
