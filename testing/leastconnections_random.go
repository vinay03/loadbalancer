package testing_test

import (
	"net/http"
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
		delayedRequestEndSync := &sync.WaitGroup{}
		delayedRequestStartSync := &sync.WaitGroup{}

		totalTargets := 3
		longRequestReplicaNumber := -1

		delayedRequestEndSync.Add(1)
		delayedRequestStartSync.Add(1)

		go func(delayedRequestEndSync *sync.WaitGroup) {
			delayedRequestStartSync.Done()
			payload := GetDelayedRequestPayload(1)
			res, body := Request(LISTENER_8080_URL + "delayed").Post(payload)
			// Check status code
			Expect(res.StatusCode).To(Equal(200))

			longRequestReplicaNumber = body.ReplicaId
			delayedRequestEndSync.Done()
		}(delayedRequestEndSync)

		delayedRequestStartSync.Wait()
		time.Sleep(100 * time.Millisecond)

		for i := 0; i < 10; i++ {
			res, body := Request(LISTENER_8080_URL + "delayed").Post(GetDelayedRequestPayload(0))
			// Check status code
			Expect(res.StatusCode).To(Equal(200))
			replicaIdCheck := (body.ReplicaId >= 1 && body.ReplicaId <= totalTargets && body.ReplicaId != longRequestReplicaNumber)
			Expect(replicaIdCheck).To(BeTrue())
		}

		delayedRequestEndSync.Wait()
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

	It("With Multiple targets of mixed 'IsAlive' status ", func() {
		delayedRequestEndSync := &sync.WaitGroup{}
		delayedRequestStartSync := &sync.WaitGroup{}

		longRequestReplicaNumber := -1

		delayedRequestEndSync.Add(1)
		delayedRequestStartSync.Add(1)

		go func(delayedRequestEndSync *sync.WaitGroup) {
			delayedRequestStartSync.Done()
			payload := GetDelayedRequestPayload(2) // Delayed for 2 second
			res, body := Request(LISTENER_8080_URL + "delayed").Post(payload)
			// Check status code
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			longRequestReplicaNumber = body.ReplicaId
			delayedRequestEndSync.Done()
		}(delayedRequestEndSync)

		delayedRequestStartSync.Wait()
		time.Sleep(100 * time.Millisecond)

		uniqueReplicaIds := map[int]bool{}
		firstToCloseTargetIndex := -1
		secondToCloseTargetIndex := -1
		for i := 0; i < 10; i++ {
			res, body := Request(LISTENER_8080_URL).Get()
			Expect(res.StatusCode).To(Equal(http.StatusOK))

			uniqueReplicaIds[body.ReplicaId] = true
			replicaIndex := body.ReplicaId - 1
			if firstToCloseTargetIndex == -1 {
				firstToCloseTargetIndex = replicaIndex
			} else if secondToCloseTargetIndex == -1 && firstToCloseTargetIndex != replicaIndex {
				secondToCloseTargetIndex = replicaIndex
			}
		}
		Expect(len(uniqueReplicaIds)).To(Equal(2))

		TestServersPool[firstToCloseTargetIndex].Stop()

		for i := 0; i < 10; i++ {
			res, body := Request(LISTENER_8080_URL).Get()
			if res.StatusCode == http.StatusOK {
				Expect(body.ReplicaId).Should(BeElementOf([]int{
					secondToCloseTargetIndex + 1,
				}))
			}
		}
		TestServersPool[secondToCloseTargetIndex].Stop()

		for i := 0; i < 10; i++ {
			res, body := Request(LISTENER_8080_URL).Get()
			if res.StatusCode == http.StatusOK {
				// Check replica ID
				Expect(body.ReplicaId).ShouldNot(BeElementOf([]int{
					firstToCloseTargetIndex + 1,
					secondToCloseTargetIndex + 1,
				}))
			}
		}

		delayedRequestEndSync.Wait()
		TestServersPool[longRequestReplicaNumber-1].Stop()

		res, _ := Request(LISTENER_8080_URL).Get()
		Expect(res.StatusCode).To(Equal(http.StatusBadGateway))
	})

})
