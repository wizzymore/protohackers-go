package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/reader"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

func main() {
	s, err := NewServer(step_two)
	if err != nil {
		log.Error().Err(err).Msg("Error creating server")
		return
	}

	doneCh := make(chan struct{})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Handle interrupt signal
	go func() {
		<-sigCh
		log.Info().Msg("Received interrupt signal")
		doneCh <- struct{}{}
	}()

	go s.Start()

	<-doneCh

	log.Info().Msg("Shutting down server...")
	if err := s.Stop(); err != nil {
		log.Error().Err(err).Msg("Error stopping server")
	} else {
		log.Info().Msg("Server stopped")
	}
}

func step_zero(conn net.Conn) {
	log := log.With().Str("remote_addr", conn.RemoteAddr().String()).Logger()
	defer func() {
		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msg("Closing connection")
		conn.Close()
	}()

	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				log.Info().Msg("Client closed the connection")
				return
			}
			log.Err(err).Msg("Client read error")
			return
		}

		conn.Write(buf[:n])
	}
}

func step_one(conn net.Conn) {
	log := log.With().Str("remote_addr", conn.RemoteAddr().String()).Logger()
	defer func() {
		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msg("Closing connection")
		conn.Close()
	}()

	left := []byte{}
	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		log.Info().Msgf("Got new buffer %s", strings.ReplaceAll(string(buf), "\n", "\\n"))
		if err != nil {
			if err == io.EOF {
				log.Info().Msg("Client closed the connection")
				return
			}
			log.Err(err).Msg("Client read error")
			return
		}

		start := 0
		for i := range buf[:n] {
			if buf[i] == '\n' {
				left = append(left, buf[start:i]...)
				if !handle_step_one(log, conn, left) {
					conn.Write([]byte("bye, bye\n"))
					return
				}
				left = []byte{}
				start = i + 1
			} else if i == n-1 {
				left = append(left, buf[start:n]...)
			}
		}
	}
}

type StepThreeData struct {
	timestamp int32
	price     int32
}

func step_two(conn net.Conn) {
	log := log.With().Str("remote_addr", conn.RemoteAddr().String()).Logger()
	defer func() {
		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msg("Closing connection")
		conn.Close()
	}()

	var data []StepThreeData

	for {
		buf := make([]byte, 9)
		n, err := conn.Read(buf)
		if err != nil || len(buf) != n {
			logger := log.Error()
			if err != nil {
				logger.Err(err)
			} else {
				logger.Str("error", "Did not read enough bytes").Int("n", n).Int("buf", len(buf))
			}
			logger.Msg("Could not read from connection")
			return
		}

		buffer := bytes.NewBuffer(buf)
		r, _, err := buffer.ReadRune()
		if err != nil {
			log.Error().Err(err).Msg("Could not read rune")
			return
		}

		if r == 'I' {
			d := StepThreeData{}
			reader.ReadB(buffer, &d.timestamp)
			reader.ReadB(buffer, &d.price)
			data = append(data, d)
		} else if r == 'Q' {
			var minTime int32
			var maxTime int32
			reader.ReadB(buffer, &minTime)
			reader.ReadB(buffer, &maxTime)
			var sum int32
			var count int32
			for it := range slices.Values(data) {
				if minTime <= it.timestamp && it.timestamp <= maxTime {
					sum += it.price
					count++
				}
			}
			avg := sum / count
			binary.Write(conn, binary.BigEndian, avg)
		} else {
			log.Error().Msgf("Did not receive I or Q command, got: '%v'", r)
			continue
		}
	}
}
