package main

import (
	"encoding/json"
	"log"
)

func PrettyPrint(anyData interface{}) {
	b, err := json.MarshalIndent(anyData, "", "  ")
	if err != nil {
		log.Println(err)
	}
	log.Println(string(b))
}
