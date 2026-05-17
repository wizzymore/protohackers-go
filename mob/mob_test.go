package mob_test

import (
	"bufio"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzymore/tcp-go/mob"
)

func TestServer(t *testing.T) {
	// Initialize protohackers mock server
	server, err := net.Listen("tcp", ":16963")
	require.NoError(t, err, "Could not create bogus server")

	// Initialize our proxy server
	proxyServer, err := mob.NewMobServer("127.0.0.1:16963")
	require.NoError(t, err, "Could not create mob server")

	go proxyServer.Start()
	defer proxyServer.Stop()

	// Connect client to proxy
	client, err := net.Dial("tcp", ":8081")
	require.NoError(t, err, "Could not connect to mob server")

	// Accept the connection from the proxy on behalf of the client
	server_conn, err := server.Accept()
	require.NoError(t, err, "Could not accept proxy connection on bogus")

	// Send welcome message, the proxy should pass this to our client
	welcomeMessage := "Welcome to budgetchat! What shall I call you?\n"

	server_reader := bufio.NewReader(server_conn)
	server_writer := bufio.NewWriter(server_conn)
	server_writer.WriteString(welcomeMessage)
	server_writer.Flush()

	client_reader := bufio.NewReader(client)
	client_writer := bufio.NewWriter(client)
	result, err := client_reader.ReadString('\n')
	require.NoError(t, err, "Could not read from server")

	assert.Equal(t, result, welcomeMessage, "Did not receive welcome message")

	// Test that all wallet ids gets replaced with the bogus value 7YWHMfk9JZe0LM0g1ZauHuiSxhI
	client_writer.WriteString("Hi alice, please send payment to 7F1u3wSD5RbOHQmupo9nx4TnhQ\n")
	client_writer.Flush()

	result, err = server_reader.ReadString('\n')
	require.NoError(t, err, "Could not read message on bogus")

	assert.Equal(t, result, "Hi alice, please send payment to 7YWHMfk9JZe0LM0g1ZauHuiSxhI\n", "Did not replace token with BOGUS token")
	assert.True(t, !strings.Contains(result, "7F1u3wSD5RbOHQmupo9nx4TnhQ"), "Should not contain original wallet id")
}
