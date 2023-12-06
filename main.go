package main

import (
	"fmt"
	"log"
	"os"
)

func handleErr(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	var yamlConfigFile string
	if len(os.Args) > 1 {
		yamlConfigFile = os.Args[1]
	}
	cnf, err := LoadConfigFromFile(yamlConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	// Start load balancers
	startLoadBalancers(cnf)
}
