package main

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type LoadBalancerServer struct {
	srv       http.Server
	port      string
	router    *mux.Router
	balancers []*LoadBalancer
	IsRunning bool
}

var LoadBalancerServersPool map[string]*LoadBalancerServer

func (lbs *LoadBalancerServer) Start() (err error) {
	if lbs.IsRunning {
		err = errors.New("LoadBalancer server is already running")
		return
	}

	lbs.srv.Handler = lbs.router

	go func(lbs *LoadBalancerServer) {
		log.Info().Str("port", lbs.port).Msg("Starting Load Balancer Server")
		lbs.IsRunning = true
		err := lbs.srv.ListenAndServe()
		if err == http.ErrServerClosed {
			lbs.IsRunning = false
			log.Info().Str("port", lbs.port).Msg("Load Balancer server stopped")
		} else if err != nil {
			lbs.IsRunning = false
			log.Info().Str("port", lbs.port).Err(err).Msg("Load Balancer server failed to start.")
		}
	}(lbs)

	return nil
}

func (lbs *LoadBalancerServer) Shutdown(serversSync *sync.WaitGroup) {
	log.Info().Str("port", lbs.port).Msg("Stopping server listening at :" + lbs.port)
	serversSync.Add(len(lbs.balancers))
	for _, balancerCnf := range lbs.balancers {
		go func(serversSync *sync.WaitGroup, balancerCnf *LoadBalancer) {
			balancerCnf.liveConnections.Wait()
			balancerCnf.srv.Shutdown(context.Background())
			serversSync.Done()
		}(serversSync, balancerCnf)
	}
}
