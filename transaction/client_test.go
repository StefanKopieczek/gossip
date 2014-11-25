package transaction

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
	"github.com/stefankopieczek/gossip/parser"
	"github.com/stefankopieczek/gossip/transport"
)

var c_SERVER string = "localhost:5060"
var c_CLIENT string = "localhost:5061"

var testLog *log.Logger = log.New(os.Stderr, ">>> ", 0)

var c_INVITE *base.Request
var c_OK *base.Response

func TestAAAASetup(t *testing.T) {
	log.SetDefaultLogLevel(log.WARN)
	testLog.Level = log.INFO
	var err error

	c_INVITE, err = request([]string{
		"INVITE sip:joe@bloggs.com SIP/2.0",
		"CSeq: 1 INVITE",
		"Via: SIP/2.0/UDP " + c_CLIENT + ";branch=z9hG4bK776asdhds",
		"",
		"",
	})
	if err != nil {
		panic("Unexpected error in setup: " + err.Error())
	}

	c_OK, err = response([]string{
		"SIP/2.0 200 OK",
		"CSeq: 1 INVITE",
		"Via: SIP/2.0/UDP " + c_SERVER + ";branch=z9hG4bK776asdhds",
		"",
		"",
	})
	if err != nil {
		panic("Unexpected error in setup: " + err.Error())
	}
}

func TestPortInUse(t *testing.T) {
	_, err := NewManager("udp", c_CLIENT)
	// defer client1.Stop()
	assertNoError(t, err)

	_, err = NewManager("udp", c_CLIENT)
	// defer client2.Stop
	if err == nil {
		t.Fatalf("Unexpected success when port was in use.")
	}
}

func TestSendInviteUDP(t *testing.T) {
	test := transactionTest{actions: []action{
		&clientSend{c_INVITE},
		&serverRecv{c_INVITE},
	}}
	test.Execute(t)
}

func TestReceiveOKUDP(t *testing.T) {
	test := transactionTest{actions: []action{
		&clientSend{c_INVITE},
		&serverRecv{c_INVITE},
		&serverSend{c_OK},
		&clientRecv{c_OK},
	}}
	test.Execute(t)
}

func TestResendInviteUDP(t *testing.T) {
	test := transactionTest{actions: []action{
		&clientSend{c_INVITE},
		&serverRecv{c_INVITE},
		&serverTimedRecv{c_INVITE, time.Millisecond * 300, time.Millisecond * 600},
		&serverTimedRecv{c_INVITE, time.Millisecond * 700, time.Millisecond * 1100},
		&serverTimedRecv{c_INVITE, time.Millisecond * 1500, time.Millisecond * 2100},
		&serverTimedRecv{c_INVITE, time.Millisecond * 3000, time.Millisecond * 4100},
		&serverTimedRecv{c_INVITE, time.Millisecond * 6000, time.Millisecond * 8100},
		&serverTimedRecv{c_INVITE, time.Millisecond * 12000, time.Millisecond * 16100},
	}}
	test.Execute(t)
}

type action interface {
	Act(test *transactionTest) error
}

type transactionTest struct {
	actions    []action
	client     *Manager
	server     *transport.Manager
	serverRecv chan base.SipMessage
	lastTx     *ClientTransaction
}

func (test *transactionTest) Execute(t *testing.T) {
	var err error
	test.client, err = NewManager("udp", c_CLIENT)
	assertNoError(t, err)
	defer test.client.Stop()

	test.server, err = transport.NewManager("udp")
	assertNoError(t, err)
	defer test.server.Stop()
	test.serverRecv = test.server.GetChannel()
	test.server.Listen(c_SERVER)

	for _, actn := range test.actions {
		testLog.Debug("Performing action %v", actn)
		assertNoError(t, actn.Act(test))
	}
}

type clientSend struct {
	msg *base.Request
}

func (actn *clientSend) Act(test *transactionTest) error {
	test.lastTx = test.client.Send(actn.msg, c_SERVER)
	return nil
}

type serverSend struct {
	msg base.SipMessage
}

func (actn *serverSend) Act(test *transactionTest) error {
	return test.server.Send(c_CLIENT, actn.msg)
}

type clientRecv struct {
	expected *base.Response
}

func (actn *clientRecv) Act(test *transactionTest) error {
	responses := test.lastTx.Responses()
	select {
	case response, ok := <-responses:
		if !ok {
			return fmt.Errorf("Response channel prematurely closed")
		} else if response.String() != actn.expected.String() {
			return fmt.Errorf("Unexpected response:\n%s", response.String())
		} else {
			return nil
		}
	case <-time.After(time.Second):
		return fmt.Errorf("Timed out waiting for response")
	}
}

type serverRecv struct {
	expected base.SipMessage
}

func (actn *serverRecv) Act(test *transactionTest) error {
	select {
	case msg, ok := <-test.serverRecv:
		if !ok {
			return fmt.Errorf("Server receive channel prematurely closed")
		} else if msg.String() != actn.expected.String() {
			return fmt.Errorf("Unexpected message arrived at server:\n%s", msg.String())
		} else {
			return nil
		}
	case <-time.After(time.Second):
		return fmt.Errorf("Timed out waiting for message %s on server", actn.expected.Short())
	}
}

type serverTimedRecv struct {
	expected       base.SipMessage
	deadPeriod     time.Duration
	expectedPeriod time.Duration
}

func (actn *serverTimedRecv) Act(test *transactionTest) error {
	// First wait for a duration of deadPeriod; we expect no messages in this time.
	select {
	case msg, ok := <-test.serverRecv:
		if !ok {
			return fmt.Errorf("Server receive channel prematurely closed")
		} else {
			return fmt.Errorf("Message %s arrived unexpectedly early", msg.Short())
		}
	case <-time.After(actn.deadPeriod):
		break
	}

	// Now wait to receive the message, with a timeout of expectedPeriod.
	select {
	case msg, ok := <-test.serverRecv:
		if !ok {
			return fmt.Errorf("Server receive channel prematurely closed")
		} else if msg.String() != actn.expected.String() {
			return fmt.Errorf("Unexpected message arrived at server:\n%s", msg.String())
		} else {
			return nil
		}
	case <-time.After(actn.expectedPeriod):
		return fmt.Errorf("Timed out waiting for message %s on server", actn.expected.Short())
	}
}

func assert(t *testing.T, b bool, msg string) {
	if !b {
		t.Fatalf(msg)
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
