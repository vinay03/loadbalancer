package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

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
			Routeprefix       string             `yaml:"routeprefix"`
			Id                string             `yaml:"id"`
			Mode              string             `yaml:"mode"`
			CustomHeaders     []CustomHeaderRule `yaml:"customHeaders"`
			TargetWaitTimeout int                `yaml:"targetWaitTimeout"`
			Targets           []TargetYAMLConfig `yaml:"targets"`
		} `yaml:"routes"`
	} `yaml:"listeners"`
}

// Default Config and constants
const (
	// Listener Protocol
	LS_PROTOCOL_HTTP  = "http"
	LS_PROTOCOL_HTTPS = "https"

	// Balancer Modes
	LB_MODE_RANDOM                   = "Random"
	LB_MODE_ROUNDROBIN               = "RoundRobin"
	LB_MODE_WEIGHTED_ROUNDROBIN      = "WeightedRoundRobin"
	LB_MODE_PERFORMANCE_BASED        = "PerformanceBased"
	LB_MODE_LEAST_CONNECTIONS_RANDOM = "LeastConnectionsRandom"

	AUTO_GENERATED_BALANCER_ID_LENGTH = 10

	DEFAULT_TARGET_WAIT_TIMEOUT time.Duration = 15 * time.Second

	TARGET_CONNECTION_TIMEOUT   = 300 * time.Second
	TARGET_CONNECTION_KEEPALIVE = 300 * time.Second

	DEFAULT_TARGET_WEIGHT = 1
)

var supportedListenerProtocols []string = []string{
	LS_PROTOCOL_HTTP,
	LS_PROTOCOL_HTTPS,
}

var supportedBalancers []string = []string{
	LB_MODE_RANDOM,
	LB_MODE_ROUNDROBIN, LB_MODE_WEIGHTED_ROUNDROBIN,
	LB_MODE_LEAST_CONNECTIONS_RANDOM,
}

const (
	DefaultLoadBalancerType string = LB_MODE_ROUNDROBIN
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

			for index, route := range listener.Routes {
				// Check Id field
				if len(route.Id) < 1 {
					route.Id = generateBalancerId(balancerIdsCheckPool)
					listener.Routes[index].Id = route.Id
					log.Info().Str("new-id", route.Id).Msg("Id field was not set hence auto-assigning a unique identifier")
				}
				balancerIdsCheckPool[route.Id] = true

				// Check Route Prefix field
				if len(route.Routeprefix) < 1 {
					log.Info().Str("balancer", route.Id).Msg("`routeprefix` field not specified. Set to '/' by default.")
					route.Routeprefix = DefaultRoutePrefix
				}

				// Check Mode field
				if route.Mode == "" {
					route.Mode = DefaultLoadBalancerType
					log.Info().Str("balancer", route.Id).Msgf("Mode field defaults to '%v'", DefaultLoadBalancerType)
				}
				if !IsValidBalancerMode(route.Mode) {
					log.Error().Str("balancer", route.Id).Msgf("Mode field is set to '%v', which is invalid. Supported types are : '%+v'", route.Mode, strings.Join(supportedBalancers, "', '"))
				}

				// Check targets field
				if len(route.Targets) < 1 {
					log.Error().Str("balancer", route.Id).Msg("No redirection targets mentioned")
				}
			}
		}
	} else {
		log.Info().Msg("No listeners were configured")
	}

	return
}
