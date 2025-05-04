package main

import (
	"io"
	"net"

	"github.com/rs/zerolog/log"
)

func step_zero(conn net.Conn) {
	log := log.With().Str("remote_addr", conn.RemoteAddr().String()).Logger()
	defer func() {
		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msg("Closing connection")
		conn.Close()
	}()

	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				log.Info().Msg("Client closed the connection")
				return
			}
			log.Err(err).Msg("Client read error")
			return
		}

		conn.Write(buf[:n])
	}
}
