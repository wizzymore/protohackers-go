package mob

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/server"
)

const BOGUS string = "7YWHMfk9JZe0LM0g1ZauHuiSxhI"

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
	socket net.Conn
	value  string
}

func (mobServer *MobServer) HandleClient(conn net.Conn) {
	log := log.With().
		Str("service", "mob").
		Str("remote_addr", conn.RemoteAddr().String()).Logger()

	defer func() {
		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msg("Closing connection")
		conn.Close()
	}()

	log.Info().Msg("Dialing proxy server")
	bogusServer, err := net.Dial("tcp", mobServer.proxyAddress)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to bogus server")
		return
	}
	defer bogusServer.Close()
	messageChan := make(chan message)

	regex, err := regexp.Compile("^7[a-zA-Z0-9]{25,34}$")
	if err != nil {
		log.Error().Err(err).Msg("Failed to compile regex")
		return
	}
	ctx, ctx_cancel := context.WithCancel(context.Background())
	defer ctx_cancel()

	go func() {
		reader := bufio.NewReader(bogusServer)
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("Context done in bogus server reader")
				return
			default:
			}

			text, err := reader.ReadString('\n')
			if err != nil {
				if !errors.Is(err, net.ErrClosed) {
					log.Error().Err(err).Msg("Error reading from bogus server")
				}
				ctx_cancel()
				return
			}

			messageChan <- message{
				socket: conn,
				value:  text,
			}
		}
	}()

	go func() {
		reader := bufio.NewReader(conn)
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("Context done in client reader")
				return
			default:
			}

			text, err := reader.ReadString('\n')
			if err != nil {
				ctx_cancel()
				return
			}

			messageChan <- message{
				socket: bogusServer,
				value:  text,
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("Context done in main select")
			return
		case msg := <-messageChan:
			splits := strings.Split(strings.TrimSpace(msg.value), " ")
			for i, split := range splits {
				if regex.MatchString(split) {
					splits[i] = BOGUS
				}
			}

			_, err = msg.socket.Write(fmt.Appendf(nil, "%s\n", strings.Join(splits, " ")))
			if err != nil {
				log.Error().Err(err).Msg("Error writing to client")
				return
			}
		}
	}
}
