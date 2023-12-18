package main

import (

	// "log"
	"os"
	"os/signal"
	"syscall"
)

var LbService LoadBalancerService

func main() {

	LbService = LoadBalancerService{}

	LbService.SetParams(loadFlags())

	// Start load balancers
	LbService.Start()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-done:
		LbService.Stop()
	}
}
