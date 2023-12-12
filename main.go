package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func handleErr(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	var yamlConfigFilePath string
	if len(os.Args) > 1 {
		yamlConfigFilePath = os.Args[1]
	}
	cnf, err := LoadConfigFromFile(yamlConfigFilePath)
	if err != nil {
		log.Fatal(err)
	}

	// Start load balancers
	startLoadBalancers(cnf)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-done:
		log.Printf("Shutting down gracefully...")
		for _, balancerCnf := range LoadBalancersPool {
			balancerCnf.liveConnections.Wait()
			balancerCnf.srv.Shutdown(context.Background())
		}
	}
}
