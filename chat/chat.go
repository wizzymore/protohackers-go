package chat

import (
	"bufio"
	"fmt"
	"maps"
	"net"
	"regexp"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/server"
)

type ChatSession struct {
	socket   net.Conn
	socket_m sync.RWMutex
	username string
}

type ChatServer struct {
	server   *server.Server
	sessions map[net.Conn]*ChatSession

	connected    chan net.Conn
	disconnected chan net.Conn
	message      chan struct {
		socket net.Conn
		value  string
	}
}

func NewChatServer() (s *ChatServer, err error) {
	s = &ChatServer{}
	s.server, err = server.NewServer(s.HandleClient)
	s.sessions = make(map[net.Conn]*ChatSession)
	s.connected = make(chan net.Conn)
	s.disconnected = make(chan net.Conn)
	s.message = make(chan struct {
		socket net.Conn
		value  string
	})
	return
}

func (chatServer *ChatServer) Start() {
	go chatServer.server.Start()

	go chatServer.runChatServer()
}

func (chatServer *ChatServer) Stop() error {
	return chatServer.server.Stop()
}

func (chatServer *ChatServer) runChatServer() {
	log := log.With().Str("service", "CHAT-SERVER").Logger()
	log.Info().Msg("Chat server starter")
	usernameMaps := make(map[string]int)
	for {
		select {
		case socket := <-chatServer.connected:
			log.Debug().Str("ip", socket.RemoteAddr().String()).Msg("New client connected")
			session := &ChatSession{
				socket: socket,
			}
			chatServer.sessions[socket] = session
			session.writeLine("Please enter your username...")
		case socket := <-chatServer.disconnected:
			log.Debug().Str("ip", socket.RemoteAddr().String()).Msg("Client disconnected")
			session := chatServer.sessions[socket]
			delete(chatServer.sessions, socket)
			if !session.IsConnected() {
				break
			}
			uCounter := usernameMaps[session.username]
			if uCounter <= 1 {
				delete(usernameMaps, session.username)
			} else {
				usernameMaps[session.username] = uCounter - 1
			}
			for sess := range maps.Values(chatServer.sessions) {
				if sess.IsConnected() {
					sess.writeLine(fmt.Sprintf("* %s has left the room", session.username))
				}
			}
		case message := <-chatServer.message:
			log.Debug().Str("message", message.value).Msg("Received a new chat message")
			session := chatServer.sessions[message.socket]
			if session.username == "" {
				if message.value == "" || !regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString(message.value) {
					session.socket.Close()
					break
				}
				{
					uCounter, ok := usernameMaps[message.value]
					if !ok {
						session.username = message.value
						usernameMaps[session.username] = 1
					} else {
						session.username = fmt.Sprintf("%s%d", message.value, uCounter)
						usernameMaps[message.value] = uCounter + 1
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
				log.Info().Str("ip", session.socket.RemoteAddr().String()).Str("name", session.username).Msg("Client set their name")
				break
			}

			if message.value == "" {
				break
			}

			for sess := range maps.Values(chatServer.sessions) {
				if sess.socket == session.socket || !sess.IsConnected() {
					continue
				}
				sess.writeLine(fmt.Sprintf("[%s] %s", session.username, message.value))
			}
			log.Info().Str("ip", session.socket.RemoteAddr().String()).Str("name", session.username).Str("text", message.value).Msg("Client sent new text")
		}
	}
}

func (cs *ChatServer) HandleClient(conn net.Conn) {
	log := log.With().Str("remote_addr", conn.RemoteAddr().String()).Logger()

	defer func() {
		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msg("Closing connection")
		conn.Close()
		cs.disconnected <- conn
	}()

	cs.connected <- conn

	reader := bufio.NewReader(conn)
	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		cs.message <- struct {
			socket net.Conn
			value  string
		}{
			socket: conn,
			value:  strings.TrimRight(text, "\r\n"),
		}
	}
}

func (chatSession *ChatSession) IsConnected() bool {
	return chatSession.username != ""
}

func (chatSession *ChatSession) writeLine(line string) error {
	chatSession.socket_m.Lock()
	defer chatSession.socket_m.Unlock()
	line = strings.TrimRight(line, "\n")
	_, err := chatSession.socket.Write(fmt.Appendf(nil, "%s\n", line))
	return err
}
