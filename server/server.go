package server

import (
	"net"

	"github.com/rs/zerolog/log"
)

type Handler func(conn net.Conn)

type Server struct {
	running          bool
	listener         net.Listener
	handleConnection Handler
}

func NewServer(handler Handler) (s *Server, err error) {
	s = &Server{}
	s.listener, err = net.Listen("tcp", ":8080")
	s.handleConnection = handler
	return
}

func (s *Server) Start() {
	log.Info().Msgf("Server started on port %d", s.listener.Addr().(*net.TCPAddr).Port)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if !s.running {
				return
			}

			log.Error().Err(err).Msg("Error accepting connection")
			return
		}

		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msgf("Accepted connection from %s", conn.RemoteAddr())
		go s.handleConnection(conn)
	}
}

func (s *Server) Stop() error {
	s.running = false
	if err := s.listener.Close(); err != nil {
		return err
	}
	return nil
}
