package main

import (
	"fmt"
	"net/http"
	"os"
	"testing"
)

func Test_Sample(t *testing.T) {

	LbService = LoadBalancerService{}

	config := &LoadBalancerServiceParams{
		DebugMode: true,
		YAMLConfigString: `balancers:
  - name: simple
    type: RoundRobin
    port: 8080
    apiprefix: "/"
    servers:
      - address: http://localhost:8081
      - address: http://localhost:8082
      - address: http://localhost:8083`,
	}

	LbService.SetParams(config)
	LbService.Start()

	serverPort := "8080"

	requestURL := fmt.Sprintf("http://localhost:%v/test", serverPort)
	res, err := http.Get(requestURL)
	if err != nil {
		fmt.Printf("error making http request: %s\n", err)
		os.Exit(1)
	}
	if res.StatusCode != 200 {
		t.Errorf("Request failed with status code %v", res.StatusCode)
	}

	LbService.Stop()
}
