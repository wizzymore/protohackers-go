package server

import (
	"net"

	"github.com/rs/zerolog/log"
)

type Handler func(conn net.Conn)

type Server struct {
	Running          bool
	Listener         net.Listener
	handleConnection Handler
}

func NewServer(handler Handler) (s *Server, err error) {
	s = &Server{}
	s.Listener, err = net.Listen("tcp", ":8081")
	s.handleConnection = handler
	return
}

func (s *Server) Start() {
	log.Info().Msgf("Server started on port %d", s.Listener.Addr().(*net.TCPAddr).Port)
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

func (s *Server) Stop() error {
	s.Running = false
	if err := s.Listener.Close(); err != nil {
		return err
	}
	return nil
}
