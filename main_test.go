package main

import (
	"testing"

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
		body := new(TestServerDummyResponse)

		res := doHTTPGetRequest("http://localhost:8080/", body)
		assert.Equal(t, 200, res.StatusCode, "Request failed")
		assert.Equal(t, 1, body.ReplicaId)

		res2 := doHTTPGetRequest("http://localhost:8080/", body)
		assert.Equal(t, 200, res2.StatusCode, "Request failed")
		assert.Equal(t, 2, body.ReplicaId)

		res3 := doHTTPGetRequest("http://localhost:8080/", body)
		assert.Equal(t, 200, res3.StatusCode, "Request failed")
		assert.Equal(t, 3, body.ReplicaId)
	})
	LbService.Stop()
}
