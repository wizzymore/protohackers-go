package means

import (
	"encoding/binary"
	"errors"
	"io"
	"slices"

	"github.com/wizzymore/tcp-go/reader"
	"github.com/wizzymore/tcp-go/server"
)

type StepTwoData struct {
	timestamp int32
	price     int32
}

func Handler(c *server.TCPClient) error {
	var data []StepTwoData

	for {
		buf := make([]byte, 1)
		n, err := c.Read(buf)
		if err != nil || len(buf) != n {
			if errors.Is(err, io.EOF) {
				break
			}

			return err
		}

		r := rune(buf[0])
		switch r {
		case 'I':
			d := StepTwoData{}
			reader.ReadB(c, &d.timestamp)
			reader.ReadB(c, &d.price)
			data = append(data, d)
		case 'Q':
			var minTime int32
			var maxTime int32
			reader.ReadB(c, &minTime)
			reader.ReadB(c, &maxTime)
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
			c.Logger.Info().Int32("avg", avg).Msg("Sent new average")
			binary.Write(c, binary.BigEndian, avg)
		default:
			c.Logger.Error().Msgf("Did not receive I or Q command, got: '%v'", r)
			continue
		}
	}

	return nil
}
