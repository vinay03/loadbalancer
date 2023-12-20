package main

import (
	"encoding/json"
)

func PrettyPrint(anyData interface{}) string {
	b, err := json.MarshalIndent(anyData, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(b)
}
