package testing_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/vinay03/loadbalancer/src"

	"net/http"
	_ "net/http/pprof"
)

var _ = Describe("Random Logic", func() {
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
        mode: "Random"
        targets:
          - address: http://localhost:8091
          - address: http://localhost:8092
          - address: http://localhost:8093
      - routeprefix: "/single"
        mode: "Random"
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

	It("Algorithm with Multiple Targets", func() {
		for i := 0; i < 12; i++ {
			res, body := Request(LISTENER_8080_URL).Get()
			// Check status code
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			// Check replica ID
			Expect(body.ReplicaId).To(BeElementOf([]int{1, 2, 3}))
		}
	})

	It("Algorithm with Single Target", func() {
		for i := 0; i < 12; i++ {
			res, body := Request(LISTENER_8080_URL + "single").Get()
			// Check status code
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			// Check replica ID
			Expect(body.ReplicaId).To(Equal(1))
		}
	})

	It("With Multiple targets of mixed 'IsAlive' status ", func() {
		TestServersPool[1].Stop()
		for i := 0; i < 10; i++ {
			res, body := Request(LISTENER_8080_URL).Get()
			if res.StatusCode == http.StatusOK {
				// Check replica ID
				Expect(body.ReplicaId).To(BeElementOf([]int{1, 3}))
			}
		}

		TestServersPool[0].Stop()

		for i := 0; i < 10; i++ {
			res, body := Request(LISTENER_8080_URL).Get()
			if res.StatusCode == http.StatusOK {
				// Check replica ID
				Expect(body.ReplicaId).To(BeElementOf([]int{3}))
			}
		}

		TestServersPool[2].Stop()

		res, _ := Request(LISTENER_8080_URL).Get()
		Expect(res.StatusCode).To(Equal(http.StatusBadGateway))
	})
})
