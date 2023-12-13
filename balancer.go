package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type LoadBalancer struct {
	srv             http.Server
	liveConnections sync.WaitGroup
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

// http.Handler is a interface that expects ServeHTTP() function to be implemented.
func (lb *LoadBalancer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	lb.serveProxy(rw, req)
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	lb.liveConnections.Add(1)
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("Forwarding request to address %q\n", targetServer.Address())
	targetServer.Serve(rw, req)
	lb.liveConnections.Done()
}

func (lb *LoadBalancer) AddNewServer(server Server) {
	lb.servers = append(lb.servers, server)
}

func (lb *LoadBalancer) Start() (err error) {
	if lb.IsRunning {
		err = errors.New("LoadBalancer is already running")
		return
	}

	// srv := http.Server{}
	lb.srv.Addr = ":" + lb.port
	lb.srv.Handler = lb

	go func(lb *LoadBalancer) {
		log.Printf("Starting '%v' load balancer at '%v'\n", lb.name, lb.port)
		lb.IsRunning = true
		err := lb.srv.ListenAndServe()
		// log.Println(err)
		if err == http.ErrServerClosed {
			lb.IsRunning = false
			log.Printf("Load balancer '%v' is clsoed\n", lb.name)
		} else if err != nil {
			lb.IsRunning = false
			log.Printf("Load balancer '%v' failed to start at '%s'. %v\n", lb.name, lb.port, err)
		}
	}(lb)

	return nil
}

func startLoadBalancers(cnf *LoadBalancerYAMLConfiguration) {
	LoadBalancersPool = make(map[string]*LoadBalancer)
	for _, balancerCnf := range cnf.Balancers {
		LoadBalancersPool[balancerCnf.Name] = NewLoadBalancer(balancerCnf.Name, fmt.Sprint(balancerCnf.Port))

		for _, server := range balancerCnf.Servers {
			LoadBalancersPool[balancerCnf.Name].AddNewServer(NewSimpleServer(server.Address))
		}

		_ = LoadBalancersPool[balancerCnf.Name].Start()
	}
}
