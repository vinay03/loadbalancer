package main

import (
	"context"
	"fmt"
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

type LoadBalancer struct {
	srv             http.Server
	liveConnections sync.WaitGroup
	name            string
	port            string
	roundRobinCount int
	servers         []Server
}

var LoadBalancersPool map[string]*LoadBalancer
var LoadBalancerServersPool map[string]*LoadBalancerServer

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
func (lb *LoadBalancer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	lb.serveProxy(rw, req)
}

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
func (lbs *LoadBalancerServer) Start() (err error) {
	// if lbs.IsRunning {
	// 	err = errors.New("LoadBalancer server is already running")
	// 	return
	// }

	// srv := http.Server{}
	// lb.srv.Addr = ":" + lb.port
	// lb.srv.Handler = lb

	lbs.srv.Handler = lbs.router

	go func(lbs *LoadBalancerServer) {
		log.Info().Str("port", lbs.port).Msg("Starting Load Balancer Server")
		// lbs.IsRunning = true
		err := lbs.srv.ListenAndServe()
		if err == http.ErrServerClosed {
			// lb.IsRunning = false
			log.Info().Str("port", lbs.port).Msg("Load Balancer server stopped")
		} else if err != nil {
			// lb.IsRunning = false
			log.Info().Str("port", lbs.port).Err(err).Msg("Load Balancer server failed to start.")
		}
	}(lbs)

	return nil
}

func startLoadBalancers(cnf *LoadBalancerYAMLConfiguration) {
	LoadBalancersPool = make(map[string]*LoadBalancer)
	LoadBalancerServersPool = make(map[string]*LoadBalancerServer)

	for _, balancerCnf := range cnf.Balancers {
		var lbsrv *LoadBalancerServer
		var ok bool
		lbsrv, ok = LoadBalancerServersPool[balancerCnf.Port]
		if !ok {
			lbsrv = &LoadBalancerServer{
				port:   balancerCnf.Port,
				router: mux.NewRouter(),
				srv: http.Server{
					Addr: ":" + balancerCnf.Port,
				},
			}
			LoadBalancerServersPool[balancerCnf.Port] = lbsrv
		}

		lbalancer := NewLoadBalancer(balancerCnf.Name, fmt.Sprint(balancerCnf.Port))

		for _, server := range balancerCnf.Servers {
			lbalancer.AddNewServer(NewSimpleServer(server.Address))
		}

		// _ = lbalancer.Start()

		lbsrv.router.HandleFunc(balancerCnf.ApiPrefix, lbalancer.serveProxy)

		LoadBalancersPool[balancerCnf.Name] = lbalancer
		lbsrv.balancers = append(lbsrv.balancers, lbalancer)
	}

	// start all servers
	for _, lbServer := range LoadBalancerServersPool {
		lbServer.Start()
	}
}
