package db

import (
	"bytes"
	"net"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/server"
)

const VERSION = "alpha"

type WriteEvent struct {
	key   string
	value string
}
type ReadEvent struct {
	key string
	out chan string
}
type DeleteEvent struct{}
type StopEvent struct{}

type DbServer struct {
	server *server.UDPServer
	c      chan any
}

func NewDbServer(sync bool, bindAddr ...string) (s server.Server, err error) {
	db := &DbServer{}
	addr := ":8000"
	if len(bindAddr) > 0 {
		addr = bindAddr[0]
	}
	s = db
	db.server, err = server.NewBaseUDPServer(db.HandleClient, addr)
	if err != nil {
		return
	}
	db.server.Sync = sync
	db.c = make(chan any, 32)
	go db.startServer()
	return
}

func (self *DbServer) Start() {
	go self.server.Start()
}

func (self *DbServer) Stop() error {
	self.c <- StopEvent{}
	return self.server.Stop()
}

func endsWithCRLF(s string) bool {
	return len(s) >= 2 && s[len(s)-2:] == "\r\n"
}

func (self *DbServer) HandleClient(message string, addr net.Addr) {
	self.server.Socket.SetWriteDeadline(time.Now().Add(10 * time.Second))
	log := log.With().Str("remote_addr", addr.String()).Logger()
	had_crlf := endsWithCRLF(message)

	// Write
	if key, value, ok := strings.Cut(message, "="); ok {
		log.Info().Msgf("client %s sent a insert request for `%s` of `%s`", addr.String(), key, value)

		if key == "version" {
			return
		}

		self.c <- WriteEvent{key, value}
		return
	}

	// Read

	log.Info().Msgf("Client %s sent a get request for `%s`", addr.String(), message)

	if message == "delete" {
		self.c <- DeleteEvent{}
		return
	}

	value := ""
	if message == "version" {
		value = VERSION
	} else {
		out := make(chan string, 1)
		self.c <- ReadEvent{key: message, out: out}
		value = <-out
	}

	b := bytes.Buffer{}
	payloadSize := len(message) + len("=") + len(value)
	if had_crlf {
		payloadSize += 2
	}
	b.Grow(payloadSize)
	b.WriteString(message)
	b.WriteByte('=')
	b.WriteString(value)
	if had_crlf {
		b.WriteString("\r\n")
	}
	payload := b.Bytes()
	// Write payload limited to 1000 bytes per protohackers requirement
	n, err := self.server.Socket.WriteTo(payload[:min(len(payload), 1000)], addr)
	if err != nil {
		log.Err(err).Msg("failed to write message to socket")
	}
	log.Info().Str("value", value).Msgf("Wrote %d bytes to socket", n)
}

func (self DbServer) startServer() {
	data := make(map[string]string)
	for {
		message, ok := <-self.c
		if !ok {
			return
		}
		switch m := message.(type) {
		case StopEvent:
			return
		case WriteEvent:
			data[m.key] = m.value
		case ReadEvent:
			m.out <- data[m.key]
		case DeleteEvent:
			data = make(map[string]string)
		}
	}
}
