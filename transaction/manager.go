package transaction

import (
	"errors"
	"fmt"
	"sync"
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
	txLock    *sync.RWMutex
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
		txLock:    &sync.RWMutex{},
		transport: t,
	}

	mng.requests = make(chan *ServerTransaction, 5)

	// Spin up a goroutine to pull messages up from the depths.
	c := mng.transport.GetChannel()
	go func() {
		for msg := range c {
			go mng.handle(msg)
		}
	}()

	err = mng.transport.Listen(addr)
	if err != nil {
		return nil, err
	}

	return mng, nil
}

// Stop the manager and close down all processing on it, losing all transactions in progress.
func (mng *Manager) Stop() {
	// Stop the transport layer.
	mng.transport.Stop()
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
		// TODO: Handle this better.
		panic(errors.New("Headers('Via') returned non-Via header!"))
	}

	branch, ok := (*via)[0].Params["branch"]
	if !ok {
		log.Warn("No branch parameter on top Via header.  Transaction will be dropped.")
		return
	}

	key := key{*branch, string(tx.Origin().Method)}
	mng.txLock.Lock()
	mng.txs[key] = tx
	mng.txLock.Unlock()
}

func (mng *Manager) makeKey(s base.SipMessage) (key, bool) {
	viaHeaders := s.Headers("Via")
	via, ok := viaHeaders[0].(*base.ViaHeader)
	if !ok {
		panic(errors.New("Headers('Via') returned non-Via header!"))
	}

	branch, ok := (*via)[0].Params["branch"]
	if !ok {
		return key{}, false
	}

	var method string
	switch s := s.(type) {
	case *base.Request:
		// Correlate an ACK request to the related INVITE.
		if s.Method == base.ACK {
			method = string(base.INVITE)
		} else {
			method = string(s.Method)
		}
	case *base.Response:
		cseqs := s.Headers("CSeq")
		if len(cseqs) == 0 {
			// TODO - Handle non-existent CSeq
			panic("No CSeq on response!")
		}

		cseq, _ := s.Headers("CSeq")[0].(*base.CSeq)
		method = string(cseq.MethodName)
	}

	return key{*branch, method}, true
}

// Gets a transaction from the transaction store.
// Should only be called inside the storage handling goroutine to ensure concurrency safety.
func (mng *Manager) getTx(s base.SipMessage) (Transaction, bool) {
	key, ok := mng.makeKey(s)
	if !ok {
		// TODO: Here we should initiate more intense searching as specified in RFC3261 section 17
		log.Warn("Could not correlate message to transaction by branch/method. Dropping.")
		return nil, false
	}

	mng.txLock.RLock()
	tx, ok := mng.txs[key]
	mng.txLock.RUnlock()

	return tx, ok
}

// Deletes a transaction from the transaction store.
// Should only be called inside the storage handling goroutine to ensure concurrency safety.
func (mng *Manager) delTx(t Transaction) {
	key, ok := mng.makeKey(t.Origin())
	if !ok {
		log.Debug("Could not build lookup key for transaction. Is it missing a branch parameter?")
	}

	mng.txLock.Lock()
	delete(mng.txs, key)
	mng.txLock.Unlock()
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
	tx.tm = mng

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
		log.Warn("Failed to correlate response to active transaction. Dropping it.")
		return
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

	// If we failed to correlate an ACK, just drop it.
	if r.Method == base.ACK {
		log.Warn("Couldn't correlate ACK to an open transaction. Dropping it.")
		return
	}

	// Create a new transaction
	tx := &ServerTransaction{}
	tx.tm = mng
	tx.origin = r
	tx.transport = mng.transport

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

	port := uint16(5060)

	if hop.Port != nil {
		port = *hop.Port
	}

	tx.dest = fmt.Sprintf("%s:%d", hop.Host, port)
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
