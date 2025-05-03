package main

import (
	"encoding/json"
	"net"

	"github.com/rs/zerolog"
)

type StepOneRequest struct {
	Method *string `json:"method"`
	Number *int    `json:"number"`
}

type StepOneResponse struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

func handle_step_one(log zerolog.Logger, conn net.Conn, request []byte) bool {
	log.Debug().Str("request", string(request)).Msg("Got message from client")
	var requestData StepOneRequest
	if err := json.Unmarshal(request, &requestData); err != nil {
		log.Error().Err(err).Msg("Could not parse JSON request")
		return false
	}
	if requestData.Method == nil || *requestData.Method != "prime" || requestData.Number == nil {
		return false
	}
	var responseData StepOneResponse
	responseData.Method = "prime"
	responseData.Prime = isPrime(*requestData.Number)
	data, _ := json.Marshal(responseData)
	data = append(data, '\n')
	conn.Write(data)
	return true
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
