package main

import (
	"encoding/json"
	"net"

	"github.com/rs/zerolog"
)

type StepOneResponse struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

func handle_step_one(log zerolog.Logger, conn net.Conn, request []byte) bool {
	log.Debug().Str("request", string(request)).Msg("Got message from client")
	var requestData map[string]interface{}
	if err := json.Unmarshal(request, &requestData); err != nil {
		log.Error().Err(err).Msg("Could not parse JSON request")
		return false
	}
	if !hasKey(requestData, "method") || requestData["method"] != "isPrime" || !hasKey(requestData, "number") {
		return false
	}
	var responseData StepOneResponse
	responseData.Method = "isPrime"
	if number, ok := requestData["number"].(float64); ok && number == float64(int(number)) {
		responseData.Prime = isPrime(int(number))
	} else {
		responseData.Prime = false
	}
	data, _ := json.Marshal(responseData)
	data = append(data, '\n')
	conn.Write(data)
	return true
}

func hasKey[T comparable, V any](m map[T]V, key T) bool {
	_, ok := m[key]
	return ok
}

func isPrime(n int) bool {
	if n <= 1 {
		return false
	}
	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}
