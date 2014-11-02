package transport

import (
	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
	"github.com/stefankopieczek/gossip/parser"
)

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

const c_LISTENER_QUEUE_SIZE int = 1000
const c_SOCKET_EXPIRY time.Duration = time.Hour

type Manager struct {
	notifier
	transport transport
	parser    parser.Parser
}

type transport interface {
	IsStreamed() bool
	Listen(parser parser.Parser) error
	Send(addr string, message base.SipMessage) error
	Stop()
}

func NewManager(transportType string, localAddress string) (manager *Manager, err error) {
	err = fmt.Errorf("Unknown transport type '%s'", transportType)
	var transport transport
	switch strings.ToLower(transportType) {
	case "udp":
		transport, err = NewUdp(localAddress)
	case "tcp":
		transport, err = NewTcp(localAddress)
	case "tls":
		// TODO
	}

	if transport != nil && err != nil {
		manager = &Manager{transport: transport}
	}

	return
}

func (manager *Manager) Listen() error {
	parsedMessages := make(chan base.SipMessage, 0)
	errors := make(chan error, 0)
	manager.parser = parser.NewParser(parsedMessages, errors, manager.transport.IsStreamed())

	go func() {
		running := true
		for running {
			select {
			case message, ok := <-parsedMessages:
				if ok {
					manager.notifyAll(message)
				} else {
					log.Info("Parser stopped in Transport Manager; will stop listening")
					running = false
				}
			case err, ok := <-errors:
				if ok {
					// The parser has hit a terminal error. We need to restart it.
					log.Warn("Failed to parse SIP message: %s", err.Error())
					manager.parser = parser.NewParser(parsedMessages, errors, manager.transport.IsStreamed())
				} else {
					log.Info("Parser stopped in Transport Manager; will stop listening")
				}
			}
		}
	}()

	err := manager.transport.Listen(manager.parser)
	if err != nil {
		// TODO - Stop parser
		manager.parser = nil
	}

	return err
}

func (manager *Manager) Send(addr string, message base.SipMessage) error {
	return manager.transport.Send(addr, message)
}

func (manager *Manager) Stop() {
	// Stop parser - TODO
	manager.transport.Stop()
}

type notifier struct {
	listeners    map[listener]bool
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

func (n *notifier) GetChannel() (l listener) {
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
	conns   map[string]*connManager
	stopped bool
}

type connManager struct {
	addr   string
	conn   net.Conn
	timer  *time.Timer
	update chan net.Conn
	stop   chan bool
}

// Create a new connection table.
func (t *connTable) Init() {
	t.conns = make(map[string]*connManager)
}

// Push a connection to the connection table, registered under a specific address.
// If it is a new connection, start the socket expiry timer.
// If it is a known connection, restart the timer.
func (t *connTable) Notify(addr string, conn net.Conn) {
	if t.stopped {
		log.Debug("Ignoring conn notification for address %s after table stop.")
		return
	}

	manager, ok := t.conns[addr]
	if !ok {
		log.Debug("No connection manager registered for %s; spawn one", addr)
		manager = &connManager{addr, conn, &time.Timer{}, make(chan net.Conn), make(chan bool)}
		t.conns[addr] = manager
		go func(mgr *connManager) {
			// We expect to close off connections explicitly, but let's be safe and clean up
			// if we close unexpectedly.
			defer func(c net.Conn) {
				if c != nil {
					c.Close()
				}
			}(mgr.conn)

			for {
				select {
				case <-mgr.timer.C:
					// Socket expiry timer has run out. Close the connection.
					log.Debug("Socket %v (%s) inactive for too long; close it", mgr.conn, mgr.addr)
					mgr.conn.Close()
					mgr.conn = nil
				case update := <-mgr.update:
					// We've been pinged with a connection; update it and refresh the
					// timer.
					log.Debug("Manager for address %s received new socket %v; update records", mgr.addr, mgr.conn)
					mgr.conn = update
					mgr.timer.Stop()
					mgr.timer = time.NewTimer(c_SOCKET_EXPIRY)
				case stop := <-mgr.stop:
					// We've received a termination signal; stop managing this connection.
					if stop {
						log.Debug("Connection manager for address %s got the kill signal. Stopping.", mgr.addr)
						mgr.timer.Stop()
						mgr.conn.Close()
						mgr.conn = nil
						break
					}
				}
			}
		}(manager)
	}

	log.Debug("Push new socket %v to manager for address %s", conn, addr)
	manager.update <- conn
}

// Return an existing open socket for the given address, or nil if no such socket
// exists.
func (t *connTable) GetConn(addr string) net.Conn {
	manager, ok := t.conns[addr]
	if ok {
		log.Debug("Query connection for address %s returns %v", addr, manager.conn)
		return manager.conn
	} else {
		log.Debug("Query connection for address %s returns nil (no registered manager)", addr)
		return nil
	}
}

// Close all sockets and stop socket management.
// The table cannot be restarted after Stop() has been called, and GetConn() will return nil.
func (t *connTable) Stop() {
	t.stopped = true
	for _, manager := range t.conns {
		manager.stop <- true
	}
}
