package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Sample(t *testing.T) {

	LbService = LoadBalancerService{}

	config := &LoadBalancerServiceParams{
		DebugMode:        true,
		YAMLConfigString: generateBasicYAML("RoundRobin", "/"),
	}

	LbService.SetParams(config)
	LbService.Apply()

	t.Run("Check basic balancer config", func(t *testing.T) {
		res := doHTTPGetRequest("http://localhost:8080/")
		assert.Equal(t, 201, res.StatusCode, "Request failed")
	})

	LbService.Stop()
	time.Sleep(3 * time.Second)
}
