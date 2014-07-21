package transport

import (
    "github.com/stefankopieczek/gossip/base"
    "github.com/stefankopieczek/gossip/log"
)

import (
    "fmt"
    "net"
    "sync"
    "time"
)

const c_LISTENER_QUEUE_SIZE int = 1000

/**
  Temporary implementation of a transport layer for SIP messages using UDP.
  This will be heavily revised, and should not be relied upon.
*/

type Manager interface {
	Listen()
	Send(message base.SipMessage)
	GetChannel() chan base.SipMessage
	Stop()
}

type notifier struct {
	listeners map[listener]bool
	listenerLock sync.Mutex
}

func (n *notifier) register(l listener) {
    if n.listeners == nil {
        n.listeners = make(map[listener]bool)
    }
	n.listenerLock.Lock()
	n.listeners[l] = true
	n.listenerLock.Unlock()
}

func (n *notifier) getChannel() (l listener) {
    c := make(chan base.SipMessage, c_LISTENER_QUEUE_SIZE)
    n.register(c)
    return c
}

func (n *notifier) notifyAll(msg base.SipMessage) {
	// Dispatch the message to all registered listeners.
	// If the listener is a closed channel, remove it from the list.
	deadListeners := make([]chan base.SipMessage, 0)
	n.listenerLock.Lock()
    log.Debug(fmt.Sprintf("Notify %d listeners of message", len(n.listeners)))
	for listener := range n.listeners {
		sent := listener.notify(msg)
		if !sent {
			deadListeners = append(deadListeners, listener)
		}
	}
	for _, deadListener := range deadListeners {
        log.Debug(fmt.Sprintf("Expiring listener %#v", deadListener))
		delete(n.listeners, deadListener)
	}
	n.listenerLock.Unlock()
}

type listener chan base.SipMessage

// notify tries to send a message to the listener.
// If the underlying channel has been closed by the receiver, return 'false';
// otherwise, return true.
func (c listener) notify(message base.SipMessage) (ok bool) {
	defer func() { recover() }()
	c <- message
	return true
}

// Fields of connTable should only be modified by the dedicated goroutine called by Init().
// All other callers should use connTable's associated public methods to access it.
type connTable struct {
    conns map[string]net.Conn
    timers map[string]time.Timer
    update chan connUpdate
    expire chan string
    stop chan bool
}

func (table *connTable) Init() {
    table.conns = make(map[string]net.Conn)
    table.timers = make(map[string]time.Timer)

    go func(t *connTable) {
        // Select from timers, stop chan, and update chan.
    }(table)
}

func (table *connTable) Stop() {
    table.stop <- true
}

func (table *connTable) Update(conn net.Conn, addr string) {
    table.update <- connUpdate{conn, addr}
}

func (table *connTable) Expire(addr string) {
    table.expire <- addr
}

type connUpdate struct {
    conn net.Conn
    address string
}
