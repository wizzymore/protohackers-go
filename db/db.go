package db

import (
	"net"
	"strings"
	"sync"

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
	b := strings.Builder{}
	b.Grow(len(message) + len(value) + len("="))
	b.WriteString(message)
	b.WriteRune('=')
	b.WriteString(value)
	out := []byte(b.String())
	if len(out) > 1000 {
		out = out[0:1000]
	}
	s.server.Socket.WriteTo(out, addr)
	s.mu.RUnlock()
}
