package chat

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/server"
)

type ChatSession struct {
	client   *server.TCPClient
	username string
}

type message struct {
	client *server.TCPClient
	value  string
}

type ChatServer struct {
	server   *server.TCPServer
	sessions map[*server.TCPClient]*ChatSession

	connected    chan *server.TCPClient
	disconnected chan *server.TCPClient
	message      chan message
}

func NewChatServer() (s server.Server, err error) {
	cs := &ChatServer{}
	cs.server, err = server.NewTCPServer(cs.HandleClient)
	cs.sessions = make(map[*server.TCPClient]*ChatSession)
	cs.connected = make(chan *server.TCPClient)
	cs.disconnected = make(chan *server.TCPClient)
	cs.message = make(chan message)
	return cs, err
}

func (chatServer *ChatServer) Start() {
	go chatServer.server.Start()

	go chatServer.runChatServer()
}

func (chatServer *ChatServer) Stop() error {
	return chatServer.server.Stop()
}

func (chatServer *ChatServer) runChatServer() {
	log := log.With().Str("service", "chat").Logger()
	log.Info().Msg("Chat server starter")
	usernameMaps := make(map[string]int)
	for {
		select {
		case client := <-chatServer.connected:
			log.Debug().Uint("peer", client.Id).Msg("New client connected")
			session := &ChatSession{
				client: client,
			}
			chatServer.sessions[client] = session
			session.writeLine("Please enter your username...")
		case client := <-chatServer.disconnected:
			log.Debug().Uint("peer", client.Id).Msg("Client disconnected")
			session := chatServer.sessions[client]
			delete(chatServer.sessions, client)
			if !session.IsConnected() {
				break
			}
			uCounter := usernameMaps[session.username]
			if uCounter <= 1 {
				delete(usernameMaps, session.username)
			} else {
				usernameMaps[session.username] = uCounter - 1
			}
			for _, sess := range chatServer.sessions {
				if sess.IsConnected() {
					sess.writeLine(fmt.Sprintf("* %s has left the room", session.username))
				}
			}
		case message := <-chatServer.message:
			log.Debug().Str("message", message.value).Msg("Received a new chat message")
			session := chatServer.sessions[message.client]
			if session.username == "" {
				if message.value == "" || !regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString(message.value) {
					session.client.Close()
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
				for _, sess := range chatServer.sessions {
					if session.client == sess.client || !sess.IsConnected() {
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
				log.Info().Uint("peer", session.client.Id).Str("name", session.username).Msg("Client set their name")
				break
			}

			if message.value == "" {
				break
			}

			for _, sess := range chatServer.sessions {
				if sess.client == session.client || !sess.IsConnected() {
					continue
				}
				sess.writeLine(fmt.Sprintf("[%s] %s", session.username, message.value))
			}
			log.Info().
				Uint("peer", session.client.Id).
				Str("name", session.username).
				Str("text", message.value).
				Msg("Client sent new text")
		}
	}
}

func (cs *ChatServer) HandleClient(c *server.TCPClient) error {
	defer func() {
		cs.disconnected <- c
	}()

	cs.connected <- c

	reader := bufio.NewReader(c)
	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			return err
		}

		cs.message <- struct {
			client *server.TCPClient
			value  string
		}{
			client: c,
			value:  strings.TrimRight(text, "\r\n"),
		}
	}

	return nil
}

func (chatSession *ChatSession) IsConnected() bool {
	return chatSession.username != ""
}

func (chatSession *ChatSession) writeLine(line string) error {
	line = strings.TrimRight(line, "\n")
	_, err := chatSession.client.Write(fmt.Appendf(nil, "%s\n", line))
	return err
}
