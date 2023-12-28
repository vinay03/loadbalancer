package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type TestServerDummyResponse struct {
	Message   string `json:"message"`
	ReplicaId int    `json:"replicaId"`
}

func GetNumberedHandler(ReplicaNumber int) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(TestServerDummyResponse{
			Message:   fmt.Sprintf("Response to URI '%v' from Replica #%v", req.URL, ReplicaNumber),
			ReplicaId: ReplicaNumber,
		})
	}
}

func GetDelayedHandler(ReplicaNumber int) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		log.Println("Starting wait...")
		time.Sleep(20 * time.Second)
		log.Println("ending wait...")
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(TestServerDummyResponse{
			Message:   fmt.Sprintf("Response to URI '%v' from Replica #%v", req.URL, ReplicaNumber),
			ReplicaId: ReplicaNumber,
		})
	}
}

func getHealthHandlerFunc() func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(struct {
			active bool
		}{
			active: true,
		})
	}
}

func StartTestServers(replicasCount int) {
	serverPortStart := 8090

	for ReplicaNumber := 1; ReplicaNumber <= replicasCount; ReplicaNumber++ {
		srv := http.Server{}
		srv.Addr = fmt.Sprintf(":%v", serverPortStart+ReplicaNumber)
		srv.Handler = &mux.Router{}
		router := &mux.Router{}

		handlerFunc := GetNumberedHandler(ReplicaNumber)
		router.HandleFunc("/", handlerFunc).Methods("GET")
		router.HandleFunc("/:path", handlerFunc).Methods("GET")

		healthHandlerFunc := getHealthHandlerFunc()
		router.HandleFunc("/health", healthHandlerFunc).Methods("GET")

		delayedHandlerFunc := GetDelayedHandler(ReplicaNumber)
		router.HandleFunc("/delayed", delayedHandlerFunc).Methods("GET")

		fmt.Printf("API for replica #%v started\n", ReplicaNumber)
		go srv.ListenAndServe()
	}
}

func YAMLLine(step int, content string) (line string) {
	if step > 0 {
		line += strings.Repeat("  ", step)
	}
	line += content + "\n"
	return line
}

func generateBasicYAML(mode string, route string) string {
	yaml := `listeners:
  - protocol: http
    port: 8080
    ssl_certificate: test-value
    ssl_certificate_key: test-value
    routes:
      - routeprefix: "%s"
        mode: "%s"
        targets: 
          - address: http://localhost:8091
          - address: http://localhost:8092
          - address: http://localhost:8093
          - address: http://localhost:8093`
	return fmt.Sprintf(yaml, route, mode)
}

func StopTestServers() {

}

func doHTTPGetRequest(requestURL string, v any) *http.Response {
	res, err := http.Get(requestURL)
	if err != nil {
		fmt.Printf("error making http request: %s\n", err)
		os.Exit(1)
	}
	json.NewDecoder(res.Body).Decode(v)
	return res
}
