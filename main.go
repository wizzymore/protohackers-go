package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/chat"
	"github.com/wizzymore/tcp-go/mob"
	"github.com/wizzymore/tcp-go/server"
)

var logLevelFlag = flag.Int("log", int(zerolog.DebugLevel), "Set the log level: 0=debug, 1=info, 2=warn, 3=error, 4=fatal, 5=panic")

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	flag.Parse()
	fmt.Println("Log level set to:", *logLevelFlag)
	zerolog.SetGlobalLevel(zerolog.Level(*logLevelFlag))

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

type Server interface {
	Start()
	Stop() error
}

func main() {
	command := ""
	{
		should_skip := false
		for _, cmd := range os.Args[1:] {
			if strings.HasPrefix(cmd, "-") {
				should_skip = true
				continue
			}
			if should_skip {
				should_skip = false
				continue
			}
			command = strings.ToLower(cmd)
			command = strings.Trim(command, " \t\n")
			break
		}
	}
	if command == "" {
		os.Stderr.WriteString("No command provided. Valid commands: [\"mob\", \"chat\", \"test\", \"prime-time\", \"means\"]\n")
		os.Exit(0)
	}

	var s Server
	var err error
	switch command {
	case "mob":
		s, err = mob.NewMobServer()
	case "chat":
		s, err = chat.NewChatServer()
	case "test":
		s, err = server.NewServer(step_zero)
	case "prime-time":
		s, err = server.NewServer(step_one)
	case "means":
		s, err = server.NewServer(step_two)
	default:
		os.Stderr.WriteString(fmt.Sprintf("Unknown command: %s. Valid commands: [\"mob\", \"chat\", \"test\", \"prime-time\", \"means\"]\n", command))
		os.Exit(0)
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
