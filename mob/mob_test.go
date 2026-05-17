package mob_test

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/wizzymore/tcp-go/mob"
)

func TestServer(t *testing.T) {
	bogusServer, err := net.Listen("tcp", ":16963")
	if err != nil {
		t.Fatal("Could not create bogus server", err)
	}

	proxyServer, err := mob.NewMobServer("127.0.0.1:16963")
	if err != nil {
		t.Fatal("Could not create mob server", err)
	}

	go proxyServer.Start()
	defer proxyServer.Stop()

	conn, err := net.Dial("tcp", ":8081")
	if err != nil {
		t.Fatal("Could not connect to mob server", err)
	}

	t.Log("Waiting for connection on bogus server")
	bConn, err := bogusServer.Accept()
	if err != nil {
		t.Fatal("Could not accept proxy connection on bogus", err)
	}
	t.Log("Received connection on bogus server")
	bWriter := bufio.NewWriter(bConn)
	welcomeMessage := "Welcome to budgetchat! What shall I call you?\n"
	bWriter.WriteString(welcomeMessage)
	bWriter.Flush()

	cReader := bufio.NewReader(conn)
	conn.SetReadDeadline(time.Now().Add(time.Millisecond * 200))
	result, err := cReader.ReadString('\n')
	if err != nil {
		t.Fatal("Could not read from mob server", err)
	}

	if result != welcomeMessage {
		t.Fatal("Did not received welcome message, got: ", result)
	}

	cWriter := bufio.NewWriter(conn)
	cWriter.WriteString("Hi alice, please send payment to 7F1u3wSD5RbOHQmupo9nx4TnhQ\n")
	cWriter.Flush()

	bReader := bufio.NewReader(bConn)
	result, err = bReader.ReadString('\n')
	if err != nil {
		t.Fatal("Could not read message on bogus", err)
	}

	if result != "Hi alice, please send payment to 7YWHMfk9JZe0LM0g1ZauHuiSxhI\n" {
		t.Fatal("Did not replace token with BOGUS token, got:", result)
	}
}
