package server

import (
	"net"

	"github.com/rs/zerolog/log"
)

type UDPHandler func(string, net.Addr)

type UDPServer struct {
	Running          bool
	Socket           *net.UDPConn
	Sync             bool
	handleConnection UDPHandler
}

func NewBaseUDPServer(handler UDPHandler, bindAddr ...string) (s *UDPServer, err error) {
	s = &UDPServer{}
	bind := ":8000"
	if len(bindAddr) > 0 {
		bind = bindAddr[0]
	}
	addr, err := net.ResolveUDPAddr("udp", bind)
	if err != nil {
		return
	}
	if s.Socket, err = net.ListenUDP("udp", addr); err != nil {
		return
	}
	s.handleConnection = handler
	return
}

func (s *UDPServer) Start() {
	addr := s.Socket.LocalAddr()
	log.Info().Msgf("server started on %s", addr.String())
	if s.Sync {
		log.Info().Msg("server running in sync")
	}
	for {
		// This will only read 512 bytes whatever i do to it
		buf := make([]byte, 1000)
		n, addr, err := s.Socket.ReadFromUDP(buf)
		if err != nil {
			if !s.Running {
				return
			}

			log.Error().Err(err).Msg("error reading from socket")
			continue
		}

		log.Info().Msgf("Received %d bytes from %s", n, addr.String())

		if s.Sync {
			go s.handleConnection(string(buf[:n]), addr)
		} else {
			s.handleConnection(string(buf[:n]), addr)
		}
	}
}

func (s *UDPServer) Stop() error {
	s.Running = false
	if err := s.Socket.Close(); err != nil {
		return err
	}
	return nil
}
