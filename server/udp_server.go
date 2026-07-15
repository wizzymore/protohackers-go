package server

import (
	"errors"
	"net"
	"os"
	"slices"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const MAX_DATAGRAM_PACKET = 65_536
const MAX_DATAGRAM_SIZE = 65_507

type UDPHandler func(c *UDPClient) error

type UDPServer struct {
	Socket           net.PacketConn
	handleConnection UDPHandler
	timeout          time.Duration
}

type UDPClient struct {
	Msgs   chan []byte
	Logger zerolog.Logger

	conn         net.PacketConn
	addr         net.Addr
	lastActivity time.Time
}

func (self *UDPClient) Write(p []byte) (err error) {
	if len(p) > MAX_DATAGRAM_SIZE {
		return errors.New("UDP message too large")
	}
	self.conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
	n, err := self.conn.WriteTo(p, self.addr)
	if err != nil {
		return err
	}
	if n != len(p) {
		return errors.New("UDP message only partially sent")
	}
	return nil
}

func NewBaseUDPServer(handler UDPHandler, timeout time.Duration, bindAddr ...string) (s *UDPServer, err error) {
	s = &UDPServer{}
	addr := ":8000"
	if len(bindAddr) > 0 {
		addr = bindAddr[0]
	}
	if s.Socket, err = net.ListenPacket("udp", addr); err != nil {
		return
	}
	s.timeout = timeout
	s.handleConnection = handler
	return
}

func (self *UDPServer) Start() {
	addr := self.Socket.LocalAddr()
	log.Info().Msgf("server started on %s", addr.String())

	buf := make([]byte, MAX_DATAGRAM_PACKET)
	clients := make(map[string]*UDPClient)
	for {
		deadline := time.Time{}
		deadline_addr := ""

		for addr, c := range clients {
			t := c.lastActivity.Add(self.timeout)
			if deadline_addr == "" || t.Compare(deadline) == -1 {
				deadline = t
				deadline_addr = addr
			}
		}

		self.Socket.SetReadDeadline(deadline)
		n, addr, err := self.Socket.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if errors.Is(err, os.ErrDeadlineExceeded) {
				clients[deadline_addr].Logger.Debug().Msg("client timeout reached")
				close(clients[deadline_addr].Msgs)
				delete(clients, deadline_addr)
				continue
			}
			log.Fatal().Err(err).Msg("could not read from UDP socket")
		}

		connection_id := addr.String()
		c, has := clients[connection_id]
		if !has {
			c = &UDPClient{
				Msgs:         make(chan []byte),
				Logger:       log.With().Str("addr", connection_id).Logger(),
				conn:         self.Socket,
				addr:         addr,
				lastActivity: time.Now(),
			}
			clients[connection_id] = c
			go func(c *UDPClient) {
				c.Logger.Info().Msg("client connected")
				err := self.handleConnection(c)
				if err != nil {
					c.Logger.Err(err).Msg("client did not handle ok")
				} else {
					c.Logger.Info().Msg("client done - timed out")
				}
			}(c)
		}

		c.lastActivity = time.Now()
		c.Logger.Debug().Str("last_activity", c.lastActivity.Format("15:04:05")).Msgf("client sent %d bytes", n)
		c.Msgs <- slices.Clone(buf[:n])
	}
}

func (s *UDPServer) Stop() error {
	if err := s.Socket.Close(); err != nil {
		return err
	}
	return nil
}
