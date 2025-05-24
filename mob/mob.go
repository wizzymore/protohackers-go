package mob

import (
	"bufio"
	"net"
	"regexp"

	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/server"
)

const BOGUS string = "7YWHMfk9JZe0LM0g1ZauHuiSxhI"

type MobServer struct {
	server *server.Server
}

func NewMobServer() (s *MobServer, err error) {
	s = &MobServer{}
	s.server, err = server.NewServer(s.HandleClient)
	return
}

func (chatServer *MobServer) Start() {
	go chatServer.server.Start()
}

func (chatServer *MobServer) Stop() error {
	return chatServer.server.Stop()
}

type message struct {
	socket net.Conn
	value  string
}

func (cs *MobServer) HandleClient(conn net.Conn) {
	log := log.With().Str("remote_addr", conn.RemoteAddr().String()).Logger()

	defer func() {
		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msg("Closing connection")
		conn.Close()
	}()

	bogusServer, err := net.Dial("tcp", "chat.protohackers.com:16963")
	defer bogusServer.Close()
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to bogus server")
		return
	}
	closeChan := make(chan struct{})
	messageChan := make(chan message)

	regex, err := regexp.Compile("7[a-zA-Z0-9]{25,35}")
	if err != nil {
		log.Error().Err(err).Msg("Failed to compile regex")
		return
	}

	go func() {
		reader := bufio.NewReader(bogusServer)
		for {
			select {
			case <-closeChan:
				return
			default:
			}

			text, err := reader.ReadString('\n')
			if err != nil {
				log.Error().Err(err).Msg("Error reading from bogus server")
				closeChan <- struct{}{}
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
			case <-closeChan:
				return
			default:
			}

			text, err := reader.ReadString('\n')
			if err != nil {
				closeChan <- struct{}{}
				return
			}

			messageChan <- message{
				socket: bogusServer,
				value:  text,
			}
		}
	}()

	for msg := range messageChan {
		text := regex.ReplaceAllString(msg.value, BOGUS)
		_, err = msg.socket.Write([]byte(text))
		if err != nil {
			log.Error().Err(err).Msg("Error writing to client")
			return
		}
	}
}

// func (chatSession *ChatSession) writeLine(line string) error {
// 	chatSession.socket_m.Lock()
// 	defer chatSession.socket_m.Unlock()
// 	line = strings.TrimRight(line, "\n")
// 	_, err := chatSession.socket.Write(fmt.Appendf(nil, "%s\n", line))
// 	return err
// }
