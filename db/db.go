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

	// Write
	if pos := strings.Index(message, "="); pos != -1 {
		key := message[0:pos]
		value := message[pos+1:]

		log.Info().Msgf("Client %s sent a insert request for `%q` of `%q`", addr.String(), key, value)

		if key == "version" {
			return
		}

		s.mu.Lock()
		s.data[key] = value
		s.mu.Unlock()
		return
	}

	// Read

	log.Info().Msgf("Client %s sent a get request for `%q`", addr.String(), message)

	if message == "delete" {
		s.mu.Lock()
		s.data = make(map[string]string)
		s.mu.Unlock()
		return
	}

	value := ""
	if message == "version" {
		value = VERSION
	} else {
		s.mu.RLock()
		if v, ok := s.data[message]; ok {
			value = v
		}
		s.mu.RUnlock()
	}

	b := bytes.Buffer{}
	b.Grow(len(message) + len(value) + len("="))
	b.WriteString(message)
	b.WriteRune('=')
	b.WriteString(value)
	n, err := s.server.Socket.WriteTo(b.Bytes(), addr)
	if err != nil {
		log.Error().Msg("Failed to write message to socket")
	}
	log.Info().Str("value", value).Msgf("Wrote %d bytes to socket", n)
}
