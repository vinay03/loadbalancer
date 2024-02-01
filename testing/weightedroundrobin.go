package testing_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/vinay03/loadbalancer/src"
)

var _ = Describe("Weighted Round Robin Logic", func() {
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
        mode: "WeightedRoundRobin"
        targets:
          - address: http://localhost:8091
            weight: 3
          - address: http://localhost:8092
            weight: 2
          - address: http://localhost:8093
            weight: 1
      - routeprefix: "/single"
        mode: "WeightedRoundRobin"
        targets:
          - address: http://localhost:8091
            weight: 2`,
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
			1, 1, 1, 2, 2, 3,
			1, 1, 1, 2, 2, 3,
			1, 1, 1, 2, 2, 3,
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

		TestServersPool[0].Stop()

		res, _ = Request(LISTENER_8080_URL).Get()
		Expect(res.StatusCode).To(Equal(http.StatusBadGateway))

		TestData := []int{
			2, 2, 3, 2, 2, 3, 2, 2, 3, 2, 2,
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

		TestData = []int{
			2, 2, 2, 2, 2, 2, 2,
		}
		for _, expectedReplicaId := range TestData {
			res, body := Request(LISTENER_8080_URL).Get()
			if res.StatusCode == 200 {
				// Check replica ID
				Expect(body.ReplicaId).To(Equal(expectedReplicaId))
			}
		}

		TestServersPool[1].Stop()

		res, _ = Request(LISTENER_8080_URL).Get()
		Expect(res.StatusCode).To(Equal(http.StatusBadGateway))

	})

})
