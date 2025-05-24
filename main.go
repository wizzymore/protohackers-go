package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/chat"
	"github.com/wizzymore/tcp-go/server"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

type Server interface {
	Start()
	Stop() error
}

func main() {
	command := ""
	for _, cmd := range os.Args[1:] {
		if strings.HasPrefix(cmd, "-") {
			continue
		}
		command = strings.ToLower(cmd)
		command = strings.Trim(command, " \t\n")
		break
	}
	if command == "" {
		log.Error().Msg("No command provided. Valid commands: [\"chat\", \"test\", \"prime-time\", \"means\"]")
		os.Exit(0)
	}

	var s Server
	var err error
	switch command {
	case "chat":
		s, err = chat.NewChatServer()
	case "test":
		s, err = server.NewServer(step_zero)
	case "prime-time":
		s, err = server.NewServer(step_one)
	case "means":
		s, err = server.NewServer(step_two)
	default:
		log.Fatal().Msgf("Unknown command: %s. Valid commands: [\"chat\", \"test\", \"prime-time\", \"means\"]", command)
	}
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
