package main

import (
	"net/http"
	"sync"

	"github.com/rs/zerolog/log"
)

type LB_STATE int

const (
	LB_STATE_INIT    LB_STATE = 0
	LB_STATE_ACTIVE  LB_STATE = 1
	LB_STATE_CLOSING LB_STATE = 2
	LB_STATE_CLOSED  LB_STATE = 3
)

type Balancer struct {
	// srv             http.Server
	liveConnections sync.WaitGroup
	Id              string
	Mode            string
	RoutePrefix     string
	// port            string
	roundRobinCount int
	Targets         []*Target
	State           LB_STATE
}

var LoadBalancersPool map[string]*Balancer

func (lb *Balancer) getNextAvailableServer() *Target {
	target := lb.Targets[lb.roundRobinCount%len(lb.Targets)]
	for !target.IsAlive() {
		lb.roundRobinCount++
		target = lb.Targets[lb.roundRobinCount%len(lb.Targets)]
	}
	lb.roundRobinCount++
	return target
}

// http.Handler is a interface that expects ServeHTTP() function to be implemented.
// func (lb *Balancer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
// 	lb.serveProxy(rw, req)
// }

func (lb *Balancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	target := lb.getNextAvailableServer()
	log.Debug().
		Str("uri", req.RequestURI).
		Str("balancer", lb.Id).
		Str("to", target.Address()).
		Msg("Forwarding request")

	lb.liveConnections.Add(1)
	target.Serve(rw, req)
	lb.liveConnections.Done()
}

func (lb *Balancer) AddNewServer(addr string) {
	lb.Targets = append(lb.Targets, NewTarget(addr))
}
