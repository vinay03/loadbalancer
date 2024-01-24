package testing_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/vinay03/loadbalancer/src"
)

var _ = Describe("Round Robin Logic", func() {
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
			Expect(res.StatusCode).To(Equal(200))
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
			Expect(res.StatusCode).To(Equal(200))
			// Check replica ID
			Expect(body.ReplicaId).To(Equal(expectedReplicaId))
		}
	})

})