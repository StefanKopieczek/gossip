package transaction

import (
	"errors"
	"time"

	"github.com/discoviking/fsm"
	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/transport"
)

// Generic Client Transaction

const (
	T1 = 500 * time.Millisecond
)

type Transaction interface {
	Receive(m base.SipMessage)
	Origin() *base.Request
	Destination() string
	Transport() *transport.Manager
}

type transaction struct {
	fsm       *fsm.FSM       // FSM which governs the behavior of this transaction.
	origin    *base.Request  // Request that started this transaction.
	lastIn    *base.Response // Most recently received message.
	dest      string         // Of the form hostname:port
	transport *transport.Manager
}

func (tx *transaction) Origin() *base.Request {
	return tx.origin
}

func (tx *transaction) Destination() string {
	return tx.dest
}

func (tx *transaction) Transport() *transport.Manager {
	return tx.transport
}

type ClientTransaction struct {
	transaction

	tu           chan *base.Response // Channel to transaction user.
	tu_err       chan error          // Channel to report up errors to TU.
	timer_a_time time.Duration       // Current duration of timer A.
	timer_a      *time.Timer
	timer_b      *time.Timer
	timer_d      *time.Timer
}

func (tx *ClientTransaction) Receive(m base.SipMessage) {
	r, ok := m.(*base.Response)
	if !ok {
		// TODO: Log Error.
	}

	tx.lastIn = r

	var input fsm.Input
	switch {
	case r.StatusCode < 200:
		input = client_input_1xx
	case r.StatusCode < 300:
		input = client_input_2xx
	default:
		input = client_input_300_plus
	}

	tx.fsm.Spin(input)
}

// Resend the originating request.
func (tx *ClientTransaction) resend() {
	err := tx.transport.Send(tx.dest, tx.origin)
	if err != nil {
		tx.fsm.Spin(client_input_transport_err)
	}
}

// Pass up the most recently received response to the TU.
func (tx *ClientTransaction) passUp() {
	tx.tu <- tx.lastIn
}

// Send an error to the TU.
func (tx *ClientTransaction) transportError() {
	tx.tu_err <- errors.New("failed to send message.")
}

// Inform the TU that the transaction timed out.
func (tx *ClientTransaction) timeoutError() {
	tx.tu_err <- errors.New("transaction timed out.")
}

// Send an automatic ACK.
func (tx *ClientTransaction) Ack() {
	ack := base.NewRequest(base.ACK,
		tx.origin.Recipient,
		tx.origin.SipVersion,
		[]base.SipHeader{},
		"")

	// Copy headers from original request.
	// TODO: Safety
	base.CopyHeaders("From", tx.origin, ack)
	base.CopyHeaders("Call-Id", tx.origin, ack)
	base.CopyHeaders("Route", tx.origin, ack)
	cseq := tx.origin.Headers("CSeq")[0].Copy()
	cseq.(*base.CSeq).MethodName = base.ACK
	ack.AddHeader(cseq)
	via := tx.origin.Headers("Via")[0].Copy()
	ack.AddHeader(via)

	// Copy headers from response.
	base.CopyHeaders("To", tx.lastIn, ack)

	// Send the ACK.
	tx.transport.Send(tx.dest, ack)
}

// Return the channel we send responses on.
func (tx *ClientTransaction) Responses() <-chan *base.Response {
	return (<-chan *base.Response)(tx.tu)
}

// Return the channel we send errors on.
func (tx *ClientTransaction) Errors() <-chan error {
	return (<-chan error)(tx.tu_err)
}
