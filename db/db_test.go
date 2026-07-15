package db_test

import (
	"bufio"
	"crypto/rand"
	"math/big"
	mathrand "math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzymore/tcp-go/db"
)

func init() {
	s, err := db.NewDbServer(false, "127.0.0.1:8000")
	if err != nil {
		panic(err)
	}
	s.Start()
}

func TestCanSetValue(t *testing.T) {
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
}

func TestCanReadUnsetValue(t *testing.T) {
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
}

func TestCantChangeVersion(t *testing.T) {
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
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateRandomString creates a cryptographically secure random string
func GenerateRandomString(length int) (string, error) {
	b := make([]byte, length)
	charsetLength := big.NewInt(int64(len(charset)))

	for i := range b {
		num, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", err
		}
		b[i] = charset[num.Int64()]
	}

	return string(b), nil
}
