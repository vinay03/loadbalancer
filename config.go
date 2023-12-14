package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"gopkg.in/yaml.v3"
)

type LoadBalancerYAMLConfiguration struct {
	Balancers []struct {
		Name      string "yaml:name"
		Type      string "yaml:type"
		Port      int32  "yaml:port"
		ApiPrefix string "yaml:apiprefix"
		Servers   []struct {
			Address string "yaml:address"
			Weight  int32  "yaml:weight"
		} "yaml:servers"
	} "yaml:balancers"
}

var DefaultLoadBalancerType string = "RoundRobin"

var supportedBalancers []string = []string{
	"RoundRobin",
	"WeightedRoundRobin",
	"PerformanceBased",
}

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

func UnmarshalYAML(contents *[]byte) (cnf *LoadBalancerYAMLConfiguration) {
	// var rawConfig map[string]interface{}
	cnf = &LoadBalancerYAMLConfiguration{}
	if err := yaml.Unmarshal(*contents, cnf); err != nil {
		log.Error().Err(err)
	}

	PrettyPrint(cnf.Balancers)

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
			if balancerCnf.Port == 0 {
				log.Error().Str("balancer", balancerCnf.Name).Msg("Port field is not set")
			}

			// Check servers field
			serversLen := len(balancerCnf.Servers)
			if serversLen < 1 {
				log.Error().Str("balancer", balancerCnf.Name).Msg("No redirection servers mentioned")
			}
		}
	}

	return
}