package main

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/chat"
	"github.com/wizzymore/tcp-go/db"
	"github.com/wizzymore/tcp-go/means"
	"github.com/wizzymore/tcp-go/mob"
	"github.com/wizzymore/tcp-go/primetime"
	"github.com/wizzymore/tcp-go/server"
	"github.com/wizzymore/tcp-go/smoke_test"
	"github.com/wizzymore/tcp-go/traffic"
)

var logLevelFlag = flag.Int("log", int(zerolog.DebugLevel), "Set the log level: 0=debug, 1=info, 2=warn, 3=error, 4=fatal, 5=panic")
var colorFlag = flag.Bool("nocolor", false, "Disable colored log output")

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.Level(*logLevelFlag))
}

type ServerFunc func() (server.Server, error)

var servers = map[string]ServerFunc{
	"mob":        func() (server.Server, error) { return mob.NewMobServer() },
	"db":         db.NewDbServer,
	"chat":       chat.NewChatServer,
	"test":       func() (server.Server, error) { return server.NewTCPServer(smoke_test.Handler) },
	"prime-time": func() (server.Server, error) { return server.NewTCPServer(primetime.Handler) },
	"means":      func() (server.Server, error) { return server.NewTCPServer(means.Handler) },
	"traffic":    traffic.NewTrafficServer,
}

func serversList() string {
	return strings.Join(slices.Sorted(maps.Keys(servers)), ", ")
}

func main() {
	// Open log file
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	multiWriter := zerolog.MultiLevelWriter(
		zerolog.ConsoleWriter{Out: os.Stdout, NoColor: *colorFlag},
		logFile,
	)
	log.Logger = log.Output(multiWriter)

	command := strings.Join(flag.Args(), " ")
	if command == "" {
		fmt.Printf("No command provided. Valid commands: [%s]\n", serversList())
		os.Exit(1)
	}

	var s server.Server

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
