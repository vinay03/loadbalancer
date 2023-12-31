package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Sample(t *testing.T) {

	LbService = LoadBalancerService{}

	RoundRobinBalancerURL := "http://localhost:8080/"
	WeightedRoundRobinBalancerURL := "http://localhost:8081/"

	config := &LoadBalancerServiceParams{
		DebugMode:          true,
		YAMLConfigFilePath: "examples/02_Test_mixed_config.yaml",
	}

	LbService.SetParams(config)
	LbService.Apply()

	// Start Test Servers
	StartTestServers(3)

	// Check basic configuration
	t.Run("Check basic balancer config: Round Robin", func(t *testing.T) {
		TestData := []int{1, 2, 3, 1, 2, 3, 1}
		for _, expectedReplicaId := range TestData {
			body := new(TestServerDummyResponse)
			res := doHTTPGetRequest(RoundRobinBalancerURL, body)
			assert.Equal(t, 200, res.StatusCode, "Request failed")
			assert.Equal(t, expectedReplicaId, body.ReplicaId)
		}
	})

	t.Run("Check Basic Balancer config: Weighted Round Robin", func(t *testing.T) {
		TestData := []int{1, 1, 1, 2, 2, 3, 1, 1, 1, 2, 2, 3, 1, 1, 1}
		for _, expectedReplicaId := range TestData {
			body := new(TestServerDummyResponse)
			res := doHTTPGetRequest(WeightedRoundRobinBalancerURL, body)
			assert.Equal(t, 200, res.StatusCode, "Request failed")
			assert.Equal(t, expectedReplicaId, body.ReplicaId)
		}
	})

	// Check Headers
	t.Run("Check custom headers : static", func(t *testing.T) {
		body := new(TestServerDummyResponse)
		expectedValue := "custom-value"
		_ = doHTTPGetRequest(RoundRobinBalancerURL, body)
		actualHeaderValue := body.Headers["Custom-Header"]
		assert.Equal(t, expectedValue, actualHeaderValue, "Static custom headers feature is not working")
	})

	t.Run("Check custom headers : dynamic", func(t *testing.T) {
		TestData := map[string]string{
			"Forwarded-Protocol": "HTTP/1.1",
			"Forwarded-Host":     "localhost:8080",
			"Forwarded-By":       "round-robin-root",
			"Forwarded-tls":      "",
		}
		body := new(TestServerDummyResponse)
		_ = doHTTPGetRequest(RoundRobinBalancerURL, body)
		for headerKey, headerValue := range TestData {
			assert.Equal(t, headerValue, body.Headers[headerKey], fmt.Sprintf("Value for the header '%v' is not matching", headerKey))
		}
	})

	// Check route prefix
	// NOTE: For this test to work, "custom header : dyanmic headers" test must be passed.
	t.Run("Check route prefix feature", func(t *testing.T) {
		TestData := [][]string{
			{RoundRobinBalancerURL, "round-robin-root"},
			{RoundRobinBalancerURL, "round-robin-root"},
			{RoundRobinBalancerURL + "health", "round-robin-health"},
			{RoundRobinBalancerURL, "round-robin-root"},
			{RoundRobinBalancerURL + "test", "round-robin-root"},
			{WeightedRoundRobinBalancerURL, ""}, // Custom headers not specified in YAML
		}
		for _, testRecord := range TestData {
			body := new(TestServerDummyResponse)
			res := doHTTPGetRequest(testRecord[0], body)
			assert.Equal(t, 200, res.StatusCode, "Request failed")
			assert.Equal(t, testRecord[1], body.Headers["Forwarded-By"])
		}
	})

	LbService.Stop()
	StopTestServers()
}
