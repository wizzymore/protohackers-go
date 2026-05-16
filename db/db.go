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

func NewDbServer() (s server.IServer, err error) {
	db := &DbServer{}
	db.data = make(map[string]string)
	db.server, err = server.NewUdpServer(db.HandleClient)
	return db, err
}

func (dbServer *DbServer) Start() {
	go dbServer.server.Start()
}

func (dbServer *DbServer) Stop() error {
	return dbServer.server.Stop()
}

func (dbServer *DbServer) HandleClient(message string, addr net.Addr) {
	dbServer.server.Socket.SetWriteDeadline(time.Now().Add(10 * time.Minute))
	log := log.With().Str("remote_addr", addr.String()).Logger()

	// Write
	if key, value, ok := strings.Cut(message, "="); ok {
		log.Info().Msgf("Client %s sent a insert request for `%q` of `%q`", addr.String(), key, value)

		if key == "version" {
			return
		}

		dbServer.mu.Lock()
		dbServer.data[key] = value
		dbServer.mu.Unlock()
		return
	}

	// Read

	log.Info().Msgf("Client %s sent a get request for `%q`", addr.String(), message)

	if message == "delete" {
		dbServer.mu.Lock()
		dbServer.data = make(map[string]string)
		dbServer.mu.Unlock()
		return
	}

	value := ""
	if message == "version" {
		value = VERSION
	} else {
		dbServer.mu.RLock()
		if v, ok := dbServer.data[message]; ok {
			value = v
		}
		dbServer.mu.RUnlock()
	}

	b := bytes.Buffer{}
	b.Grow(len(message) + len(value) + len("="))
	b.WriteString(message)
	b.WriteRune('=')
	b.WriteString(value)
	n, err := dbServer.server.Socket.WriteTo(b.Bytes(), addr)
	if err != nil {
		log.Error().Msg("Failed to write message to socket")
	}
	log.Info().Str("value", value).Msgf("Wrote %d bytes to socket", n)
}
