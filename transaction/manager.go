package transaction

import (
	"errors"
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
}

// Transactions are identified by the branch parameter in the top Via header, and the method. (RFC 3261 17.1.3)
type key struct {
	branch string
	method string
}

func NewManager(trans, addr string) (*Manager, error) {
	t, err := transport.NewManager(trans, addr)
	if err != nil {
		return nil, err
	}

	mng := &Manager{
		txs:       map[key]Transaction{},
		transport: t,
	}

	// Spin up a goroutine to pull messages up from the depths.
	go func() {
		c := mng.transport.GetChannel()
		for msg := range c {
			mng.Handle(msg)
		}
	}()

	mng.transport.Listen()

	return mng, nil
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
		log.Warn("No branch parameter on top Via header.  Transactino will be dropped.")
	}

	// TODO: Safety

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
		log.Warn("No branch parameter on top Via header.  Transactino will be dropped.")
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

func (mng *Manager) Handle(msg base.SipMessage) {
	log.Info("Received message: %s", msg.Short())
	switch m := msg.(type) {
	case *base.Request:
		// TODO: Create a new server transaction.
	case *base.Response:
		mng.Correlate(m)
	default:
		// TODO: Error
	}
}

// Create Client transaction.
func (mng *Manager) Send(r *base.Request, dest string) (<-chan *base.Response, <-chan error) {
	log.Debug("Sending to %v: %v", dest, r.String())

	tx := &ClientTransaction{}
	tx.origin = r
	tx.dest = dest
	tx.transport = mng.transport

	tx.initFSM()

	respChan := make(chan *base.Response, 3)
	errChan := make(chan error, 1)
	tx.tu = (chan<- *base.Response)(respChan)
	tx.tu_err = (chan<- error)(errChan)

	tx.timer_a_time = T1
	tx.timer_a = time.NewTimer(tx.timer_a_time)
	tx.timer_b = time.NewTimer(64 * T1)

	err := mng.transport.Send(dest, r)
	if err != nil {
		log.Warn("Failed to send message: %s", err.Error())
		tx.fsm.Spin(client_input_transport_err)
	}

	mng.putTx(tx)

	return (<-chan *base.Response)(respChan), (<-chan error)(errChan)
}

// Give a received response to the correct transaction.
func (mng *Manager) Correlate(r *base.Response) {
	tx, ok := mng.getTx(r)
	if !ok {
		// TODO: Something
	}

	tx.Receive(r)
}
