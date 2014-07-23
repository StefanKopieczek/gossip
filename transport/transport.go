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
const c_SOCKET_EXPIRY time.Duration = time.Hour

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
    conns map[string]*connManager
    stopped bool
}

type connManager struct {
    conn net.Conn
    timer *time.Timer
    update chan net.Conn
    stop chan bool
}

func (t *connTable) Init() {
    t.conns = make(map[string]*connManager)
}

func (t *connTable) Notify(addr string, conn net.Conn) {
    if t.stopped {
        log.Debug("Ignoring conn notification for address %s after table stop.")
        return
    }

    manager, ok := t.conns[addr]
    if ok {
        manager.update <- conn
    } else {
        manager := connManager{conn, &time.Timer{}, make(chan net.Conn), make(chan bool)}
        go func(mgr *connManager) {
            // We expect to close off connections explicitly, but let's be safe and clean up
            // if we close unexpectedly.
            defer func(c net.Conn) {
                if c != nil {
                    c.Close()
                }
            } (mgr.conn)

            for {
                select {
                case <- mgr.timer.C:
                    // Socket expiry timer has run out. Close the connection.
                    mgr.conn.Close()
                    mgr.conn = nil
                case update := <- mgr.update:
                    // We've been pinged with a connection; update it and refresh the
                    // timer.
                    mgr.conn = update
                    mgr.timer.Stop()
                    mgr.timer = time.NewTimer(c_SOCKET_EXPIRY)
                case stop := <- mgr.stop:
                    // We've received a termination signal; stop managing this connection.
                    if stop {
                        mgr.timer.Stop()
                        mgr.conn.Close()
                        mgr.conn = nil
                        break
                    }
                }
            }
        }(&manager)
    }
}

func (t *connTable) GetConn(addr string) net.Conn {
    manager, ok := t.conns[addr]
    if ok {
        return manager.conn
    } else {
        return nil
    }
}
