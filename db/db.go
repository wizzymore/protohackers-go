package db

import (
	"bytes"
	"slices"
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

func NewDbServer() (s server.Server, err error) {
	ch := make(chan any, 128)

	s, err = server.NewBaseUDPServer(func(c *server.UDPClient) error {
		return handleClient(c, ch)
	}, time.Second)

	if err != nil {
		return
	}

	go startServer(ch)

	return
}

func endsWithCRLF(s []byte) bool {
	return len(s) >= 2 && slices.Equal(s[len(s)-2:], []byte{'\r', '\n'})
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > 100 {
		s = s[:100] + "...(truncated)"
	}
	return s
}

func handleClient(c *server.UDPClient, ch chan any) error {
	// Buffer helper for building responses
	b := bytes.Buffer{}
	for {
		message, ok := <-c.Msgs
		if !ok {
			break
		}
		had_crlf := endsWithCRLF(message)

		// Write
		i := slices.Index(message, '=')
		if i != -1 {
			key := string(message[:i])
			value := string(message[i+1:])
			c.Logger.Info().Msgf("client sent a insert request for `%s` of `%s`", key, sanitize(value))

			if key == "version" {
				c.Logger.Debug().Msg("skipped insert of version value")
				continue
			}

			ch <- WriteEvent{key, value}
			continue
		}

		// Read

		key := string(message)
		c.Logger.Info().Msgf("client sent a get request for `%s`", key)

		if key == "delete" {
			ch <- DeleteEvent{}
			continue
		}

		var value string
		if key == "version" {
			value = VERSION
		} else {
			out := make(chan string, 1)
			ch <- ReadEvent{key, out}
			value = <-out
		}

		payloadSize := len(message) + len("=") + len(value)
		if had_crlf {
			payloadSize += 2
		}
		b.Reset()
		b.Grow(payloadSize)
		b.Write(message)
		b.WriteByte('=')
		b.WriteString(value)
		if had_crlf {
			b.Write([]byte{'\r', '\n'})
		}
		if err := c.Write(b.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func startServer(c chan any) {
	data := make(map[string]string)
	for {
		message, ok := <-c
		if !ok {
			return
		}
		switch m := message.(type) {
		case StopEvent:
			log.Info().Msg("Database handling server shutdown")
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
