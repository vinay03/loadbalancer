package main

import (
	"net/http"
	"sync"

	"github.com/rs/zerolog/log"
)

type LoadBalancer struct {
	srv             http.Server
	liveConnections sync.WaitGroup
	name            string
	port            string
	roundRobinCount int
	servers         []Server
}

var LoadBalancersPool map[string]*LoadBalancer

func NewLoadBalancer(name, port string) *LoadBalancer {
	return &LoadBalancer{
		name:            name,
		port:            port,
		roundRobinCount: 0,
	}
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	for !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
}

// http.Handler is a interface that expects ServeHTTP() function to be implemented.
// func (lb *LoadBalancer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
// 	lb.serveProxy(rw, req)
// }

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	lb.liveConnections.Add(1)
	targetServer := lb.getNextAvailableServer()
	log.Debug().Str("balancer", lb.name).Str("to", targetServer.Address()).Str("api", req.RequestURI).Msg("Forwarding request")
	targetServer.Serve(rw, req)
	lb.liveConnections.Done()
}

func (lb *LoadBalancer) AddNewServer(server Server) {
	lb.servers = append(lb.servers, server)
}
