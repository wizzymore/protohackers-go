package main

import (
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

func main() {
	s, err := NewServer(step_one)
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

		for request := range strings.SplitSeq(string(buf[:n]), "\n") {
			if !handle_step_one(log, conn, request) {
				conn.Write([]byte("bye, bye\n"))
				return
			}
		}
	}
}
