package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"gopkg.in/yaml.v3"
)

type LoadBalancerYAMLConfiguration struct {
	Balancers []struct {
		Name      string "yaml:name"
		Type      string "yaml:type"
		Port      string "yaml:port"
		ApiPrefix string "yaml:apiprefix"
		Servers   []struct {
			Address string "yaml:address"
			Weight  int32  "yaml:weight"
		} "yaml:servers"
	} "yaml:balancers"
}

const (
	LB_TYPE_ROUNDROBIN          = "RoundRobin"
	LB_TYPE_WEIGHTED_ROUNDROBIN = "WeightedRoundRobin"
	LB_TYPE_PERFORMANCE_BASED   = "PerformanceBased"
)

var supportedBalancers []string = []string{
	LB_TYPE_ROUNDROBIN,
	LB_TYPE_WEIGHTED_ROUNDROBIN,
	LB_TYPE_PERFORMANCE_BASED,
}

var DefaultLoadBalancerType string = LB_TYPE_ROUNDROBIN
var DefaultApiPrefix string = "/"

func IsValidBalancerType(searchType string) (isValid bool) {
	for _, v := range supportedBalancers {
		if v == searchType {
			isValid = true
			return
		}
	}
	return
}

func LoadConfigFromFile(filepath string) (cnf *LoadBalancerYAMLConfiguration, err error) {
	filePathLength := len(filepath)
	if filePathLength == 0 {
		panic("Configuration file was not provided")
	}

	if _, err := os.Stat(filepath); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Sprintf("Configuration file not found at location: %v", filepath))
	}

	f, err := os.ReadFile(filepath)
	if err != nil {
		log.Error().Err(err)
	}

	// Unmarshal YAML config into LoadBalancerYAMLConfiguration object
	cnf = UnmarshalYAML(&f)

	return
}

func loadConfigFromString(fileContents string) (cnf *LoadBalancerYAMLConfiguration, err error) {
	fConts := []byte(fileContents)
	cnf = UnmarshalYAML(&fConts)
	return
}

func UnmarshalYAML(contents *[]byte) (cnf *LoadBalancerYAMLConfiguration) {
	// var rawConfig map[string]interface{}
	cnf = &LoadBalancerYAMLConfiguration{}
	if err := yaml.Unmarshal(*contents, cnf); err != nil {
		log.Error().Err(err)
	}

	balancersConfigCount := len(cnf.Balancers)
	if balancersConfigCount > 0 {
		for _, balancerCnf := range cnf.Balancers {
			// Check Name field
			if balancerCnf.Name == "" {
				log.Error().Msg("Name field is not set for load balancer.")
			}

			// Check type field
			if balancerCnf.Type == "" {
				balancerCnf.Type = DefaultLoadBalancerType
				log.Info().Str("balancer", balancerCnf.Name).Msgf("Type field defaults to '%v'", DefaultLoadBalancerType)
			}
			if !IsValidBalancerType(balancerCnf.Type) {
				log.Error().Str("balancer", balancerCnf.Name).Msgf("Type field is set to '%v', which is invalid. Supported types are : '%+v'", balancerCnf.Type, strings.Join(supportedBalancers, "', '"))
			}

			// Check Port field
			if balancerCnf.Port == "" {
				log.Error().Str("balancer", balancerCnf.Name).Msg("Port field is not set")
			}

			// Check servers field
			serversLen := len(balancerCnf.Servers)
			if serversLen < 1 {
				log.Error().Str("balancer", balancerCnf.Name).Msg("No redirection servers mentioned")
			}

			// Check apiprefix field
			apiprefixLen := len(balancerCnf.ApiPrefix)
			if apiprefixLen < 1 {
				log.Info().Str("balancer", balancerCnf.Name).Msg("`apiprefix` field not specified. Set to '/' by default.")
				balancerCnf.ApiPrefix = DefaultApiPrefix
			}

		}
	}

	return
}

func (cnf *LoadBalancerYAMLConfiguration) Initialize() {
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
