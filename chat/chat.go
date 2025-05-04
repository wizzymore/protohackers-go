package chat

import (
	"bytes"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/server"
)

type ConnectedMessage struct {
	session *ChatSession
}

type DisconnectedMessage struct {
	session *ChatSession
}

type SentMessage struct {
	session *ChatSession
	text    string
}

type ChatSession struct {
	socket    net.Conn
	username  string
	connected bool
	mutex     sync.RWMutex
}

type ChatServer struct {
	server.Server
	sessions   []*ChatSession
	m_sessions sync.RWMutex
	messages   chan any
}

func NewChatServer() (s ChatServer, err error) {
	s.Server, err = server.NewServer(s.HandleClient)
	s.messages = make(chan any)
	return
}

func (self *ChatServer) Start() {
	go self.Server.Start()

	go self.runChatServer()
}

func (self *ChatServer) runChatServer() {
	log := log.With().Str("service", "CHAT-SERVER").Logger()
	log.Info().Msg("Chat server starter")
	for {
		message := <-self.messages
		log.Debug().Any("message", message).Msg("Received a new message")
	free:
		switch v := message.(type) {
		case ConnectedMessage:
			// Set session as connected
			v.session.mutex.Lock()
			v.session.connected = true
			v.session.mutex.Unlock()

			// Add session to the list
			self.m_sessions.Lock()
			for session := range slices.Values(self.sessions) {
				if session.username == v.session.username {
					writeLine(v.session, "Username is already taken")
					v.session.socket.Close()
					self.m_sessions.Unlock()
					break free
				}
			}
			self.sessions = append(self.sessions, v.session)
			self.m_sessions.Unlock()

			self.m_sessions.RLock()
			for session := range slices.Values(self.sessions) {
				if session.username == v.session.username {
					writeLine(session, fmt.Sprintf("* Welcome to the chat room %s!", v.session.username))
					break
				}
				writeLine(session, fmt.Sprintf("* %s just joined!", v.session.username))
			}
			self.m_sessions.RUnlock()
		case DisconnectedMessage:
			self.m_sessions.Lock()
			var idx int
			for i, session := range self.sessions {
				if session.username == v.session.username {
					session.mutex.Lock()
					session.connected = false
					session.mutex.Unlock()
					idx = i
					continue
				}
				if session.connected {
					writeLine(session, fmt.Sprintf("* %s just disconnected!", v.session.username))
				}
			}
			if len(self.sessions) > 1 {
				self.sessions[idx] = self.sessions[len(self.sessions)-1]
				self.sessions = self.sessions[:len(self.sessions)-1]
			} else {
				self.sessions = []*ChatSession{}
			}
			self.m_sessions.Unlock()
		default:
			log.Error().Msgf("Message not implemented %#v", message)
		}
	}
}

func (cs *ChatServer) HandleClient(conn net.Conn) {
	log := log.With().Str("remote_addr", conn.RemoteAddr().String()).Logger()

	session := &ChatSession{
		socket: conn,
	}

	defer func() {
		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msg("Closing connection")
		conn.Close()
		session.Disconnect(cs)
	}()

	writeLine(session, "Please enter your username...")

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
		fmt.Println("Here")

		if buffer.Available() > 0 {
			session.ProcessClient(cs, buffer)
		}
	}
}

func (self *ChatSession) IsConnected() bool {
	return self.connected
}

func (self *ChatSession) Disconnect(cs *ChatServer) {
	cs.messages <- DisconnectedMessage{
		session: self,
	}
}

func (self *ChatSession) ProcessClient(cs *ChatServer, buffer *bytes.Buffer) {
	if self.username == "" {
		if strings.Index(buffer.String(), "\n") == -1 {
			return
		}

		{
			self.mutex.Lock()
			defer self.mutex.Unlock()
			self.username, _ = buffer.ReadString('\n')
			self.username = strings.TrimRight(self.username, "\n")
		}

		cs.messages <- ConnectedMessage{
			session: self,
		}
	}
	if !self.IsConnected() {
		return
	}
}

func writeLine(session *ChatSession, line string) error {
	session.mutex.Lock()
	defer session.mutex.Unlock()
	line = strings.TrimRight(line, "\n")
	_, err := session.socket.Write(fmt.Appendf(nil, "%s\n", line))
	return err
}
