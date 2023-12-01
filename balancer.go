package main

import (
	"errors"
	"fmt"
	"net/http"
)

type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
	IsRunning       bool
}

func NewLoadBalancer(port string) *LoadBalancer {
	return &LoadBalancer{
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

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("forwarding request to address %q\n", targetServer.Address())
	targetServer.Serve(rw, req)
}

func (lb *LoadBalancer) AddNewServer(server Server) {
	lb.servers = append(lb.servers, server)
}

func (lb *LoadBalancer) Start() (err error) {
	if lb.IsRunning {
		err = errors.New("LoadBalancer is already running")
		return
	}
	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}
	http.HandleFunc("/", handleRedirect)
	fmt.Printf("serving requests at 'localhost:%s'\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
	lb.IsRunning = true
	return nil
}
