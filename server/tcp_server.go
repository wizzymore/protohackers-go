package server

import (
	"errors"
	"io"
	"net"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type TCPHandle func(*TCPClient) error

type TCPServer struct {
	Listener         net.Listener
	handleConnection TCPHandle
}

type TCPClient struct {
	net.Conn
	Id     uint
	Logger zerolog.Logger
}

func NewTCPServer(handler TCPHandle) (s *TCPServer, err error) {
	s = &TCPServer{}
	s.Listener, err = net.Listen("tcp", ":8000")
	s.handleConnection = handler
	return
}

func (s *TCPServer) Start() {
	addr := s.Listener.Addr()
	log.Info().Msgf("server started on %s", addr.String())
	var nextId uint = 1
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Fatal().Err(err).Msg("error accepting connection")
		}

		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msgf("accepted connection from %s", conn.RemoteAddr())
		id := nextId
		nextId += 1
		go func(conn net.Conn, id uint) {
			c := &TCPClient{conn, id, log.With().Uint("peer", id).Logger()}
			defer c.Close()
			c.Logger.Info().Msg("connected")
			err := s.handleConnection(c)
			if err != nil && !errors.Is(err, net.ErrClosed) {
				if errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET) {
					c.Logger.Info().Msg("client closed the connection")
				} else {
					c.Logger.Err(err).Msg("client did not handle ok")
				}
			} else {
				c.Logger.Info().Msg("client done")
			}
		}(conn, id)
	}
}

func (s *TCPServer) Stop() error {
	if err := s.Listener.Close(); err != nil {
		return err
	}
	return nil
}
