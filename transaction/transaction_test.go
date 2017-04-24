package transaction

import (
	"fmt"
	"github.com/remodoy/gossip/base"
	"github.com/remodoy/gossip/log"
	"github.com/remodoy/gossip/parser"
	"github.com/remodoy/gossip/timing"
	"github.com/remodoy/gossip/transport"
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
