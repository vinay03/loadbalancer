package testing_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/vinay03/loadbalancer/src"
)

var _ = Describe("Features :", func() {
	var LbTestService LoadBalancerService
	BeforeEach(func() {
		LbTestService = LoadBalancerService{}

		config := &LoadBalancerServiceParams{
			DebugMode: DebugMode,
			YAMLConfigString: `listeners:
  - protocol: http
    port: 8080
    ssl_certificate:
    ssl_certificate_key:
    routes:
      - routeprefix: "/"
        mode: "Random"
        id: "root-balancer"
        customHeaders:
          - method: "any"
            headers:
              - name: "Forwarded-Protocol"
                value: "[[protocol]]"
              - name: "Forwarded-Host"
                value: "[[client.host]]"
              - name: "Forwarded-tls"
                value: "[[tls.version]]"
              - name: "Custom-Header"
                value: "custom-value"
              - name: "Forwarded-By"
                value: "[[balancer.id]]"
          - method:
        targets:
          - address: http://localhost:8091
          - address: http://localhost:8092
          - address: http://localhost:8093
      - routeprefix: "/health"
        mode: "Random"
        id: "health-balancer"
        customHeaders:
          - method: "any"
            headers:
              - name: "Forwarded-By"
                value: "[[balancer.id]]"
        targets:
          - address: http://localhost:8091
          - address: http://localhost:8092
          - address: http://localhost:8093
      - routeprefix: "/random"
        mode: "Random"
        id: "random-balancer"
        customHeaders:
          - method: "any"
            headers:
              - name: "Forwarded-By"
                value: "[[balancer.id]]"
        targets:
          - address: http://localhost:8091
          - address: http://localhost:8092
          - address: http://localhost:8093
  - protocol: http
    port: 8081
    routes:
      - routeprefix: "/"
        mode: "Random"
        id: "root-balancer-2"
        targets: 
          - address: http://localhost:8091
          - address: http://localhost:8092
          - address: http://localhost:8093`,
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

	It("Custom Static Headers", func() {
		res, body := Request(LISTENER_8080_URL).Get()
		// Check status code
		Expect(res.StatusCode).To(Equal(http.StatusOK))
		// Check header value
		Expect(body.Headers["Custom-Header"]).To(Equal("custom-value"))
	})

	It("Dynamic Headers", func() {
		TestData := map[string]string{
			"Forwarded-Protocol": "HTTP/1.1",
			"Forwarded-Host":     "localhost:8080",
			"Forwarded-By":       "root-balancer",
			"Forwarded-tls":      "",
		}
		res, body := Request(LISTENER_8080_URL).Get()
		// Check status code
		Expect(res.StatusCode).To(Equal(http.StatusOK))
		for headerKey, headerValue := range TestData {
			Expect(body.Headers[headerKey]).To(Equal(headerValue))
		}
	})

	It("route prefix matching", func() {
		TestData := [][]string{
			{LISTENER_8080_URL, "root-balancer"},
			{LISTENER_8080_URL, "root-balancer"},
			{LISTENER_8080_URL + "health", "health-balancer"},
			{LISTENER_8080_URL, "root-balancer"},
			{LISTENER_8080_URL + "test", "root-balancer"},
			{LISTENER_8080_URL + "random", "random-balancer"},
			{LISTENER_8081_URL, ""}, // Custom headers not specified in YAML
		}
		for _, testRecord := range TestData {
			// body := new(TestServerDummyResponse)
			res, body := Request(testRecord[0]).Get()
			// Check status code
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			// Check "Forwarded-By" header
			Expect(body.Headers["Forwarded-By"]).To(Equal(testRecord[1]))
		}
	})

	// Test `targetWaitTimeout` feature
})
