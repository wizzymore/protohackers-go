package main

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"slices"

	"github.com/rs/zerolog/log"
	"github.com/wizzymore/tcp-go/reader"
)

type StepTwoData struct {
	timestamp int32
	price     int32
}

func step_two(conn net.Conn) {
	log := log.With().Str("remote_addr", conn.RemoteAddr().String()).Logger()
	defer func() {
		log.Info().Str("remote_addr", conn.RemoteAddr().String()).Msg("Closing connection")
		conn.Close()
	}()

	var data []StepTwoData

	for {
		buf := make([]byte, 1)
		n, err := conn.Read(buf)
		if err != nil || len(buf) != n {
			if errors.Is(err, io.EOF) {
				log.Info().Msg("Client closed connection")
				return
			}

			logger := log.Error()
			if err != nil {
				logger.Err(err)
			} else {
				logger.Str("error", "Did not read enough bytes").Int("n", n).Int("buf_len", len(buf)).Bytes("buf", buf)
			}
			logger.Msg("Could not read from connection")
			return
		}

		r := rune(buf[0])
		// log.Info().Str("command", string(r)).Msg("Received new command")
		if r == 'I' {
			d := StepTwoData{}
			reader.ReadB(conn, &d.timestamp)
			reader.ReadB(conn, &d.price)
			// log.Info().
			// 	Int32("timestamp", d.timestamp).
			// 	Int32("price", d.price).
			// 	Msg("Received new data")
			data = append(data, d)
		} else if r == 'Q' {
			var minTime int32
			var maxTime int32
			reader.ReadB(conn, &minTime)
			reader.ReadB(conn, &maxTime)
			// log.Info().
			// 	Int32("minTime", minTime).
			// 	Int32("maxTime", maxTime).
			// 	Msg("Received new query")
			var sum int64
			var count int32
			for it := range slices.Values(data) {
				if minTime <= it.timestamp && it.timestamp <= maxTime {
					sum += int64(it.price)
					count++
				}
			}
			var avg int32
			if count > 0 {
				avg = int32(sum / int64(count))
			}
			log.Info().Int32("avg", avg).Msg("Sent new average")
			binary.Write(conn, binary.BigEndian, avg)
		} else {
			log.Error().Msgf("Did not receive I or Q command, got: '%v'", r)
			continue
		}
	}
}
