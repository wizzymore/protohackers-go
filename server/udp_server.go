package server

import (
	"net"

	"github.com/rs/zerolog/log"
)

type UdpHandler func(string, net.Addr)

type UdpServer struct {
	Running          bool
	Socket           net.PacketConn
	Async            bool
	handleConnection UdpHandler
}

func NewUdpServer(handler UdpHandler) (s *UdpServer, err error) {
	s = &UdpServer{}
	s.Socket, err = net.ListenPacket("udp", ":8081")
	s.handleConnection = handler
	return
}

func (s *UdpServer) Start() {
	log.Info().Msgf("Server started on port %d", s.Socket.LocalAddr().(*net.UDPAddr).Port)
	if !s.Async {
		log.Info().Msg("Running in sync")
	}
	for {
		buf := make([]byte, 1000)
		n, addr, err := s.Socket.ReadFrom(buf)
		if err != nil {
			if !s.Running {
				return
			}

			log.Error().Err(err).Msg("Error reading from socket")
			continue
		}

		log.Info().Msgf("Received %d bytes from %s", n, addr.String())

		if s.Async {
			go s.handleConnection(string(buf[0:n]), addr)
		} else {
			s.handleConnection(string(buf[0:n]), addr)
		}
	}
}

func (s *UdpServer) Stop() error {
	s.Running = false
	if err := s.Socket.Close(); err != nil {
		return err
	}
	return nil
}
