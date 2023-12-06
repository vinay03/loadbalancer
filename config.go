package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type LoadBalancerYAMLConfiguration struct {
	Balancers map[string]struct {
		Type    string "yaml:type"
		Port    int32  "yaml:port"
		Servers []struct {
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
		log.Fatal(err)
	}

	// Unmarshal YAML config into LoadBalancerYAMLConfiguration object
	cnf = UnmarshalYAML(&f)

	return
}

func UnmarshalYAML(contents *[]byte) (cnf *LoadBalancerYAMLConfiguration) {
	// var rawConfig map[string]interface{}
	cnf = &LoadBalancerYAMLConfiguration{}
	if err := yaml.Unmarshal(*contents, cnf); err != nil {
		log.Fatal(err)
	}

	balancersConfigCount := len(cnf.Balancers)
	if balancersConfigCount > 0 {
		for balancerId, balancerCnf := range cnf.Balancers {
			// Check type field
			if balancerCnf.Type == "" {
				balancerCnf.Type = DefaultLoadBalancerType
				log.Printf("Type field is not set for load balancer '%v'. Hence setting it to default '%v'", balancerId, DefaultLoadBalancerType)
			}
			if !IsValidBalancerType(balancerCnf.Type) {
				log.Fatalf("Type field for load balancer '%v' is set to '%v', which is invalid. Supported types are : '%+v'", balancerId, balancerCnf.Type, strings.Join(supportedBalancers, "', '"))
			}

			// Check Port field
			if balancerCnf.Port == 0 {
				log.Fatalf("Port field is not set for load balancer '%v'", balancerId)
			}

			// Check servers field
			serversLen := len(balancerCnf.Servers)
			if serversLen < 1 {
				log.Fatalf("No redirection servers mentioned for load balancer '%v'", balancerId)
			}
		}
	}

	return
}
