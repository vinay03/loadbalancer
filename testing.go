package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func GetNumberedHandler(ReplicaNumber int) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(struct {
			message string
		}{
			message: fmt.Sprintf("Response to URI '%v' from Replica #%v", req.URL, ReplicaNumber),
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
		json.NewEncoder(rw).Encode(struct {
			message string
		}{
			message: fmt.Sprintf("Response to URI '%v' from Replica #%v", req.URL, ReplicaNumber),
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

func StopTestServers() {

}
