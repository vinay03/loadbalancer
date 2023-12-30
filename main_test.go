package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Sample(t *testing.T) {

	LbService = LoadBalancerService{}

	yaml := `listeners:
  - protocol: http
    port: 8080
    ssl_certificate:
    ssl_certificate_key:
    routes:
      - routeprefix: "/"
        mode: "RoundRobin"
				id: "round-robin-root"
        targets: 
          - address: http://localhost:8091
          - address: http://localhost:8092
          - address: http://localhost:8093
  - protocol: http
    port: 8081
    ssl_certificate:
    ssl_certificate_key:
    routes:
      - routeprefix: "/"
        mode: "WeightedRoundRobin"
				id: "weighted-round-robin-root"
        targets: 
          - address: http://localhost:8091
          - address: http://localhost:8092
          - address: http://localhost:8093`
	RoundRobinBalancerURL := "http://localhost:8080/"
	WeightedRoundRobinBalancerURL := "http://localhost:8081/"

	config := &LoadBalancerServiceParams{
		DebugMode:        true,
		YAMLConfigString: yaml,
	}

	LbService.SetParams(config)
	LbService.Apply()

	t.Run("Check basic balancer config", func(t *testing.T) {

		RoundRobinTestData := []int{1, 2, 3, 1}
		for _, expectedReplicaId := range RoundRobinTestData {
			body := new(TestServerDummyResponse)
			res := doHTTPGetRequest(RoundRobinBalancerURL, body)
			assert.Equal(t, 200, res.StatusCode, "Request failed")
			assert.Equal(t, expectedReplicaId, body.ReplicaId)
		}
	})

	t.Run("Check Basic Balancer config: Weighted Round Robin", func(t *testing.T) {
		WeightedRoundRobinTestData := []int{1, 1, 1, 2, 2, 3, 1, 1, 1}
		for _, expectedReplicaId := range WeightedRoundRobinTestData {
			body := new(TestServerDummyResponse)
			res := doHTTPGetRequest(WeightedRoundRobinBalancerURL, body)
			assert.Equal(t, 200, res.StatusCode, "Request failed")
			assert.Equal(t, expectedReplicaId, body.ReplicaId)
		}
	})

	t.Run("Check route prefix field", func(t *testing.T) {

	})

	LbService.Stop()
}
