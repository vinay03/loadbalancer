package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gorilla/mux"
)

type TestServerDummyResponse struct {
	Message   string            `json:"message"`
	ReplicaId int               `json:"replicaId"`
	Headers   map[string]string `json:"_headers"`
}

func GetNumberedHandler(ReplicaNumber int) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)

		response := TestServerDummyResponse{
			Message:   fmt.Sprintf("Response to URI '%v' from Replica #%v", req.URL, ReplicaNumber),
			ReplicaId: ReplicaNumber,
		}
		response.Headers = make(map[string]string)
		for name, values := range req.Header {
			for _, value := range values {
				response.Headers[name] = value
			}
		}
		json.NewEncoder(rw).Encode(response)
	}
}

func GetDelayedHandler(ReplicaNumber int) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		// log.Println("Starting wait...")
		time.Sleep(20 * time.Second)
		// log.Println("ending wait...")
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		response := TestServerDummyResponse{
			Message:   fmt.Sprintf("Response to URI '%v' from Replica #%v", req.URL, ReplicaNumber),
			ReplicaId: ReplicaNumber,
		}
		response.Headers = make(map[string]string)
		for name, values := range req.Header {
			for _, value := range values {
				response.Headers[name] = value
			}
		}
		json.NewEncoder(rw).Encode(response)
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

var TestServersPool []*http.Server

func StartTestServers(replicasCount int) {
	serverPortStart := 8090

	TestServersPool = make([]*http.Server, replicasCount)
	TestServersSync := &sync.WaitGroup{}
	TestServersSync.Add(replicasCount)

	// for ReplicaNumber := 1; ReplicaNumber <= replicasCount; ReplicaNumber++ {
	for index, srv := range TestServersPool {
		// srv := http.Server{}
		srv = &http.Server{}
		TestServersPool[index] = srv
		ReplicaNumber := index + 1
		port := serverPortStart + ReplicaNumber
		srv.Addr = fmt.Sprintf(":%v", port)
		router := &mux.Router{}

		handlerFunc := GetNumberedHandler(ReplicaNumber)
		router.HandleFunc("/", handlerFunc).Methods("GET")
		router.HandleFunc("/:path", handlerFunc).Methods("GET")

		healthHandlerFunc := getHealthHandlerFunc()
		router.HandleFunc("/health", healthHandlerFunc).Methods("GET")

		delayedHandlerFunc := GetDelayedHandler(ReplicaNumber)
		router.HandleFunc("/delayed", delayedHandlerFunc).Methods("GET")

		srv.Handler = router
		url := "http://localhost" + srv.Addr + "/"
		go TestServerCheckState(url, TestServersSync)
		go srv.ListenAndServe()
	}
	log.Info().Msg("Waiting till the test servers are up")
	TestServersSync.Wait()
}

func TestServerCheckState(requestURL string, TestServerSync *sync.WaitGroup) {
	loopBreaker := 100
	time.Sleep(200 * time.Millisecond)
	for {
		res, err := http.Get(requestURL)
		if err != nil {
			log.Error().
				Msgf("Error making request to listener at '%v'", requestURL)
			break
		}
		if res.StatusCode == 200 {
			TestServerSync.Done()
			break
		} else {
			log.Info().Msgf("Response status '%v' from '%v ", res.StatusCode, requestURL)
		}
		time.Sleep(50 * time.Millisecond)
		loopBreaker--
		if loopBreaker <= 0 {
			log.Error().
				Msgf("Failed to start test server at : '%v'", requestURL)
			break
		}
	}
}

func YAMLLine(step int, content string) (line string) {
	if step > 0 {
		line += strings.Repeat("  ", step)
	}
	line += content + "\n"
	return line
}

func StopTestServers() {
	for _, srv := range TestServersPool {
		srv.Shutdown(context.Background())
	}
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
