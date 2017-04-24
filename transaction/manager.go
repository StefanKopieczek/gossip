package transaction

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/remodoy/gossip/base"
	"github.com/remodoy/gossip/log"
	"github.com/remodoy/gossip/timing"
	"github.com/remodoy/gossip/transport"
)

var (
	global *Manager = &Manager{
		txs: map[key]Transaction{},
	}
)

type callID string

type Manager struct {
	txs       map[key]Transaction
    callIDTxs map[callID]Transaction
	transport transport.Manager
	requests  chan *ServerTransaction
	txLock    *sync.RWMutex
}

// Transactions are identified by the branch parameter in the top Via header, and the method. (RFC 3261 17.1.3)
type key struct {
	branch string
	method string
}

func NewManager(t transport.Manager, addr string) (*Manager, error) {
	mng := &Manager{
		txs:       map[key]Transaction{},
        callIDTxs: map[callID]Transaction{},
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

	err := mng.transport.Listen(addr)
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

	branch, ok := (*via)[0].Params.Get("branch")
	if !ok {
		log.Warn("No branch parameter on top Via header.  Transaction will be dropped.")
		return
	}

	var k key
	switch branch := branch.(type) {
	case base.String:
		k = key{branch.String(), string(tx.Origin().Method)}
	case base.NoString:
		log.Warn("Empty branch parameter on top Via header. Transaction will be dropped.")
		return
	default:
		log.Warn("Unexpected type of branch value on top Via header: %T", branch)
		return
	}
	mng.txLock.Lock()
	mng.txs[k] = tx
	mng.txLock.Unlock()
}

func (mng *Manager) getCallID(s base.SipMessage) (callID, bool) {
    log.Info("F00")
    callIDHeader := s.Headers("Call-Id")
    if len(callIDHeader) == 0 {
        // No call id in message
        log.Warn("No Call-Id in message.")
        return (callID)(""), false
    }
    id, ok := callIDHeader[0].(*base.CallId)
    if !ok {
        panic(errors.New("Headers('Call-Id') returned non-Call-id header!"))
    }
    log.Info("Call-id is: %s", (string)(*id))
    return callID((string)(*id)), true
}

func (mng *Manager) putCallTx(tx Transaction) {
    log.Info("putCallTx called")
    id, ok := mng.getCallID(tx.Origin())
    if !ok {
        return
    }
    mng.txLock.Lock()
    mng.callIDTxs[id] = tx
    mng.txLock.Unlock()
    log.Info("putCallTx(%s) success", (string)(id))
}

func (mng *Manager) makeKey(s base.SipMessage) (key, bool) {
	viaHeaders := s.Headers("Via")
	via, ok := viaHeaders[0].(*base.ViaHeader)
	if !ok {
		panic(errors.New("Headers('Via') returned non-Via header!"))
	}

	b, ok := (*via)[0].Params.Get("branch")
	if !ok {
		return key{}, false
	}

	branch, ok := b.(base.String)
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

	return key{branch.String(), method}, true
}

// Gets a transaction from the transaction store.
// Should only be called inside the storage handling goroutine to ensure concurrency safety.
func (mng *Manager) getTx(s base.SipMessage) (Transaction, bool) {
	key, ok := mng.makeKey(s)
    if ok {
        mng.txLock.RLock()
        tx, ok := mng.txs[key]
        mng.txLock.RUnlock()

        if ok {
            return tx, ok
        }
    }
    callkey, ok := mng.getCallID(s)
    if ok {
        log.Info("Found call-id, getting tx: %s", callkey)
        log.Info("CallIDTxs: %v", mng.callIDTxs)
        mng.txLock.RLock()
        tx, ok := mng.callIDTxs[callkey]
        mng.txLock.RUnlock()

        if ok {
            return tx, ok
        }
    }
	// TODO: Here we should initiate more intense searching as specified in RFC3261 section 17
    log.Warn("Could not correlate message to transaction by branch/method. Dropping.")
    return nil, false
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

// Deletes a transaction from the transaction store.
// Should only be called inside the storage handling goroutine to ensure concurrency safety.
func (mng *Manager) delCallTx(t Transaction) {
    log.Info("delCallTx called")

    key, ok := mng.getCallID(t.Origin())
    if !ok {
        log.Debug("Could not build lookup key for transaction. Is it missing a Call-Id parameter?")
    }

    mng.txLock.Lock()
    delete(mng.callIDTxs, key)
    mng.txLock.Unlock()
}

func (mng *Manager) handle(msg base.SipMessage) {
	log.Info("Received messagee: %s", msg.Short())
	switch m := msg.(type) {
	case *base.Request:
        log.Info("Message is request")
		mng.request(m)
	case *base.Response:
        log.Info("Message is response")
		mng.correlate(m)
	default:
        // TODO: Error
        log.Info("Unknown event.")
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
    tx.tr = make(chan *base.Request, 3)
	tx.tu_err = make(chan error, 1)

	tx.timer_a_time = T1
	tx.timer_a = timing.AfterFunc(tx.timer_a_time, func() {
		tx.fsm.Spin(client_input_timer_a)
	})
	log.Debug("Client transaction %p, timer_b set to %v!", tx, 64*T1)
	tx.timer_b = timing.AfterFunc(64*T1, func() {
		log.Debug("Client transaction %p, timer_b fired!", tx)
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
    mng.putCallTx(tx)

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

    if r.Method == base.BYE {
        log.Warn("Got BYE without context")
        // We should respond with 200 OK without ACK
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
