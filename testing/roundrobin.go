package testing_test

import (
	"net/http"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/vinay03/loadbalancer/src"
)

var _ = Describe("Round Robin Logic", func() {
	var LbTestService LoadBalancerService
	BeforeEach(func() {
		LbTestService = LoadBalancerService{}

		config := &LoadBalancerServiceParams{
			// DebugMode: DebugMode,
			YAMLConfigString: `listeners:
  - protocol: http
    port: 8080
    routes:
      - routeprefix: "/"
        mode: "RoundRobin"
        targets:
          - address: http://localhost:8091
          - address: http://localhost:8092
          - address: http://localhost:8093
      - routeprefix: "/single"
        mode: "RoundRobin"
        targets:
          - address: http://localhost:8091`,
		}
		if DebugMode {
			config.DebugMode = DebugMode
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
		TestData := []int{
			1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3,
			1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3,
		}
		for _, expectedReplicaId := range TestData {
			res, body := Request(LISTENER_8080_URL).Get()
			// Check status code
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			// Check replica ID
			Expect(body.ReplicaId).To(Equal(expectedReplicaId))
		}
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

		res, body := Request(LISTENER_8080_URL).Get()
		Expect(res.StatusCode).To(Equal(http.StatusOK))
		Expect(body.ReplicaId).To(Equal(1))

		TestServersPool[1].Stop()

		res, _ = Request(LISTENER_8080_URL).Get()
		Expect(res.StatusCode).To(Equal(http.StatusBadGateway))

		TestData := []int{
			3, 1, 3, 1, 3, 1, 3, 1, 3,
		}
		for _, expectedReplicaId := range TestData {
			res, body := Request(LISTENER_8080_URL).Get()
			if res.StatusCode == 200 {
				// Check replica ID
				Expect(body.ReplicaId).To(Equal(expectedReplicaId))
			}
		}

		TestServersPool[0].Stop()

		res, _ = Request(LISTENER_8080_URL).Get()
		Expect(res.StatusCode).To(Equal(http.StatusBadGateway))

		TestData = []int{
			3, 3, 3, 3, 3,
		}
		for _, expectedReplicaId := range TestData {
			res, body := Request(LISTENER_8080_URL).Get()
			if res.StatusCode == 200 {
				// Check replica ID
				Expect(body.ReplicaId).To(Equal(expectedReplicaId))
			}
		}

		TestServersPool[2].Stop()

		res, _ = Request(LISTENER_8080_URL).Get()
		Expect(res.StatusCode).To(Equal(http.StatusBadGateway))

	})

	It("Load Tests", func() {
		// Start Recording History
		LbTestService.Listeners[0].Balancers[0].DebugMode = true

		endWG := &sync.WaitGroup{}

		repeatations := 50
		passSize := 3
		requestsCount := repeatations * passSize
		endWG.Add(requestsCount)

		for i := 0; i < repeatations*passSize; i++ {
			req := Request(LISTENER_8080_URL)
			go req.GetWG(endWG)
		}
		endWG.Wait()
		var history *[]int = &LbTestService.Listeners[0].Balancers[0].DebugIndicesHistory
		Expect(len(*history)).To(Equal(requestsCount))

		CompleteCheck := true
		for i := 0; i < repeatations*passSize; i += passSize {
			CompleteCheck = CompleteCheck && ((*history)[i] == 0) && ((*history)[i+1] == 1) && ((*history)[i+2] == 2)
		}
		Expect(CompleteCheck).To(BeTrue())

		// Stop recording history
		LbTestService.Listeners[0].Balancers[0].DebugMode = false
	})

})
