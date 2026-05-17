package db_test

import (
	"bufio"
	"bytes"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzymore/tcp-go/db"
)

func init() {
	log.Logger = zerolog.New(&bytes.Buffer{})
	s, _ := db.NewDbServer()
	s.Start()
}

func TestCanSetValue(t *testing.T) {
	conn, err := net.Dial("udp", "localhost:8081")
	if err != nil {
		t.Error("Could not dial", err)
		return
	}
	conn.Write([]byte("foo=123"))
	conn.Write([]byte("foo"))
	scanner := bufio.NewScanner(conn)

	conn.SetReadDeadline(time.Now().Add(time.Millisecond * 200))
	require.True(t, scanner.Scan(), "Could not read from the server")

	assert.Equal(t, []byte("foo=123"), scanner.Bytes(), "Did not set value correctly")
}

func TestCanReadUnsetValue(t *testing.T) {
	conn, err := net.Dial("udp", "localhost:8081")
	if err != nil {
		t.Error("Could not dial", err)
		return
	}
	rand_num := strconv.Itoa(rand.Int())
	conn.Write([]byte(rand_num))
	scanner := bufio.NewScanner(conn)

	conn.SetReadDeadline(time.Now().Add(time.Millisecond * 200))
	require.True(t, scanner.Scan(), "Could not read from the server")

	assert.Equal(t, []byte(rand_num+"="), scanner.Bytes(), "Did not set value correctly")
}

func TestCantChangeVersion(t *testing.T) {
	conn, err := net.Dial("udp", "localhost:8081")
	if err != nil {
		t.Error("Could not dial", err)
		return
	}
	conn.Write([]byte("version=123"))
	conn.Write([]byte("version"))
	scanner := bufio.NewScanner(conn)
	conn.SetReadDeadline(time.Now().Add(time.Millisecond * 200))
	require.True(t, scanner.Scan(), "Could not read from the server")

	assert.NotEqual(t, "version=123", scanner.Text(), "Did not set value correctly")
	assert.Equal(t, "version=alpha", scanner.Text(), "Did not set value correctly")
}
