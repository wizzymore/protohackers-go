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

func (self *MobServer) Start() {
	go self.server.Start()
}

func (self *MobServer) Stop() error {
	return self.server.Stop()
}

type message struct {
	source messageSource
	value  string
}

func (self *MobServer) HandleClient(c *server.TCPClient) error {
	c.Logger = c.Logger.With().Str("service", "mob").Logger()

	c.Logger.Debug().Msg("Dialing proxy server")
	bogusServer, err := net.Dial("tcp", self.proxyAddress)
	if err != nil {
		return errors.Join(err, errors.New("failted to connect to the proxy server"))
	}
	defer bogusServer.Close()

	proxyLog := c.Logger.With().Str("proxy_addr", bogusServer.LocalAddr().String()).Logger()
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
		clientReader := bufio.NewReader(c)
		for {
			select {
			case <-ctx.Done():
				c.Logger.Debug().Err(ctx.Err()).Msg("stopping client handler - context closed")
				return
			default:
			}

			text, err := clientReader.ReadString('\n')
			if err != nil {
				if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
					c.Logger.Error().Err(err).Msg("could not read from client")
				} else if errors.Is(err, io.EOF) {
					c.Logger.Info().Msg("client closed the connection")
				}
				ctx_cancel()
				return
			}
			text = strings.TrimSpace(text)

			c.Logger.Debug().Str("msg", text).Msg("received a new message from the client")

			messageChan <- message{
				source: FROM_CLIENT,
				value:  text,
			}
		}
	}()

	regex, err := regexp.Compile(`^7[a-zA-Z0-9]{25,34}$`)
	if err != nil {
		return err
	}

	proxyWriter := bufio.NewWriter(bogusServer)
	clientWriter := bufio.NewWriter(c)
loop:
	for {
		select {
		case <-ctx.Done():
			c.Logger.Debug().Err(ctx.Err()).Msg("Context done in main select")
			break loop
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
				return err
			}
			_, err = writer.WriteRune('\n')
			if err != nil {
				return err
			}
			err = writer.Flush()
		}
	}

	return nil
}
