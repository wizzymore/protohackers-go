package db

import (
	"bytes"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/server"
)

const VERSION = "alpha"

type DbServer struct {
	server *server.UdpServer
	data   map[string]string
	mu     sync.RWMutex
}

func NewDbServer() (s *DbServer, err error) {
	s = &DbServer{}
	s.data = make(map[string]string)
	s.server, err = server.NewUdpServer(s.HandleClient)
	return
}

func (chatServer *DbServer) Start() {
	go chatServer.server.Start()
}

func (chatServer *DbServer) Stop() error {
	return chatServer.server.Stop()
}

func (s *DbServer) HandleClient(message string, addr net.Addr) {
	s.server.Socket.SetWriteDeadline(time.Now().Add(10 * time.Minute))
	log := log.With().Str("remote_addr", addr.String()).Logger()
	log.Info().Str("message", message).Msg("Got new message")

	// Write
	if strings.Contains(message, "=") {
		split := strings.SplitN(message, "=", 2)
		key := split[0]
		value := split[1]

		if key == "version" {
			return
		}

		log.Info().Str("key", key).Str("value", value).Msg("Got a new write")

		s.mu.Lock()
		s.data[key] = value
		s.mu.Unlock()
		return
	}

	// Read
	s.mu.RLock()
	value := ""
	if message == "version" {
		value = VERSION
	} else {
		if v, ok := s.data[message]; ok {
			value = v
		}
	}
	if value == "" {
		return
	}
	log.Info().Str("key", message).Msg("Got a new read")
	b := bytes.Buffer{}
	b.Grow(len(message) + len(value) + len("="))
	b.WriteString(message)
	b.WriteRune('=')
	b.WriteString(value)
	n, err := s.server.Socket.WriteTo(b.Bytes(), addr)
	if err != nil {
		log.Error().Msg("Failed to write message to socket")
	}
	log.Info().Int("bytes", n).Msg("Wrote bytes to socket")
	s.mu.RUnlock()
}
