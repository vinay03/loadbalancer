package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func Test_Sample(t *testing.T) {

	LbService = LoadBalancerService{}

	Lister1_URL := "http://localhost:8080/"
	Lister2_URL := "http://localhost:8081/"

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
			res := doHTTPGetRequest(Lister1_URL, body)
			assert.Equal(t, 200, res.StatusCode, "Request failed")
			assert.Equal(t, expectedReplicaId, body.ReplicaId)
		}
	})

	t.Run("Check Basic Balancer config: Weighted Round Robin", func(t *testing.T) {
		TestData := []int{1, 1, 1, 2, 2, 3, 1, 1, 1, 2, 2, 3, 1, 1, 1}
		for _, expectedReplicaId := range TestData {
			body := new(TestServerDummyResponse)
			res := doHTTPGetRequest(Lister2_URL, body)
			assert.Equal(t, 200, res.StatusCode, "Request failed")
			assert.Equal(t, expectedReplicaId, body.ReplicaId)
		}
	})

	t.Run("Check basic balancer config: Random Balancer", func(t *testing.T) {
		totalTargets := 3
		for i := 0; i < 12; i++ {
			body := new(TestServerDummyResponse)
			res := doHTTPGetRequest(Lister1_URL+"random", body)
			assert.Equal(t, 200, res.StatusCode, "Request failed")
			replicaIdCheck := (body.ReplicaId >= 1 && body.ReplicaId <= totalTargets)
			assert.Equal(t, true, replicaIdCheck, "Invalid balancing logic")
		}
	})

	t.Run("Check basic balancer config: Least Connections Random Logic", func(t *testing.T) {
		body := new(TestServerDummyResponse)
		requestsEndSync := &sync.WaitGroup{}
		requestsStartSync := &sync.WaitGroup{}

		totalTargets := 3
		longRequestReplicaNumber := -1

		requestsEndSync.Add(1)
		requestsStartSync.Add(1)
		go func(requestsEndSync *sync.WaitGroup) {
			requestsStartSync.Done()
			res := doHTTPPostRequest(Lister1_URL+"delayed", `{ "delay": 1 }`, body)
			assert.Equal(t, 200, res.StatusCode, "Request failed")
			longRequestReplicaNumber = body.ReplicaId
			requestsEndSync.Done()
		}(requestsEndSync)

		requestsStartSync.Wait()
		time.Sleep(100 * time.Millisecond)

		for i := 0; i < 10; i++ {
			body := new(TestServerDummyResponse)
			res := doHTTPPostRequest(Lister1_URL+"delayed", `{ "delay": 0 }`, body)
			assert.Equal(t, 200, res.StatusCode, "Request failed")
			replicaIdCheck := (body.ReplicaId >= 1 && body.ReplicaId <= totalTargets && body.ReplicaId != longRequestReplicaNumber)
			assert.Equal(t, true, replicaIdCheck, "Invalid balancing logic")
		}

		requestsEndSync.Wait()
	})

	t.Run("Check basic balancer config: Least Connections Round Robin Logic", func(t *testing.T) {
		body := new(TestServerDummyResponse)
		requestsEndSync := &sync.WaitGroup{}
		requestsStartSync := &sync.WaitGroup{}

		// totalTargets := 3

		requestsEndSync.Add(1)
		requestsStartSync.Add(1)

		go func(requestsEndSync *sync.WaitGroup) {
			requestsStartSync.Done()
			res := doHTTPPostRequest(Lister1_URL+"delayed-roundrobin", `{ "delay": 1 }`, body)
			assert.Equal(t, 200, res.StatusCode, "Request failed")
			assert.Equal(t, 1, body.ReplicaId, "Logic failed")
			requestsEndSync.Done()
		}(requestsEndSync)

		requestsStartSync.Wait()
		time.Sleep(100 * time.Millisecond)

		// checkReplicaId := 2
		checkData := []int{2, 3, 2, 3, 2, 3, 2, 3, 2, 3}
		for _, checkReplicaId := range checkData {
			body := new(TestServerDummyResponse)
			res := doHTTPPostRequest(Lister1_URL+"delayed-roundrobin", `{ "delay": 0 }`, body)
			assert.Equal(t, 200, res.StatusCode, "Request failed")
			assert.Equal(t, checkReplicaId, body.ReplicaId, "Invalid balancing logic")
			time.Sleep(20 * time.Millisecond)
		}
		log.Info().Msg("Waiting for long request to finish")
		requestsEndSync.Wait()

		// checkReplicaId = 1
		checkData = []int{1, 2, 3, 1, 2, 3, 1, 2, 3, 1}
		for _, checkReplicaId := range checkData {
			body := new(TestServerDummyResponse)
			res := doHTTPPostRequest(Lister1_URL+"delayed-roundrobin", `{ "delay": 0 }`, body)
			assert.Equal(t, 200, res.StatusCode, "Request failed")
			assert.Equal(t, checkReplicaId, body.ReplicaId, "Invalid balancing logic")
			time.Sleep(2000 * time.Microsecond)
		}
	})

	// Check Headers
	t.Run("Check custom headers : static", func(t *testing.T) {
		body := new(TestServerDummyResponse)
		expectedValue := "custom-value"
		_ = doHTTPGetRequest(Lister1_URL, body)
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
		_ = doHTTPGetRequest(Lister1_URL, body)
		for headerKey, headerValue := range TestData {
			assert.Equal(t, headerValue, body.Headers[headerKey], fmt.Sprintf("Value for the header '%v' is not matching", headerKey))
		}
	})

	// Check route prefix
	// NOTE: For this test to work, "custom header : dyanmic headers" test must be passed.
	t.Run("Check route prefix feature", func(t *testing.T) {
		TestData := [][]string{
			{Lister1_URL, "round-robin-root"},
			{Lister1_URL, "round-robin-root"},
			{Lister1_URL + "health", "round-robin-health"},
			{Lister1_URL, "round-robin-root"},
			{Lister1_URL + "test", "round-robin-root"},
			{Lister1_URL + "random", "random-logic"},
			{Lister2_URL, ""}, // Custom headers not specified in YAML
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
