package transport

import (
	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
)

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const c_BUFSIZE int = 65507
const c_LISTENER_QUEUE_SIZE int = 1000
const c_SOCKET_EXPIRY time.Duration = time.Hour

type Manager struct {
	notifier
	transport transport
}

type transport interface {
	IsStreamed() bool
	Listen(address string) error
	Send(addr string, message base.SipMessage) error
	Stop()
}

func NewManager(transportType string) (manager *Manager, err error) {
	err = fmt.Errorf("Unknown transport type '%s'", transportType)

	var n notifier
	n.init()

	var transport transport
	switch strings.ToLower(transportType) {
	case "udp":
		transport, err = NewUdp(n.inputs)
	case "tcp":
		transport, err = NewTcp(n.inputs)
	case "tls":
		// TODO
	}

	if transport != nil && err == nil {
		manager = &Manager{notifier: n, transport: transport}
	} else {
		// Close the input chan in order to stop the notifier; this prevents
		// us leaking it.
		close(n.inputs)
	}

	return
}

func (manager *Manager) Listen(address string) error {
	return manager.transport.Listen(address)
}

func (manager *Manager) Send(addr string, message base.SipMessage) error {
	return manager.transport.Send(addr, message)
}

func (manager *Manager) Stop() {
	manager.transport.Stop()
	manager.notifier.stop()
}

type notifier struct {
	listeners    map[listener]bool
	listenerLock sync.Mutex
	inputs       chan base.SipMessage
}

func (n *notifier) init() {
	n.listeners = make(map[listener]bool)
	n.inputs = make(chan base.SipMessage)
	go n.forward()
}

func (n *notifier) register(l listener) {
	log.Debug("Notifier %p has new listener %p", n, l)
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

func (n *notifier) forward() {
	for msg := range n.inputs {
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
			close(deadListener)
			delete(n.listeners, deadListener)
		}
		n.listenerLock.Unlock()
	}
}

func (n *notifier) stop() {
	close(n.inputs)
	n.listenerLock.Lock()
	for c, _ := range n.listeners {
		close(c)
	}
	n.listeners = nil
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
