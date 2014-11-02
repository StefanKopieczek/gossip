package transaction

import (
	"time"

	"github.com/stefankopieczek/gossip/base"
)

var (
	global *Manager = &Manager{
		txs: map[key]Transaction{},
	}
)

type Manager struct {
	txs map[key]Transaction
}

// Transactions are identified by the branch parameter in the top Via header, and the method. (RFC 3261 17.1.3)
type key struct {
	branch string
	method string
}

func (mng *Manager) putTx(tx Transaction) {
	key := key{"", string(tx.Origin().Method)}
	mng.txs[key] = tx
}

func (mng *Manager) getTx(s base.SipMessage) (Transaction, bool) {
	key := key{"", ""}
	tx, ok := mng.txs[key]

	return tx, ok
}

func (mng *Manager) Handle(msg base.SipMessage, src, transport string) {
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
func (mng *Manager) Send(r *base.Request, dest, transport string) {
	tx := &ClientTransaction{}
	tx.origin = r
	tx.dest = dest
	tx.transport = transport

	tx.initFSM()

	tx.tu = make(chan *base.Response, 3)
	tx.tu_err = make(chan error, 1)

	tx.timer_a_time = T1
	tx.timer_a = time.NewTimer(tx.timer_a_time)
	tx.timer_b = time.NewTimer(64 * T1)

	// TODO: Send the request.

	mng.putTx(tx)
}

// Give a received response to the correct transaction.
func (mng *Manager) Correlate(r *base.Response) {
	tx, ok := mng.getTx(r)
	if !ok {
		// TODO: Something
	}

	tx.Receive(r)
}
