package main

import (

	// "log"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var LbService LoadBalancerService

func main() {

	LbService = LoadBalancerService{}

	LbService.SetParams(loadFlags())

	fmt.Println(PrettyPrint(LbService.Config))

	LbService.Apply()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-done:
		LbService.Stop()
	}
}
