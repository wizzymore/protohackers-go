package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/chat"
	"github.com/wizzymore/tcp-go/db"
	"github.com/wizzymore/tcp-go/mob"
	"github.com/wizzymore/tcp-go/server"
)

var logLevelFlag = flag.Int("log", int(zerolog.DebugLevel), "Set the log level: 0=debug, 1=info, 2=warn, 3=error, 4=fatal, 5=panic")
var colorFlag = flag.Bool("nocolor", false, "Disable colored log output")

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.Level(*logLevelFlag))

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: *colorFlag})
}

type ServerFunc func() (server.IServer, error)

var servers = map[string]ServerFunc{
	"mob":        mob.NewMobServer,
	"db":         db.NewDbServer,
	"chat":       chat.NewChatServer,
	"test":       func() (server.IServer, error) { return server.NewServer(step_zero) },
	"prime-time": func() (server.IServer, error) { return server.NewServer(step_one) },
	"means":      func() (server.IServer, error) { return server.NewServer(step_two) },
}

func serversList() string {
	var result []string
	for key := range servers {
		result = append(result, key)
	}
	slices.Sort(result)
	return strings.Join(result, ", ")
}

func main() {
	command := strings.Join(flag.Args(), " ")
	if command == "" {
		fmt.Printf("No command provided. Valid commands: [%s]\n", serversList())
		os.Exit(1)
	}

	var s server.IServer
	var err error

	if serverFunc, ok := servers[command]; ok {
		s, err = serverFunc()
	} else {
		fmt.Printf("Unknown command: %s. Valid commands: [%s]\n", command, serversList())
		os.Exit(1)
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
