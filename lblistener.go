package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type LISTENER_STATE int

const (
	LISTENER_STATE_INIT    LISTENER_STATE = 0
	LISTENER_STATE_ACTIVE  LISTENER_STATE = 1
	LISTENER_STATE_CLOSING LISTENER_STATE = 2
	LISTENER_STATE_CLOSED  LISTENER_STATE = 3
)

type Listener struct {
	Srv               http.Server
	Port              string
	Protocol          string
	SSLCertificate    string
	SSLCertificateKey string
	Balancers         []*Balancer
	State             LISTENER_STATE
	ListenerWG        *sync.WaitGroup
	// IsRunning         bool
}

// var LoadBalancerListenersPool map[string]*LoadBalancerListener

func (lbs *Listener) Start() (err error) {
	if lbs.State != LISTENER_STATE_INIT {
		err = errors.New("LoadBalancer server is already running")
		return
	}

	go func(lbs *Listener) {
		log.Info().
			Str("port", lbs.Port).Str("protocol", lbs.Protocol).
			Msg("Starting Load Balancer Server")

		lbs.ListenerWG.Add(1)

		lbs.checkStateByPoll()

		err := lbs.Srv.ListenAndServe()
		if err == http.ErrServerClosed {
			lbs.State = LISTENER_STATE_CLOSED
			log.Info().Str("port", lbs.Port).Str("protocol", lbs.Protocol).Msg("Load Balancer server stopped")
		} else if err != nil {
			lbs.State = LISTENER_STATE_CLOSED
			log.Info().Str("port", lbs.Port).Err(err).Str("protocol", lbs.Protocol).Msg("Load Balancer server failed to start.")
		}
		lbs.ListenerWG.Done()
	}(lbs)

	return nil
}

func (lbs *Listener) checkStateByPoll() {
	loopBreaker := 1000
	go func(lbs *Listener) {
		for {
			if lbs.State != LISTENER_STATE_INIT {
				log.Error().
					Str("port", lbs.Port).
					Str("protocol", lbs.Protocol).
					Msgf("Exiting State checker due to unexpected listener state '%v'", lbs.GetState())
				break
			}
			requestURL := lbs.Protocol + "://localhost:" + lbs.Port + "/"
			res, err := http.Get(requestURL)
			if err != nil {
				log.Error().
					Str("port", lbs.Port).
					Str("protocol", lbs.Protocol).
					Msgf("Error making request to listener at '%v'", requestURL)
			}
			if res.StatusCode == 200 {
				log.Info().
					Str("port", lbs.Port).
					Str("protocol", lbs.Protocol).
					Msg("Listener is active")
				break
			}
			time.Sleep(2 * time.Millisecond)
			loopBreaker--
			if loopBreaker <= 0 {
				break
			}
		}
	}(lbs)
}

func (lbs *Listener) Shutdown(serversSync *sync.WaitGroup) {
	log.Debug().
		Str("port", lbs.Port).
		Str("protocol", lbs.Protocol).
		Msg("Stopping listener at :" + lbs.Port)

	// Count Balancers
	balancersSync := &sync.WaitGroup{}
	balancersSync.Add(len(lbs.Balancers))

	for _, balancer := range lbs.Balancers {
		go func(balancersSync *sync.WaitGroup, balancer *Balancer) {
			log.Debug().Str("balancer", balancer.Id).Msg("Closing Load Balancer")
			balancer.State = LB_STATE_CLOSING
			balancer.liveConnections.Wait()
			balancer.State = LB_STATE_CLOSED
			balancersSync.Done()
			log.Debug().Str("balancer", balancer.Id).Msg("- Load Balancer Closed")
		}(balancersSync, balancer)
	}
	balancersSync.Wait()

	lbs.State = LISTENER_STATE_CLOSED
	_ = lbs.Srv.Shutdown(context.Background())
	lbs.ListenerWG.Wait()

	serversSync.Done()
}

// Returns state in string format
func (lbs *Listener) GetState() string {
	states := map[LISTENER_STATE]string{
		0: "init",
		1: "active",
		2: "closing",
		3: "inactive",
	}
	return states[lbs.State]
}

// Handles are incoming requests for listener
func (lbs *Listener) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if lbs.State != LISTENER_STATE_ACTIVE {
		if lbs.State == LISTENER_STATE_INIT {
			lbs.State = LISTENER_STATE_ACTIVE
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte("Activated"))
			return
		} else {
			log.Info().Str("state", lbs.GetState()).Msg("Request rejected. Listener is not in active state")
			return
		}
	}
	requestURL := req.URL.RequestURI()

	candidateBalancer := struct {
		balancer *Balancer
		weight   int
	}{}

	found := false
	for _, balancer := range lbs.Balancers {
		if strings.Index(requestURL, balancer.RoutePrefix) == 0 && balancer.IsAvailable() {
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
			Str("uri", req.RequestURI).
			Str("method", req.Method).
			Msg("Request received")

		// Pass request to the chosen balancer
		candidateBalancer.balancer.serveProxy(rw, req)
	} else {
		log.Info().
			Str("route", requestURL).
			Msg("Request rejected. No matching balancer found.")
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
