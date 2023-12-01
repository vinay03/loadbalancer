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
	// servers := []Server{
	// 	NewSimpleServer("http://localhost:8081"),
	// 	NewSimpleServer("http://localhost:8082"),
	// 	NewSimpleServer("http://localhost:8083"),
	// }
	lb := NewLoadBalancer("8000", nil)

	lb.AddNewServer(NewSimpleServer("http://localhost:8081"))
	lb.AddNewServer(NewSimpleServer("http://localhost:8082"))
	lb.AddNewServer(NewSimpleServer("http://localhost:8083"))

	// Start LoadBalancing
	lb.Start()
}
