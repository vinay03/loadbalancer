package testing_test

import (
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/vinay03/loadbalancer/src"
)

var _ = Describe("Least Connections - Random Logic", func() {
	var LbTestService LoadBalancerService

	BeforeEach(func() {
		LbTestService = LoadBalancerService{}

		config := &LoadBalancerServiceParams{
			DebugMode: DebugMode,
			YAMLConfigString: `listeners:
  - protocol: http
    port: 8080
    routes:
      - routeprefix: "/"
        mode: "LeastConnectionsRandom"
        targets:
          - address: http://localhost:8091
          - address: http://localhost:8092
          - address: http://localhost:8093
      - routeprefix: "/single"
        mode: "LeastConnectionsRandom"
        targets:
          - address: http://localhost:8091`,
		}

		LbTestService.SetParams(config)
		LbTestService.Apply()

		// Start Test Servers
		StartTestServers(3)
	})

	AfterEach(func() {
		LbTestService.Stop()
		StopTestServers()
	})

	It("Algorithm with multiple targets", func() {
		requestsEndSync := &sync.WaitGroup{}
		requestsStartSync := &sync.WaitGroup{}

		totalTargets := 3
		longRequestReplicaNumber := -1

		requestsEndSync.Add(1)
		requestsStartSync.Add(1)

		go func(requestsEndSync *sync.WaitGroup) {
			requestsStartSync.Done()
			payload := GetDelayedRequestPayload(1)
			res, body := Request(LISTENER_8080_URL + "delayed").Post(payload)
			// Check status code
			Expect(res.StatusCode).To(Equal(200))

			// res := doHTTPPostRequest(Lister1_URL+"delayed", `{ "delay": 1 }`, body)
			// assert.Equal(t, 200, res.StatusCode, "Request failed")
			longRequestReplicaNumber = body.ReplicaId
			requestsEndSync.Done()
		}(requestsEndSync)

		requestsStartSync.Wait()
		time.Sleep(100 * time.Millisecond)

		for i := 0; i < 10; i++ {
			res, body := Request(LISTENER_8080_URL + "delayed").Post(GetDelayedRequestPayload(0))
			// Check status code
			Expect(res.StatusCode).To(Equal(200))
			replicaIdCheck := (body.ReplicaId >= 1 && body.ReplicaId <= totalTargets && body.ReplicaId != longRequestReplicaNumber)
			Expect(replicaIdCheck).To(BeTrue())
		}

		requestsEndSync.Wait()
	})

	It("Algorithm with single target", func() {
		TestData := []int{
			1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
		}
		for _, expectedReplicaId := range TestData {
			res, body := Request(LISTENER_8080_URL + "single").Get()
			// Check status code
			Expect(res.StatusCode).To(Equal(200))
			// Check replica ID
			Expect(body.ReplicaId).To(Equal(expectedReplicaId))
		}
	})

})
