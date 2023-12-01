package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type LoadBalancerConfig struct {
	Balancers map[string]Balancer "yaml:balancers"
}

var supportedBalancers []string = []string{
	"RoundRobin",
	"WeightedRoundRobin",
	"PerformanceBased",
}

type Balancer struct {
	Type     string         "yaml:type"
	Port     string         "yaml:port"
	ApiMatch string         "yaml:apiMatch"
	Servers  []ServerConfig "yaml:servers"
}

type ServerConfig struct {
	Address string "yaml:address"
	Weight  int32  "yaml:weight"
}

func LoadConfigFromFile(filepath string) (cnf *LoadBalancerConfig, err error) {
	filePathLength := len(filepath)
	if filePathLength == 0 {
		panic("Configuration filepath was not provided")
	}

	if _, err := os.Stat(filepath); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Sprintf("Configuration file not found at location: %v", filepath))
	}

	f, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshal YAML config into LoadBalancerConfig object
	cnf = UnmarshalYAML(&f)

	// Validate configurations
	err = ValidateConfiguration(cnf)

	return
}

func UnmarshalYAML(contents *[]byte) (cnf *LoadBalancerConfig) {
	// var rawConfig map[string]interface{}
	cnf = &LoadBalancerConfig{}
	if err := yaml.Unmarshal(*contents, cnf); err != nil {
		log.Fatal(err)
	}
	return
}

func ValidateConfiguration(cnf *LoadBalancerConfig) (err error) {
	if len(cnf.Balancers) < 1 {
		return errors.New("No load balancers are configured")
	}

	for name, lb := range cnf.Balancers {
		fmt.Printf("Configuration for '%v' balancer is %+v\n", name, lb)
	}
	return
}
