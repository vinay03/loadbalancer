package testing_test

import (
	"net/http"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/vinay03/loadbalancer/src"
)

var _ = Describe("Least Connections - Round Robin Logic", func() {
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
        mode: "LeastConnectionsRoundRobin"
        targets:
          - address: http://localhost:8091
          - address: http://localhost:8092
          - address: http://localhost:8093
      - routeprefix: "/single"
        mode: "LeastConnectionsRoundRobin"
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

		longRequestReplicaNumber := -1
		firstReplicaId := 1

		requestsEndSync.Add(1)
		requestsStartSync.Add(1)

		go func(requestsEndSync *sync.WaitGroup) {
			longRequestReplicaNumber = firstReplicaId
			requestsStartSync.Done()
			payload := GetDelayedRequestPayload(2)
			res, body := Request(LISTENER_8080_URL + "delayed").Post(payload)
			// Check status code
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			Expect(body.ReplicaId).To(Equal(firstReplicaId))
			requestsEndSync.Done()
		}(requestsEndSync)

		requestsStartSync.Wait()
		time.Sleep(100 * time.Millisecond)

		TestData := []int{
			1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3,
		}
		for _, expectedReplicaId := range TestData {
			if expectedReplicaId == longRequestReplicaNumber {
				continue
			}
			res, body := Request(LISTENER_8080_URL + "delayed").Post(GetDelayedRequestPayload(0))
			// Check status code
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			Expect(body.ReplicaId).To(Equal(expectedReplicaId))
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
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			// Check replica ID
			Expect(body.ReplicaId).To(Equal(expectedReplicaId))
		}
	})

	It("With Multiple targets of mixed 'IsAlive' status ", func() {
		delayedRequestEndWG := &sync.WaitGroup{}
		delayedRequestStartWG := &sync.WaitGroup{}

		delayedRequestEndWG.Add(1)
		delayedRequestStartWG.Add(1)

		go func() {
			delayedRequestStartWG.Done()
			payload := GetDelayedRequestPayload(2) // Delayed for 2 second
			res, _ := Request(LISTENER_8080_URL + "delayed").Post(payload)
			// Check status code
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			delayedRequestEndWG.Done()
		}()

		delayedRequestStartWG.Wait()
		time.Sleep(100 * time.Millisecond)

		TestData := []int{
			2, 3, 2, 3, 2, 3, 2,
		}
		for _, expectedReplicaId := range TestData {
			res, body := Request(LISTENER_8080_URL).Get()
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			Expect(body.ReplicaId).To(Equal(expectedReplicaId))
		}

		TestServersPool[2].Stop()

		res, _ := Request(LISTENER_8080_URL).Get()
		Expect(res.StatusCode).To(Equal(http.StatusBadGateway))

		TestData = []int{
			2, 2, 2, 2, 2,
		}
		for _, expectedReplicaId := range TestData {
			res, body := Request(LISTENER_8080_URL).Get()
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			// Check replica ID
			Expect(body.ReplicaId).To(Equal(expectedReplicaId))
		}

		TestServersPool[1].Stop()

		res, _ = Request(LISTENER_8080_URL).Get()
		Expect(res.StatusCode).To(Equal(http.StatusBadGateway))

		TestData = []int{
			1, 1, 1, 1, 1,
		}
		for _, expectedReplicaId := range TestData {
			res, body := Request(LISTENER_8080_URL).Get()
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			// Check replica ID
			Expect(body.ReplicaId).To(Equal(expectedReplicaId))
		}

		TestServersPool[0].Stop()

		res, _ = Request(LISTENER_8080_URL).Get()
		Expect(res.StatusCode).To(Equal(http.StatusBadGateway))
	})

})
