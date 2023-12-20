package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"gopkg.in/yaml.v3"
)

type LoadBalancerYAMLConfiguration struct {
	Listeners []struct {
		Protocol          string `yaml:"protocol"`
		Port              string `yaml:"port"`
		SSLCertificate    string `yaml:"ssl_certificate"`
		SSLCertificateKey string `yaml:"ssl_certificate_key"`
		Routes            []struct {
			Routeprefix string `yaml:"routeprefix"`
			Id          string `yaml:"id"`
			Mode        string `yaml:"mode"`
			Targets     []struct {
				Address string `yaml:"address"`
				Weight  int32  `yaml:"weight"`
			} `yaml:"targets"`
		} `yaml:"routes"`
	} `yaml:"listeners"`
}

const (
	// Listener Protocol
	LS_PROTOCOL_HTTP  = "http"
	LS_PROTOCOL_HTTPS = "https"

	// Balancer Modes
	LB_TYPE_RANDOM              = "Random"
	LB_TYPE_ROUNDROBIN          = "RoundRobin"
	LB_TYPE_WEIGHTED_ROUNDROBIN = "WeightedRoundRobin"
	LB_TYPE_PERFORMANCE_BASED   = "PerformanceBased"

	AUTO_GENERATED_BALANCER_ID_LENGTH = 10
)

var supportedListenerProtocols []string = []string{
	LS_PROTOCOL_HTTP,
	LS_PROTOCOL_HTTPS,
}

var supportedBalancers []string = []string{
	LB_TYPE_ROUNDROBIN,
}

const (
	DefaultLoadBalancerType string = LB_TYPE_ROUNDROBIN
	DefaultRoutePrefix      string = "/"
	DefaultListenerPort     string = "80"
	DefaultListenerProtocol string = LS_PROTOCOL_HTTP
)

func IsValidListenerProtocol(protocol string) bool {
	for _, val := range supportedListenerProtocols {
		if val == protocol {
			return true
		}
	}
	return false
}
func IsValidBalancerMode(searchType string) bool {
	for _, v := range supportedBalancers {
		if v == searchType {
			return true
		}
	}
	return false
}

func LoadConfigFromFile(filepath string) (cnf *LoadBalancerYAMLConfiguration, err error) {
	if _, err := os.Stat(filepath); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Sprintf("Configuration file not found at location: %v", filepath))
	}

	f, err := os.ReadFile(filepath)
	if err != nil {
		log.Error().Err(err)
	}

	// Unmarshal YAML config
	cnf = UnmarshalYAML(&f)

	return
}

func loadConfigFromString(fileContents string) (cnf *LoadBalancerYAMLConfiguration, err error) {
	contents := []byte(fileContents)
	// Unmarshal YAML config
	cnf = UnmarshalYAML(&contents)
	return
}

func _getRandomString() string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, AUTO_GENERATED_BALANCER_ID_LENGTH)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func generateBalancerId(balancerIdsPool map[string]bool) (newId string) {
	maxTries := 5
	for i := 0; i < maxTries; i++ {
		newId = _getRandomString()
		_, ok := balancerIdsPool[newId]
		if ok {
			continue
		}
	}
	return
}

func UnmarshalYAML(contents *[]byte) (cnf *LoadBalancerYAMLConfiguration) {
	cnf = &LoadBalancerYAMLConfiguration{}
	if err := yaml.Unmarshal(*contents, cnf); err != nil {
		log.Error().Err(err)
	}

	portCheckPool := map[string]bool{}
	balancerIdsCheckPool := map[string]bool{}

	if len(cnf.Listeners) > 0 {
		for _, listener := range cnf.Listeners {

			// Check type field
			// if balancerCnf.Type == "" {
			// 	balancerCnf.Type = DefaultLoadBalancerType
			// 	log.Info().Str("balancer", balancerCnf.Name).Msgf("Type field defaults to '%v'", DefaultLoadBalancerType)
			// }
			// if !IsValidBalancerType(balancerCnf.Type) {
			// 	log.Error().Str("balancer", balancerCnf.Name).Msgf("Type field is set to '%v', which is invalid. Supported types are : '%+v'", balancerCnf.Type, strings.Join(supportedBalancers, "', '"))
			// }

			// check protocol field
			if listener.Protocol == "" {
				log.Info().Msgf("Protocol field not set, hence setting to default '%v'", DefaultListenerProtocol)
				listener.Protocol = DefaultListenerProtocol
			} else if !IsValidListenerProtocol(listener.Protocol) {
				log.Error().Msgf("Listener Protocol is not valid. Valid values are : '%+v", strings.Join(supportedListenerProtocols, "', '"))
			}

			// Check Port field
			if listener.Port == "" {
				log.Info().Msgf("Port not specified, hence setting to default port '%v'", DefaultListenerPort)
				listener.Port = DefaultListenerPort
			}
			portCheckPool[listener.Protocol+":"+listener.Port] = true

			// Check secure listener settings
			if listener.Protocol == LS_PROTOCOL_HTTPS {
				if listener.SSLCertificate == "" || listener.SSLCertificateKey == "" {
					log.Error().Msgf("SSL certificate fields are mandatory if protocol is set to '%v'", LS_PROTOCOL_HTTPS)
				}
			}

			for _, route := range listener.Routes {
				// Check Id field
				if len(route.Id) < 1 {
					route.Id = generateBalancerId(balancerIdsCheckPool)
					log.Info().Str("new-id", route.Id).Msg("Id field was not set hence auto-assigning a unique identifier")
				}
				balancerIdsCheckPool[route.Id] = true

				// Check Route Prefix field
				if len(route.Routeprefix) < 1 {
					log.Info().Str("id", route.Id).Msg("`routeprefix` field not specified. Set to '/' by default.")
					route.Routeprefix = DefaultRoutePrefix
				}

				// Check targets field
				if len(route.Targets) < 1 {
					log.Error().Str("id", route.Id).Msg("No redirection targets mentioned")
				}
			}
		}
	} else {
		log.Info().Msg("No listeners were configured")
	}

	return
}

// func (cnf *LoadBalancerYAMLConfiguration) Initialize() {
// 	LoadBalancersPool = make(map[string]*LoadBalancer)
// 	LoadBalancerServersPool = make(map[string]*LoadBalancerServer)

// 	for _, balancerCnf := range cnf.Balancers {
// 		var lbsrv *LoadBalancerServer
// 		var ok bool
// 		lbsrv, ok = LoadBalancerServersPool[balancerCnf.Port]
// 		if !ok {
// 			lbsrv = &LoadBalancerServer{
// 				port:   balancerCnf.Port,
// 				router: mux.NewRouter(),
// 				srv: http.Server{
// 					Addr: ":" + balancerCnf.Port,
// 				},
// 			}
// 			LoadBalancerServersPool[balancerCnf.Port] = lbsrv
// 		}

// 		lbalancer := NewLoadBalancer(balancerCnf.Name, fmt.Sprint(balancerCnf.Port))

// 		for _, server := range balancerCnf.Targets {
// 			lbalancer.AddNewServer(NewSimpleServer(server.Address))
// 		}

// 		// _ = lbalancer.Start()

// 		lbsrv.router.HandleFunc(balancerCnf.ApiPrefix, lbalancer.serveProxy)

// 		LoadBalancersPool[balancerCnf.Name] = lbalancer
// 		lbsrv.balancers = append(lbsrv.balancers, lbalancer)
// 	}

// 	// start all servers
// 	for _, lbServer := range LoadBalancerServersPool {
// 		lbServer.Start()
// 	}
// }
