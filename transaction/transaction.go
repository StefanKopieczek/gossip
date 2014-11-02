package transaction

import (
	"errors"
	"time"

	"github.com/discoviking/fsm"
	"github.com/stefankopieczek/gossip/base"
)

// Generic Client Transaction

const (
	T1 = 500 * time.Millisecond
)

type Transaction interface {
	Receive(m base.SipMessage)
	Origin() *base.Request
	Destination() string
	Transport() string
}

type transaction struct {
	fsm       *fsm.FSM       // FSM which governs the behavior of this transaction.
	origin    *base.Request  // Request that started this transaction.
	lastIn    *base.Response // Most recently received message.
	dest      string         // Of the form hostname:port
	transport string         // "udp", "tcp", "tls"
}

func (tx *transaction) Origin() *base.Request {
	return tx.origin
}

func (tx *transaction) Destination() string {
	return tx.dest
}

func (tx *transaction) Transport() string {
	return tx.transport
}

type ClientTransaction struct {
	transaction

	tu           chan<- *base.Response // Channel to transaction user.
	tu_err       chan<- error          // Channel to report up errors to TU.
	timer_a_time time.Duration         // Current duration of timer A.
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
	// TODO: Send the request.
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
func (tx *ClientTransaction) sendAck() {
	// TODO: Send an ACK.
}
