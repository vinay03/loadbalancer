package main

import (
	"fmt"
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
	_, err := LoadConfigFromFile(yamlConfigFile)
	if err != nil {
		fmt.Println(err)
	}

	// lb := NewLoadBalancer("8000")

	// lb.AddNewServer(NewSimpleServer("http://localhost:8081"))
	// lb.AddNewServer(NewSimpleServer("http://localhost:8082"))
	// lb.AddNewServer(NewSimpleServer("http://localhost:8083"))

	// // Start LoadBalancing
	// _ = lb.Start()
}
