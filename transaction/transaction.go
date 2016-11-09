package transaction

import (
	"errors"
	"time"

	"github.com/discoviking/fsm"
	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
	"github.com/stefankopieczek/gossip/timing"
	"github.com/stefankopieczek/gossip/transport"
	"strings"
)

// Generic Client Transaction

const (
	T1 = 500 * time.Millisecond
	T2 = 4 * time.Second
)

type Transaction interface {
	Receive(m base.SipMessage)
	Origin() *base.Request
	Destination() string
	Transport() transport.Manager
	Delete()
}

type transaction struct {
	fsm       *fsm.FSM       // FSM which governs the behavior of this transaction.
	origin    *base.Request  // Request that started this transaction.
	lastResp  *base.Response // Most recently received message.
	dest      string         // Of the form hostname:port
	transport transport.Manager
	tm        *Manager
}

func (tx *transaction) Origin() *base.Request {
	return tx.origin
}

func (tx *transaction) Destination() string {
	return tx.dest
}

func (tx *transaction) Transport() transport.Manager {
	return tx.transport
}

func (tx *ServerTransaction) Delete() {
	tx.tm.delTx(tx)
}

func (tx *ClientTransaction) Delete() {
	log.Warn("Tx: %p, tm: %p", tx, tx.tm)
	tx.tm.delTx(tx)
}

type ClientTransaction struct {
	transaction

	tu           chan *base.Response // Channel to transaction user.
	tu_err       chan error          // Channel to report up errors to TU.
	timer_a_time time.Duration       // Current duration of timer A.
	timer_a      timing.Timer
	timer_b      timing.Timer
	timer_d_time time.Duration // Current duration of timer A.
	timer_d      timing.Timer
}

type ServerTransaction struct {
	transaction

	tu      chan *base.Response // Channel to transaction user.
	tu_err  chan error          // Channel to report up errors to TU.
	ack     chan *base.Request  // Channel we send the ACK up on.
	timer_g timing.Timer
	timer_h timing.Timer
	timer_i timing.Timer
}

func (tx *ServerTransaction) Receive(m base.SipMessage) {
	r, ok := m.(*base.Request)
	if !ok {
		log.Warn("Client transaction received request")
	}

	var input fsm.Input = fsm.NO_INPUT
	switch {
	case r.Method == tx.origin.Method:
		input = server_input_request
	case r.Method == base.ACK:
		input = server_input_ack
		tx.ack <- r
	default:
		log.Warn("Invalid message correlated to server transaction.")
	}

	tx.fsm.Spin(input)
}

func (tx *ServerTransaction) Respond(r *base.Response) {
	tx.lastResp = r

	var input fsm.Input
	switch {
	case r.StatusCode < 200:
		input = server_input_user_1xx
	case r.StatusCode < 300:
		input = server_input_user_2xx
	default:
		input = server_input_user_300_plus
	}

	tx.fsm.Spin(input)
}

func (tx *ServerTransaction) Ack() <-chan *base.Request {
	return (<-chan *base.Request)(tx.ack)
}

func (tx *ClientTransaction) Receive(m base.SipMessage) {
	r, ok := m.(*base.Response)
	if !ok {
		log.Warn("Client transaction received request")
	}

	tx.lastResp = r

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
	log.Info("Client transaction %p resending request: %v", tx, tx.origin.Short())
	err := tx.transport.Send(tx.dest, tx.origin)
	if err != nil {
		tx.fsm.Spin(client_input_transport_err)
	}
}

// Pass up the most recently received response to the TU.
func (tx *ClientTransaction) passUp() {
	log.Info("Client transaction %p passing up response: %v", tx, tx.lastResp.Short())
	tx.tu <- tx.lastResp
}

// Send an error to the TU.
func (tx *ClientTransaction) transportError() {
	log.Info("Client transaction %p had a transport-level error", tx)
	tx.tu_err <- errors.New("failed to send message.")
}

// Inform the TU that the transaction timed out.
func (tx *ClientTransaction) timeoutError() {
	log.Info("Client transaction %p timed out", tx)
	tx.tu_err <- errors.New("transaction timed out.")
}

// Send an automatic ACK.
func (tx *ClientTransaction) Ack() {

	// rfc3261
	// TODO: fix later
	var ackTarget base.Uri
	if len(tx.lastResp.Headers("Contact")) > 0 {
		var ackTargetHdr *base.ContactHeader
		ackTargetHdrx := tx.lastResp.Headers("Contact")[0]
		ackTargetHdr = ackTargetHdrx.(*base.ContactHeader)
		ackTarget = ackTargetHdr.Address
	} else {
		ackTarget = tx.origin.Recipient
	}

	ack := base.NewRequest(base.ACK,
		ackTarget,
		tx.origin.SipVersion,
		[]base.SipHeader{},
		"")

	// Copy headers from original request.
	// TODO: Safety

	for _, via := range tx.origin.Headers("Via") {
		xvia := via.Copy()
		ack.AddHeader(xvia)
	}

	var maxForwards base.MaxForwards = 70
	ack.AddHeader(maxForwards)


	for index := range tx.lastResp.Headers("Record-Route") {
		hdr := tx.lastResp.Headers("Record-Route")[len(tx.lastResp.Headers("Record-Route"))-1-index]
		rt := strings.SplitN(hdr.String(), ":", 2)[1]
		var route base.GenericHeader = base.GenericHeader{
			HeaderName: "Route",
			Contents: rt[1:],
		}
		ack.AddHeader(route.Copy())
	}


	base.CopyHeaders("From", tx.origin, ack)
	// Copy headers from response.
	base.CopyHeaders("To", tx.lastResp, ack)

	base.CopyHeaders("Call-Id", tx.origin, ack)

	// base.CopyHeaders("Route", tx.origin, ack)
	cseq := tx.origin.Headers("CSeq")[0].Copy()
	cseq.(*base.CSeq).MethodName = base.ACK
	ack.AddHeader(cseq)


	ack.AddHeader(base.ContentLength(0))
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
