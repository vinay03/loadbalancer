package main

import (
	"flag"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type LoadBalancerService struct {
	Params                 *LoadBalancerServiceParams
	Config                 *LoadBalancerYAMLConfiguration
	Servers                map[string]*LoadBalancerServer
	BalancersNameReference map[string]*LoadBalancer
	State                  string
}

type LoadBalancerServiceParams struct {
	DebugMode          bool
	YAMLConfigFilePath string
	YAMLConfigString   string
}

func loadFlags() *LoadBalancerServiceParams {
	params := &LoadBalancerServiceParams{}

	debug := flag.Bool("debug", false, "Sets log level to debug")
	configFile := flag.String("config", "", "Path to YAML config file.")

	flag.Parse()

	// Load Debug flag
	params.DebugMode = *debug

	// Load config file path
	params.YAMLConfigFilePath = *configFile

	return params
}

func (lbs *LoadBalancerService) SetParams(config *LoadBalancerServiceParams) {
	lbs.Params = config

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	if lbs.Params.DebugMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	})

	if len(lbs.Params.YAMLConfigFilePath) > 0 {
		var err error
		lbs.Config, err = LoadConfigFromFile(lbs.Params.YAMLConfigFilePath)
		if err != nil {
			log.Error().Err(err)
		}
	}

	if len(lbs.Params.YAMLConfigString) > 0 {
		var err error
		lbs.Config, err = loadConfigFromString(lbs.Params.YAMLConfigString)
		if err != nil {
			log.Error().Err(err)
		}
	}

	if lbs.Params.DebugMode {
		log.Info().Msg("Configuration loaded: " + PrettyPrint(lbs.Config))
	}
}
func (lbs *LoadBalancerService) Start() {
	lbs.Config.Initialize()
}
func (lbs *LoadBalancerService) Stop() {
	log.Info().Msg("Stopping Load Balancers Service...")
	serversSync := &sync.WaitGroup{}
	for _, lbServer := range LoadBalancerServersPool {
		lbServer.Shutdown(serversSync)
	}
}
