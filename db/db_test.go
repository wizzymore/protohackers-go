package db_test

import (
	"bufio"
	"math/rand"
	"net"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/wizzymore/tcp-go/db"
)

func init() {
	s, _ := db.NewDbServer()
	s.Start()
}

func TestCanSetValue(t *testing.T) {
	t.Parallel()
	conn, err := net.Dial("udp", "localhost:8081")
	if err != nil {
		t.Error("Could not dial", err)
		return
	}
	conn.Write([]byte("foo=123"))
	conn.Write([]byte("foo"))
	scanner := bufio.NewScanner(conn)

	conn.SetReadDeadline(time.Now().Add(time.Second))
	if !scanner.Scan() {
		t.Errorf("Could not read from the server")
		t.FailNow()
	}

	if slices.Compare(scanner.Bytes(), []byte("foo=123")) != 0 {
		t.Errorf("Did not set value correctly, got: %s", scanner.Bytes())
		t.FailNow()
	}
}

func TestCanReadUnsetValue(t *testing.T) {
	t.Parallel()
	conn, err := net.Dial("udp", "localhost:8081")
	if err != nil {
		t.Error("Could not dial", err)
		return
	}
	rand_num := strconv.Itoa(rand.Int())
	conn.Write([]byte(rand_num))
	scanner := bufio.NewScanner(conn)

	conn.SetReadDeadline(time.Now().Add(time.Second))
	if !scanner.Scan() {
		t.Errorf("Could not read from the server")
		t.FailNow()
	}

	if slices.Compare(scanner.Bytes(), []byte(rand_num+"=")) != 0 {
		t.Errorf("Did not set value correctly, got: %s", scanner.Bytes())
		t.FailNow()
	}
}

func TestCantChangeVersion(t *testing.T) {
	t.Parallel()
	conn, err := net.Dial("udp", "localhost:8081")
	if err != nil {
		t.Error("Could not dial", err)
		return
	}
	conn.Write([]byte("version=123"))
	conn.Write([]byte("version"))
	scanner := bufio.NewScanner(conn)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	if !scanner.Scan() {
		t.Errorf("Could not read from the server")
		t.FailNow()
	}

	version := scanner.Text()
	if version == "123" {
		t.Errorf("Did not set value correctly, got: %s", version)
		t.FailNow()
	}
}
