package smoke_test

import (
	"io"

	"github.com/wizzymore/tcp-go/server"
)

func Handler(c *server.TCPClient) error {
	for {
		buf := make([]byte, 1024)
		n, err := c.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			c.Logger.Err(err).Msg("Client read error")
			return err
		}

		c.Write(buf[:n])
	}
}
