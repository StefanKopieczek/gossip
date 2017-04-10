package transaction

import (
	"fmt"
	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
	"github.com/stefankopieczek/gossip/parser"
	"github.com/stefankopieczek/gossip/timing"
	"github.com/stefankopieczek/gossip/transport"
	"net"
	"strings"
	"testing"
	"time"
)

// UTs for transaction layer.

// Dummy transport manager.
type dummyTransport struct {
	listenReqs chan string
	messages   chan sentMessage
	toTM       chan base.SipMessage
}

type sentMessage struct {
	addr string
	msg  base.SipMessage
}

func newDummyTransport() *dummyTransport {
	return &dummyTransport{
		listenReqs: make(chan string, 5),
		messages:   make(chan sentMessage, 5),
		toTM:       make(chan base.SipMessage, 5),
	}
}

// Implement transport.Manager interface.
func (t *dummyTransport) Listen(address string) error {
	t.listenReqs <- address
	return nil
}

func (t *dummyTransport) Send(addr string, message base.SipMessage) error {
	t.messages <- sentMessage{addr, message}
	return nil
}

func (t *dummyTransport) Stop() {}

func (t *dummyTransport) GetChannel() transport.Listener {
	return t.toTM
}

// Test infra.
type action interface {
	Act(test *transactionTest) error
}

type transactionTest struct {
	t         *testing.T
	actions   []action
	tm        *Manager
	transport *dummyTransport
	lastTx    *ClientTransaction
}

func (test *transactionTest) Execute() {
	var err error
	timing.MockMode = true
	log.SetDefaultLogLevel(log.DEBUG)
	transport := newDummyTransport()
	test.tm, err = NewManager(transport, c_CLIENT)
	assertNoError(test.t, err)
	defer test.tm.Stop()

	test.transport = transport

	for _, actn := range test.actions {
		test.t.Logf("Performing action %v", actn)
		assertNoError(test.t, actn.Act(test))
	}
}

type userSend struct {
	msg *base.Request
}

func (actn *userSend) Act(test *transactionTest) error {
	test.t.Logf("Transaction User sending message:\n%v", actn.msg.String())
	test.lastTx = test.tm.Send(actn.msg, c_SERVER)
	return nil
}

type transportSend struct {
	msg base.SipMessage
}

func (actn *transportSend) Act(test *transactionTest) error {
	test.t.Logf("Transport Layer sending message\n%v", actn.msg.String())
	test.transport.toTM <- actn.msg
	return nil
}

type userRecv struct {
	expected *base.Response
}

func (actn *userRecv) Act(test *transactionTest) error {
	responses := test.lastTx.Responses()
	select {
	case response, ok := <-responses:
		if !ok {
			return fmt.Errorf("Response channel prematurely closed")
		} else if response.String() != actn.expected.String() {
			return fmt.Errorf("Unexpected response:\n%s", response.String())
		} else {
			test.t.Logf("Transaction User received correct message\n%v", response.String())
			return nil
		}
	case <-time.After(time.Second):
		return fmt.Errorf("Timed out waiting for response")
	}
}

type transportRecv struct {
	expected base.SipMessage
}

func (actn *transportRecv) Act(test *transactionTest) error {
	select {
	case msg, ok := <-test.transport.messages:
		if !ok {
			return fmt.Errorf("Transport layer receive channel prematurely closed")
		} else if msg.msg.String() != actn.expected.String() {
			return fmt.Errorf("Unexpected message arrived at transport:\n%s", msg.msg.String())
		} else {
			test.t.Logf("Transport received correct message\n %v", msg.msg.String())
			return nil
		}
	case <-time.After(time.Second):
		return fmt.Errorf("Timed out waiting for message at transport")
	}
}

type wait struct {
	d time.Duration
}

func (actn *wait) Act(test *transactionTest) error {
	test.t.Logf("Elapsing time by %v", actn.d)
	timing.Elapse(actn.d)
	return nil
}

func assert(t *testing.T, b bool, msg string) {
	if !b {
		t.Errorf(msg)
	}
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
}

func message(rawMsg []string) (base.SipMessage, error) {
	return parser.ParseMessage([]byte(strings.Join(rawMsg, "\r\n")))
}

func request(rawMsg []string) (*base.Request, error) {
	msg, err := message(rawMsg)

	if err != nil {
		return nil, err
	}

	switch msg.(type) {
	case *base.Request:
		return msg.(*base.Request), nil
	default:
		return nil, fmt.Errorf("%s is not a request", msg.Short)
	}
}

func response(rawMsg []string) (*base.Response, error) {
	msg, err := message(rawMsg)

	if err != nil {
		return nil, err
	}

	switch msg.(type) {
	case *base.Response:
		return msg.(*base.Response), nil
	default:
		return nil, fmt.Errorf("%s is not a response", msg.Short)
	}
}

// Confirm transaction manager requests for transport to listen.
func TestListenRequest(t *testing.T) {
	trans := newDummyTransport()
	m, err := NewManager(trans, "1.1.1.1")
	if err != nil {
		t.Fatalf("Error creating TM: %v", err)
	}

	addr := <-trans.listenReqs
	if addr != "1.1.1.1" {
		t.Fatalf("Created TM with addr 1.1.1.1 but were asked to listen on %v", addr)
	}

	m.Stop()
}

func TestUDPResponseUsesSameSourcePort(t *testing.T) {
	log.SetDefaultLogLevel(log.DEBUG)
	client_hostport := "127.0.0.1:10001"
	server_hostport := "127.0.0.1:10002"

	// Resolve client UDP endpoint.
	client_addr, err := net.ResolveUDPAddr("udp", client_hostport)
	if err != nil {
		t.Fatalf("Error resolving UDP addr %s: %v", client_addr, err)
	}

	// Resolve server UDP endpoint.
	server_addr, err := net.ResolveUDPAddr("udp", server_hostport)
	if err != nil {
		t.Fatalf("Error resolving UDP addr %s: %v", server_hostport, err)
	}

	// Start the server process.
	server_transport, err := transport.NewManager("udp")
	if err != nil {
		t.Fatalf("Error creating transport: %v", err)
	}
	server, err := NewManager(server_transport, server_hostport)
	if err != nil {
		t.Fatalf("Error creating transport manager: %v", err)
	}

	// Set up server daemon to reply to incoming requests with a 200 OK.
	ok := base.NewResponse(
		"SIP/2.0",
		200,
		"SIP/2.0",
		make([]base.SipHeader, 0),
		"")
	go func() {
		log.Debug("Server started listening for invites")
		transaction := <-server.requests
		log.Debug("Server received invite")
		transaction.Respond(ok)
		log.Debug("Server responded to invite")
	}()

	// Open the client->server connection that will be used to send the initial INVITE and receive the reply.
	conn, err := net.DialUDP("udp", client_addr, server_addr)
	if err != nil {
		t.Fatalf("Error opening a server connection from %v to %v: %v", client_addr, server_addr, err)
	}

	// Start listening for the 200 OK reply from the server.
	complete := make(chan bool, 0)
	go func() {
		log.Info("Started listening for replies from the server on %v", conn.LocalAddr())
		buffer := make([]byte, 65507)
		_, laddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			t.Fatalf("Error reading from UDP conn %v: %v", conn, err)
		} else if laddr.String() != client_hostport {
			t.Errorf("Received response from port %s; was expecting %s", laddr.String(), client_hostport)
		} else {
			log.Info("Received response on correct port")
			complete <- true
		}
	}()

	// Send the test INVITE to the server.
	log.Info("Starting sleep")
	<-time.After(time.Second * 5)
	log.Info("Waking up to send packet")
	to_uri, _ := parser.ParseUri("sip:bob@example.com")
	via_port := uint16(client_addr.Port)
	via_hop := base.ViaHop{"SIP", "2.0", "udp", client_addr.IP.String(), &via_port,
		base.NewParams().Add("branch", base.String{"z9hG4bKkjshdyff"})}
	var via_header base.ViaHeader = []*base.ViaHop{&via_hop}
	invite := base.NewRequest(
		base.INVITE,
		to_uri,
		"SIP/2.0",
		[]base.SipHeader{via_header},
		"")
	_, err = conn.Write([]byte(invite.String()))

	// Wait for the listener process to confirm it's received a valid reply from the server..
	select {
	case <-complete:
		break
	case <-time.After(time.Second * 10):
		t.Errorf("Timed out waiting for response from server")
	}
}
