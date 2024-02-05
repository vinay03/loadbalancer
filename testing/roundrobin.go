package testing_test

import (
	"fmt"
	"net/http"
	"sync"
	"time"

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

	FIt("Load Tests", func() {
		LbTestService.Listeners[0].Balancers[0].DebugMode = true

		endWG := &sync.WaitGroup{}

		repeatations := 6000
		endWG.Add(repeatations * 3)

		for i := 0; i < repeatations*3; i++ {
			req := Request(LISTENER_8080_URL)
			go req.GetWG(endWG)
		}
		// time.Sleep(2 * time.Second)
		endWG.Wait()
		var history *[]int = &LbTestService.Listeners[0].Balancers[0].DebugIndicesHistory
		fmt.Println("Length: ", len(*history))
		// fmt.Println(LbTestService.Listeners[0].Balancers[0].DebugIndicesHistory)
		for i := 0; i < repeatations*3; i += 3 {
			roundCheck := (LbTestService.Listeners[0].Balancers[0].DebugIndicesHistory[i] == 0) &&
				(LbTestService.Listeners[0].Balancers[0].DebugIndicesHistory[i+1] == 1) &&
				(LbTestService.Listeners[0].Balancers[0].DebugIndicesHistory[i+2] == 2)
			Expect(roundCheck).To(BeTrue())
		}
		time.Sleep(800)

		LbTestService.Listeners[0].Balancers[0].DebugMode = false
	})

})
