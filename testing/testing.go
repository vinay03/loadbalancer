package testing_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gorilla/mux"
)

var DebugMode bool = true

const (
	LISTENER_8080_URL = "http://localhost:8080/"
	LISTENER_8081_URL = "http://localhost:8081/"

	TestServerPortStart = 8090
)

var AllTestServersSync *sync.WaitGroup

type TestServerDummyResponse struct {
	Message   string            `json:"message"`
	ReplicaId int               `json:"replicaId"`
	Headers   map[string]string `json:"_headers"`
}

func GetNumberedHandler(testserver *TestServer, ReplicaNumber int, defaultDelayInterval time.Duration) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		delayInterval := defaultDelayInterval

		if req.Method == "POST" {
			var t struct {
				Delay int `json:"delay"`
			}
			err := json.NewDecoder(req.Body).Decode(&t)
			if err != nil {
				log.Error().Msg(err.Error())
			}
			if t.Delay >= 0 {
				delayInterval = time.Duration(t.Delay) * time.Second
			}
		}

		if delayInterval > 0 {
			log.Info().Msgf("Waiting for %v", delayInterval)
			time.Sleep(delayInterval)
			log.Info().Msgf("Wait for %v completed", delayInterval)
		}
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

type TestServer struct {
	Srv           *http.Server
	ReplicaNumber int
	Port          int
}

func NewTestServer(ReplicaNumber int) *TestServer {
	testserver := &TestServer{}
	port := TestServerPortStart + ReplicaNumber
	testserver.Port = port

	testserver.Srv = &http.Server{
		Addr: fmt.Sprintf(":%v", port),
	}
	testserver.Srv.Addr = fmt.Sprintf(":%v", port)

	router := &mux.Router{}

	handlerFunc := GetNumberedHandler(testserver, ReplicaNumber, 0*time.Second)
	router.HandleFunc("/", handlerFunc).Methods("GET")
	router.HandleFunc("/{path}", handlerFunc).Methods("GET", "POST")

	delayedHandlerFunc := GetNumberedHandler(testserver, ReplicaNumber, 0*time.Second)
	router.HandleFunc("/delayed", delayedHandlerFunc).Methods("GET", "POST")

	testserver.Srv.Handler = router
	return testserver
}

func (ts *TestServer) Start() {
	url := "http://localhost" + ts.Srv.Addr + "/"
	go TestServerCheckState(url, AllTestServersSync)
	go ts.Srv.ListenAndServe()
}
func (ts *TestServer) Stop() {
	ts.Srv.Shutdown(context.Background())
}

var TestServersPool []*TestServer

func StartTestServers(replicasCount int) {
	// serverPortStart := 8090

	TestServersPool = make([]*TestServer, replicasCount)
	AllTestServersSync = &sync.WaitGroup{}
	AllTestServersSync.Add(replicasCount)

	for index, _ := range TestServersPool {
		ReplicaNumber := index + 1
		testserver := NewTestServer(ReplicaNumber)
		TestServersPool[index] = testserver
		testserver.Start()
	}
	log.Info().Msg("Waiting till the test servers are up")
	AllTestServersSync.Wait()
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

func StopTestServers() {
	for _, testserver := range TestServersPool {
		testserver.Stop()
	}
}

type TestRequest struct {
	Address string
	Req     *http.Request
	Method  string
}

func Request(URL string) *TestRequest {
	tr := &TestRequest{}
	tr.Address = URL
	return tr
}

func (tr *TestRequest) Get() (*http.Response, *TestServerDummyResponse) {
	var err error
	tr.Req, err = http.NewRequest("GET", tr.Address, nil)
	if err != nil {
		fmt.Printf("error making http request: %s\n", err)
		os.Exit(1)
	}
	tr.Req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(tr.Req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	v := &TestServerDummyResponse{}
	json.NewDecoder(res.Body).Decode(v)
	return res, v
}

func GetDelayedRequestPayload(second int) string {
	jsonBytes, _ := json.Marshal(struct {
		Delay int `json:"delay"`
	}{
		Delay: second,
	})
	jsonString := string(jsonBytes)
	return jsonString
}

func (tr *TestRequest) Post(data string) (*http.Response, *TestServerDummyResponse) {
	var err error
	body := []byte(data)

	// Create a HTTP post request
	req, err := http.NewRequest("POST", tr.Address, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()

	v := &TestServerDummyResponse{}
	json.NewDecoder(res.Body).Decode(v)
	return res, v
}
