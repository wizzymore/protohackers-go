package chat

import (
	"bytes"
	"fmt"
	"maps"
	"net"
	"regexp"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/server"
)

type ConnectedMessage struct {
	socket net.Conn
}

type DisconnectedMessage struct {
	socket net.Conn
}

type SentMessage struct {
	socket net.Conn
	text   string
}

type ChatSession struct {
	socket   net.Conn
	socket_m sync.RWMutex
	username string
}

type ChatServer struct {
	server.Server
	sessions map[net.Conn]*ChatSession
	messages chan any
}

func NewChatServer() (s ChatServer, err error) {
	s.Server, err = server.NewServer(s.HandleClient)
	s.sessions = make(map[net.Conn]*ChatSession)
	s.messages = make(chan any)
	return
}

func (chatSever *ChatServer) Start() {
	go chatSever.Server.Start()

	go chatSever.runChatServer()
}

func (chatServer *ChatServer) runChatServer() {
	log := log.With().Str("service", "CHAT-SERVER").Logger()
	log.Info().Msg("Chat server starter")
	usernameMaps := make(map[string]int)
	for {
		m := <-chatServer.messages
		switch message := m.(type) {
		case ConnectedMessage:
			log.Debug().Str("ip", message.socket.RemoteAddr().String()).Msg("New client connected")
			session := &ChatSession{
				socket: message.socket,
			}
			chatServer.sessions[message.socket] = session
			session.writeLine("Please enter your username...")
		case DisconnectedMessage:
			log.Debug().Str("ip", message.socket.RemoteAddr().String()).Msg("Client disconnected")
			session := chatServer.sessions[message.socket]
			delete(chatServer.sessions, message.socket)
			if !session.IsConnected() {
				break
			}
			uCounter := usernameMaps[session.username]
			usernameMaps[session.username] = uCounter - 1
			for sess := range maps.Values(chatServer.sessions) {
				if sess.IsConnected() {
					sess.writeLine(fmt.Sprintf("* %s has left the room", session.username))
				}
			}
		case SentMessage:
			log.Debug().Str("message", message.text).Msg("Received a new chat message")
			session := chatServer.sessions[message.socket]
			if session.username == "" {
				if message.text == "" || !regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString(message.text) {
					session.socket.Close()
					break
				}
				{
					uCounter, ok := usernameMaps[message.text]
					if !ok {
						session.username = message.text
						usernameMaps[session.username] = 1
					} else {
						session.username = fmt.Sprintf("%s%d", message.text, uCounter)
						usernameMaps[message.text] = uCounter + 1
					}
				}
				usernames := []string{}
				for sess := range maps.Values(chatServer.sessions) {
					if session.socket == sess.socket || !sess.IsConnected() {
						continue
					}
					usernames = append(usernames, sess.username)
					sess.writeLine(fmt.Sprintf("* %s has entered the room", session.username))
				}
				if len(usernames) > 0 {
					session.writeLine(fmt.Sprintf("* The room contains: %s", strings.Join(usernames, ", ")))
				} else {
					session.writeLine("* The room is currently empty")
				}
				break
			}

			if message.text == "" {
				break
			}

			for sess := range maps.Values(chatServer.sessions) {
				if sess.socket == session.socket || !sess.IsConnected() {
					continue
				}
				sess.writeLine(fmt.Sprintf("[%s] %s", session.username, message.text))
			}
		default:
			log.Error().Msgf("Message not implemented %#v", message)
		}
	}
}

func (cs *ChatServer) HandleClient(conn net.Conn) {
	log := log.With().Str("remote_addr", conn.RemoteAddr().String()).Logger()

	defer func() {
		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msg("Closing connection")
		conn.Close()
		cs.messages <- DisconnectedMessage{
			socket: conn,
		}
	}()

	cs.messages <- ConnectedMessage{
		socket: conn,
	}

	buffer := bytes.NewBuffer(nil)
	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if n > 0 {
			buffer.Write(buf[:n])
		}
		if err != nil {
			return
		}

		if buffer.Available() > 0 {
			if !strings.Contains(buffer.String(), "\n") {
				continue
			}
			text, _ := buffer.ReadString('\n')
			cs.messages <- SentMessage{
				socket: conn,
				text:   text[:len(text)-1],
			}
		}
	}
}

func (chatSession *ChatSession) IsConnected() bool {
	return chatSession.username != ""
}

func (chatSession *ChatSession) Disconnect(cs *ChatServer) {
	cs.messages <- DisconnectedMessage{
		socket: chatSession.socket,
	}
}

func (chatSession *ChatSession) writeLine(line string) error {
	chatSession.socket_m.Lock()
	defer chatSession.socket_m.Unlock()
	line = strings.TrimRight(line, "\n")
	_, err := chatSession.socket.Write(fmt.Appendf(nil, "%s\n", line))
	return err
}
