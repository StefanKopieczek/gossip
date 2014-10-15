package transaction

import (
	"errors"
	"fmt"
	"time"

	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
	"github.com/stefankopieczek/gossip/transport"
)

var (
	global *Manager = &Manager{
		txs: map[key]Transaction{},
	}
)

type Manager struct {
	txs       map[key]Transaction
	transport *transport.Manager
	requests  chan *ServerTransaction
}

// Transactions are identified by the branch parameter in the top Via header, and the method. (RFC 3261 17.1.3)
type key struct {
	branch string
	method string
}

func NewManager(trans, addr string) (*Manager, error) {
	t, err := transport.NewManager(trans)
	if err != nil {
		return nil, err
	}

	mng := &Manager{
		txs:       map[key]Transaction{},
		transport: t,
	}

	mng.requests = make(chan *ServerTransaction, 5)

	// Spin up a goroutine to pull messages up from the depths.
	go func() {
		c := mng.transport.GetChannel()
		for msg := range c {
			mng.handle(msg)
		}
	}()

	mng.transport.Listen(addr)

	return mng, nil
}

func (mng *Manager) Requests() <-chan *ServerTransaction {
	return (<-chan *ServerTransaction)(mng.requests)
}

func (mng *Manager) putTx(tx Transaction) {
	viaHeaders := tx.Origin().Headers("Via")
	if len(viaHeaders) == 0 {
		log.Warn("No Via header on new transaction. Transaction will be dropped.")
		return
	}

	via, ok := viaHeaders[0].(*base.ViaHeader)
	if !ok {
		panic(errors.New("Headers('Via') returned non-Via header!"))
	}

	branch, ok := (*via)[0].Params["branch"]
	if !ok {
		log.Warn("No branch parameter on top Via header.  Transaction will be dropped.")
		return
	}

	key := key{*branch, string(tx.Origin().Method)}
	mng.txs[key] = tx
}

func (mng *Manager) getTx(s base.SipMessage) (Transaction, bool) {
	viaHeaders := s.Headers("Via")
	via, ok := viaHeaders[0].(*base.ViaHeader)
	if !ok {
		panic(errors.New("Headers('Via') returned non-Via header!"))
	}

	branch, ok := (*via)[0].Params["branch"]
	if !ok {
		log.Warn("No branch parameter on top Via header.  Transaction will be dropped.")
		return nil, false
	}

	var method string
	switch s := s.(type) {
	case *base.Request:
		method = string(s.Method)
	case *base.Response:
		cseq, _ := s.Headers("CSeq")[0].(*base.CSeq)
		method = string(cseq.MethodName)
	}

	key := key{*branch, method}
	tx, ok := mng.txs[key]

	return tx, ok
}

func (mng *Manager) handle(msg base.SipMessage) {
	log.Info("Received message: %s", msg.Short())
	switch m := msg.(type) {
	case *base.Request:
		mng.request(m)
	case *base.Response:
		mng.correlate(m)
	default:
		// TODO: Error
	}
}

// Create Client transaction.
func (mng *Manager) Send(r *base.Request, dest string) *ClientTransaction {
	log.Debug("Sending to %v: %v", dest, r.String())

	tx := &ClientTransaction{}
	tx.origin = r
	tx.dest = dest
	tx.transport = mng.transport

	tx.initFSM()

	tx.tu = make(chan *base.Response, 3)
	tx.tu_err = make(chan error, 1)

	tx.timer_a_time = T1
	tx.timer_a = time.AfterFunc(tx.timer_a_time, func() {
		tx.fsm.Spin(client_input_timer_a)
	})
	tx.timer_b = time.AfterFunc(64*T1, func() {
		tx.fsm.Spin(client_input_timer_b)
	})

	// Timer D is set to 32 seconds for unreliable transports, and 0 seconds otherwise.
	tx.timer_d_time = 32 * time.Second

	err := mng.transport.Send(dest, r)
	if err != nil {
		log.Warn("Failed to send message: %s", err.Error())
		tx.fsm.Spin(client_input_transport_err)
	}

	mng.putTx(tx)

	return tx
}

// Give a received response to the correct transaction.
func (mng *Manager) correlate(r *base.Response) {
	tx, ok := mng.getTx(r)
	if !ok {
		// TODO: Something
	}

	tx.Receive(r)
}

// Handle a request.
func (mng *Manager) request(r *base.Request) {
	t, ok := mng.getTx(r)
	if ok {
		t.Receive(r)
		return
	}

	// Create a new transaction
	tx := &ServerTransaction{}
	tx.origin = r

	// Use the remote address in the top Via header.  This is not correct behaviour.
	viaHeaders := tx.Origin().Headers("Via")
	if len(viaHeaders) == 0 {
		log.Warn("No Via header on new transaction. Transaction will be dropped.")
		return
	}

	via, ok := viaHeaders[0].(*base.ViaHeader)
	if !ok {
		panic(errors.New("Headers('Via') returned non-Via header!"))
	}

	if len(*via) == 0 {
		log.Warn("Via header contained no hops! Transaction will be dropped.")
		return
	}

	hop := (*via)[0]

	tx.dest = fmt.Sprintf("%v:%v", hop.Host, *hop.Port)
	tx.transport = mng.transport

	tx.initFSM()

	tx.tu = make(chan *base.Response, 3)
	tx.tu_err = make(chan error, 1)
	tx.ack = make(chan *base.Request, 1)

	// Send a 100 Trying immediately.
	// Technically we shouldn't do this if we trustthe user to do it within 200ms,
	// but I'm not sure how to handle that situation right now.

	// Pretend the user sent us a 100 to send.
	trying := base.NewResponse(
		"SIP/2.0",
		100,
		"Trying",
		[]base.SipHeader{},
		"",
	)

	base.CopyHeaders("Via", tx.origin, trying)
	base.CopyHeaders("From", tx.origin, trying)
	base.CopyHeaders("To", tx.origin, trying)
	base.CopyHeaders("Call-Id", tx.origin, trying)
	base.CopyHeaders("CSeq", tx.origin, trying)

	tx.lastResp = trying
	tx.fsm.Spin(server_input_user_1xx)

	mng.requests <- tx
}
