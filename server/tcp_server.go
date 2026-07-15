package server

import (
	"net"

	"github.com/rs/zerolog/log"
)

type ServerHandle func(conn net.Conn)

type Server interface {
	Start()
	Stop() error
}

type TCPServer struct {
	Running          bool
	Listener         net.Listener
	handleConnection ServerHandle
}

func NewTCPServer(handler ServerHandle) (s *TCPServer, err error) {
	s = &TCPServer{}
	s.Listener, err = net.Listen("tcp", ":8000")
	s.handleConnection = handler
	return
}

func (s *TCPServer) Start() {
	addr := s.Listener.Addr()
	log.Info().Msgf("Server started on %s", addr.String())
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if !s.Running {
				return
			}

			log.Error().Err(err).Msg("Error accepting connection")
			return
		}

		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msgf("Accepted connection from %s", conn.RemoteAddr())
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) Stop() error {
	s.Running = false
	if err := s.Listener.Close(); err != nil {
		return err
	}
	return nil
}
