package main

import (
	"flag"
	"fmt"
	"time"

	// "log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var DebugMode bool
var YAMLConfigFilePath string

func handleErr(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func loadFlags() {
	debug := flag.Bool("debug", false, "Sets log level to debug")
	configFile := flag.String("config", "", "Path to YAML config file.")

	flag.Parse()

	// Load Debug flag
	DebugMode = *debug

	// Load config file path
	YAMLConfigFilePath = *configFile
}

func main() {
	// Process CLI flags
	loadFlags()

	// Initialize Logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	if DebugMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// Process YAML config file
	cnf, err := LoadConfigFromFile(YAMLConfigFilePath)
	if err != nil {
		log.Error().Err(err)
	}

	// Start load balancers
	startLoadBalancers(cnf)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-done:
		log.Info().Msg("Shutting down gracefully...")
		serversSync := &sync.WaitGroup{}
		for _, lbServer := range LoadBalancerServersPool {
			lbServer.Shutdown(serversSync)
		}
	}
}
