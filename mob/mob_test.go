package mob_test

import (
	"bufio"
	"net"
	"testing"

	"github.com/wizzymore/tcp-go/mob"
)

func TestServer(t *testing.T) {
	// Initialize protohackers mock server
	server, err := net.Listen("tcp", ":16963")
	if err != nil {
		t.Fatal("Could not create bogus server", err)
	}

	// Initialize our proxy server
	proxyServer, err := mob.NewMobServer("127.0.0.1:16963")
	if err != nil {
		t.Fatal("Could not create mob server", err)
	}

	go proxyServer.Start()
	defer proxyServer.Stop()

	// Connect client to proxy
	client, err := net.Dial("tcp", ":8081")
	if err != nil {
		t.Fatal("Could not connect to mob server", err)
	}

	// Accept the connection from the proxy on behalf of the client
	server_conn, err := server.Accept()
	if err != nil {
		t.Fatal("Could not accept proxy connection on bogus", err)
	}

	// Send welcome message, the proxy should pass this to our client
	welcomeMessage := "Welcome to budgetchat! What shall I call you?\n"

	server_reader := bufio.NewReader(server_conn)
	server_writer := bufio.NewWriter(server_conn)
	server_writer.WriteString(welcomeMessage)
	server_writer.Flush()

	client_reader := bufio.NewReader(client)
	client_writer := bufio.NewWriter(client)
	result, err := client_reader.ReadString('\n')
	if err != nil {
		t.Fatal("Could not read from mob server", err)
	}

	if result != welcomeMessage {
		t.Fatal("Did not received welcome message, got: ", result)
	}

	// Test that all wallet ids gets replaced with the bogus value 7YWHMfk9JZe0LM0g1ZauHuiSxhI
	client_writer.WriteString("Hi alice, please send payment to 7F1u3wSD5RbOHQmupo9nx4TnhQ\n")
	client_writer.Flush()

	result, err = server_reader.ReadString('\n')
	if err != nil {
		t.Fatal("Could not read message on bogus", err)
	}

	if result != "Hi alice, please send payment to 7YWHMfk9JZe0LM0g1ZauHuiSxhI\n" {
		t.Fatal("Did not replace token with BOGUS token, got:", result)
	}
}
