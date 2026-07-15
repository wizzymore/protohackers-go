package primetime

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/rs/zerolog/log"
)

type step_one_request struct {
	Method string   `json:"method"`
	Number *float64 `json:"number"`
}

type step_one_response struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

func handle_step_one(w io.Writer, request []byte) error {
	var requestData step_one_request
	if err := json.Unmarshal(request, &requestData); err != nil {
		return errors.Join(err, errors.New("could not parse JSON request"))
	}
	if requestData.Method != "isPrime" {
		if requestData.Method != "" {
			return fmt.Errorf("unsupported method: %s", requestData.Method)
		}
		return errors.New("method was not provided")
	}
	if requestData.Number == nil {
		return errors.New("provided number was nil")
	}
	var responseData step_one_response
	responseData.Method = "isPrime"
	if *requestData.Number == float64(int(*requestData.Number)) {
		responseData.Prime = isPrime(int(*requestData.Number))
	} else {
		responseData.Prime = false
	}
	data, err := json.Marshal(responseData)
	if err != nil {
		return errors.Join(err, fmt.Errorf("could not marshal responseData: %v", responseData))
	}
	w.Write(data)
	return nil
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

func Handler(conn net.Conn) {
	log := log.With().Str("remote_addr", conn.RemoteAddr().String()).Logger()
	defer func() {
		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msg("Closing connection")
		conn.Close()
	}()

	reader := bufio.NewReader(conn)
	writer := newWriter(conn)
	for {
		out, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				log.Info().Msg("client closed the connection")
				break
			}
			log.Err(err).Msg("client read error")
			break
		}
		log.Info().Msgf("got new data %s", out)

		if err := handle_step_one(writer, out); err != nil {
			log.Err(err).Msg("could not handle received input")
			conn.Write([]byte("bye, bye\n"))
			return
		}
	}
}

type NewLineWriter struct {
	w io.Writer
}

func (self NewLineWriter) Write(p []byte) (n int, err error) {
	n, err = self.w.Write(p)
	if err != nil {
		return
	}
	var n2 int
	n2, err = self.w.Write([]byte{'\n'})
	n = n + n2
	return
}

func newWriter(w io.Writer) io.Writer {
	return &NewLineWriter{w}
}
