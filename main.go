package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	// "log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func handleErr(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	debug := flag.Bool("debug", false, "Sets log level to debug")

	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	var yamlConfigFilePath string
	if len(os.Args) > 1 {
		yamlConfigFilePath = os.Args[1]
	}
	cnf, err := LoadConfigFromFile(yamlConfigFilePath)
	if err != nil {
		log.Error().Err(err)
	}

	log.Log()

	// Start load balancers
	startLoadBalancers(cnf)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-done:
		log.Info().Msg("Shutting down gracefully...")
		for _, balancerCnf := range LoadBalancersPool {
			balancerCnf.liveConnections.Wait()
			balancerCnf.srv.Shutdown(context.Background())
		}
	}
}
