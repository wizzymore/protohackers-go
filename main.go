package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/chat"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

func main() {
	// s, err := server.NewServer(step_two)
	s, err := chat.NewChatServer()
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
