package mob

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"regexp"
	"strings"
	"unicode"

	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/server"
)

const BOGUS string = "7YWHMfk9JZe0LM0g1ZauHuiSxhI"

type messageSource int

const (
	FROM_CLIENT messageSource = iota
	FROM_PROXY
)

type MobServer struct {
	server       *server.TCPServer
	proxyAddress string
}

func NewMobServer(proxyAddress ...string) (s server.Server, err error) {
	mob := &MobServer{}
	if len(proxyAddress) == 0 {
		mob.proxyAddress = "chat.protohackers.com:16963"
	} else {
		mob.proxyAddress = proxyAddress[0]
	}
	mob.server, err = server.NewTCPServer(mob.HandleClient)
	return mob, err
}

func (mobServer *MobServer) Start() {
	go mobServer.server.Start()
}

func (mobServer *MobServer) Stop() error {
	return mobServer.server.Stop()
}

type message struct {
	source messageSource
	value  string
}

func (mobServer *MobServer) HandleClient(conn net.Conn) {
	log := log.With().
		Str("service", "mob").
		Str("remote_addr", conn.RemoteAddr().String()).Logger()

	defer func() {
		log.Info().Msg("closing the client connection")
		conn.Close()
	}()

	log.Debug().Msg("Dialing proxy server")
	bogusServer, err := net.Dial("tcp", mobServer.proxyAddress)
	if err != nil {
		log.Error().Err(err).Msg("failed to connect to the proxy server")
		return
	}
	defer bogusServer.Close()
	proxyLog := log.With().Str("proxy_addr", bogusServer.LocalAddr().String()).Logger()
	proxyLog.Info().Msg("Connected to the proxy server")

	// Communication channel between the proxy and our client
	messageChan := make(chan message, 32)

	ctx, ctx_cancel := context.WithCancel(context.Background())
	defer ctx_cancel()

	// Proxy communication handler
	go func() {
		proxyReader := bufio.NewReader(bogusServer)
		for {
			select {
			case <-ctx.Done():
				proxyLog.Debug().Err(ctx.Err()).Msg("stopping proxy handler - context closed")
				return
			default:
			}

			text, err := proxyReader.ReadString('\n')
			if err != nil {
				if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
					proxyLog.Error().Err(err).Msg("could not read from proxy server")
				} else if errors.Is(err, io.EOF) {
					proxyLog.Info().Msg("proxy server closed the connection")
				}
				ctx_cancel()
				return
			}
			text = strings.TrimSpace(text)

			proxyLog.Debug().Str("msg", text).Msg("received a new message from the proxy")

			messageChan <- message{
				source: FROM_PROXY,
				value:  text,
			}
		}
	}()

	// Client communication handler
	go func() {
		clientReader := bufio.NewReader(conn)
		for {
			select {
			case <-ctx.Done():
				log.Debug().Err(ctx.Err()).Msg("stopping client handler - context closed")
				return
			default:
			}

			text, err := clientReader.ReadString('\n')
			if err != nil {
				if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
					log.Error().Err(err).Msg("could not read from client")
				} else if errors.Is(err, io.EOF) {
					log.Info().Msg("client closed the connection")
				}
				ctx_cancel()
				return
			}
			text = strings.TrimSpace(text)

			log.Debug().Str("msg", text).Msg("received a new message from the client")

			messageChan <- message{
				source: FROM_CLIENT,
				value:  text,
			}
		}
	}()

	regex, err := regexp.Compile(`^7[a-zA-Z0-9]{25,34}$`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to compile regex")
		return
	}

	proxyWriter := bufio.NewWriter(bogusServer)
	clientWriter := bufio.NewWriter(conn)
	for {
		select {
		case <-ctx.Done():
			log.Debug().Err(ctx.Err()).Msg("Context done in main select")
			return
		case msg := <-messageChan:
			words := strings.FieldsFunc(msg.value, func(r rune) bool {
				return unicode.IsSpace(r)
			})
			for i, word := range words {
				if regex.MatchString(word) {
					words[i] = BOGUS
				}
			}
			msg.value = strings.Join(words, " ")

			var writer *bufio.Writer
			switch msg.source {
			case FROM_CLIENT:
				writer = proxyWriter
			case FROM_PROXY:
				writer = clientWriter
			}

			_, err = writer.WriteString(msg.value)
			if err != nil {
				log.Error().Err(err).Msg("Error writing to client")
				return
			}
			_, err = writer.WriteRune('\n')
			if err != nil {
				log.Error().Err(err).Msg("Error writing to client")
				return
			}
			err = writer.Flush()
		}
	}
}
