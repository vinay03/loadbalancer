package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
)

type LoadBalancer struct {
	name            string
	port            string
	roundRobinCount int
	servers         []Server
	IsRunning       bool
}

var LoadBalancersPool map[string]*LoadBalancer

func NewLoadBalancer(name, port string) *LoadBalancer {
	return &LoadBalancer{
		name:            name,
		port:            port,
		roundRobinCount: 0,
	}
}

func (lb *LoadBalancer) Init(name, port string) {
	lb.name = name
	lb.port = port
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
	log.Printf("Load balancer '%v' started listening to '%s'\n", lb.name, lb.port)
	http.ListenAndServe(":"+lb.port, nil)
	lb.IsRunning = true
	return nil
}

func startLoadBalancers(cnf *LoadBalancerYAMLConfiguration) {
	LoadBalancersPool = make(map[string]*LoadBalancer)
	for Id, balancerCnf := range cnf.Balancers {

		LoadBalancersPool[Id] = NewLoadBalancer(Id, fmt.Sprint(balancerCnf.Port))

		for _, server := range balancerCnf.Servers {
			LoadBalancersPool[Id].AddNewServer(NewSimpleServer(server.Address))
		}

		_ = LoadBalancersPool[Id].Start()
	}
}
