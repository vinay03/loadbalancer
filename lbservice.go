package main

import (
	"flag"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type LoadBalancerService struct {
	Params               *LoadBalancerServiceParams
	Config               *LoadBalancerYAMLConfiguration
	Listeners            []*Listener
	BalancersIdReference map[string]*Balancer
	State                string
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
}

func (lbs *LoadBalancerService) Apply() {
	lbs.BalancersIdReference = make(map[string]*Balancer)
	for _, listenerCnf := range lbs.Config.Listeners {
		lbListener := Listener{
			Port:     listenerCnf.Port,
			Protocol: listenerCnf.Protocol,
			Srv: http.Server{
				Addr: ":" + listenerCnf.Port,
			},
		}

		for _, route := range listenerCnf.Routes {
			lbalancer := &Balancer{
				Id:                route.Id,
				Mode:              route.Mode,
				RoutePrefix:       route.Routeprefix,
				TargetWaitTimeout: time.Duration(route.TargetWaitTimeout) * time.Second,
				CustomHeaderRules: route.CustomHeaders,
			}
			lbalancer.SetBalancerLogic()
			for _, target := range route.Targets {
				lbalancer.AddNewServer(target.Address)
			}

			lbListener.Balancers = append(lbListener.Balancers, lbalancer)
			lbs.BalancersIdReference[route.Id] = lbalancer
		}
		lbs.Listeners = append(lbs.Listeners, &lbListener)
		lbListener.Srv.Handler = lbListener.GetListenerHandler()
	}

	// start all listeners
	for _, lblistener := range lbs.Listeners {
		lblistener.Start()
	}
}

func (lbs *LoadBalancerService) Stop() {
	log.Info().Msg("Triggered shutdown procedure for Load Balancer Service...")
	serversSync := &sync.WaitGroup{}
	serversSync.Add(len(lbs.Listeners))
	for _, listener := range lbs.Listeners {
		go func(serversSync *sync.WaitGroup, listener *Listener) {
			listener.Shutdown(serversSync)
		}(serversSync, listener)
	}
	serversSync.Wait()
	log.Info().Msg("Load Balancer service shutdown completed")
}
