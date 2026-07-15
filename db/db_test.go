package db_test

import (
	"bufio"
	mathrand "math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzymore/tcp-go/db"
)

func TestDB(t *testing.T) {
	s, err := db.NewDbServer()
	if err != nil {
		panic(err)
	}
	go s.Start()
	defer s.Stop()

	t.Run("can set value", func(t *testing.T) {
		conn, err := net.Dial("udp", ":8000")
		if err != nil {
			t.Error(err, "could not dial the server")
			return
		}
		conn.Write([]byte("foo=123"))
		conn.Write([]byte("foo"))
		scanner := bufio.NewScanner(conn)

		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 200))
		require.True(t, scanner.Scan(), "Could not read from the server")

		assert.Equal(t, []byte("foo=123"), scanner.Bytes(), "Did not set value correctly")
	})

	t.Run("can read unset value", func(t *testing.T) {
		conn, err := net.Dial("udp", ":8000")
		if err != nil {
			t.Error(err, "could not dial the server")
			return
		}
		rand_num := strconv.Itoa(mathrand.Int())
		conn.Write([]byte(rand_num))
		scanner := bufio.NewScanner(conn)

		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 200))
		require.True(t, scanner.Scan(), "Could not read from the server")

		assert.Equal(t, []byte(rand_num+"="), scanner.Bytes(), "Did not set value correctly")
	})

	t.Run("can't change version", func(t *testing.T) {
		conn, err := net.Dial("udp", ":8000")
		if err != nil {
			t.Error(err, "could not dial the server")
			return
		}
		conn.Write([]byte("version=123"))
		conn.Write([]byte("version"))
		scanner := bufio.NewScanner(conn)
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 200))
		require.True(t, scanner.Scan(), "Could not read from the server")

		assert.NotEqual(t, "version=123", scanner.Text(), "Did not set value correctly")
		assert.Equal(t, "version=alpha", scanner.Text(), "Did not set value correctly")
	})
}
