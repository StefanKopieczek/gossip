package transport

import (
	"time"

	"github.com/stefankopieczek/gossip/log"
)

// Fields of connTable should only be modified by the dedicated goroutine called by Init().
// All other callers should use connTable's associated public methods to access it.
type connTable struct {
	conns   map[string]*connWatcher
	stopped bool
}

type connWatcher struct {
	addr   string
	conn   *connection
	timer  *time.Timer
	update chan *connection
	stop   chan bool
}

// Create a new connection table.
func (t *connTable) Init() {
	log.Info("Init conntable %p")
	t.conns = make(map[string]*connWatcher)
}

// Push a connection to the connection table, registered under a specific address.
// If it is a new connection, start the socket expiry timer.
// If it is a known connection, restart the timer.
func (t *connTable) Notify(addr string, conn *connection) {
	if t.stopped {
		log.Debug("Ignoring conn notification for address %s after table stop.", addr)
		return
	}

	watcher, ok := t.conns[addr]
	if !ok {
		log.Debug("No connection watcher registered for %s; spawn one", addr)
		watcher = &connWatcher{addr, conn, time.NewTimer(c_SOCKET_EXPIRY), make(chan *connection), make(chan bool)}
		t.conns[addr] = watcher
		go func(watcher *connWatcher) {
			// We expect to close off connections explicitly, but let's be safe and clean up
			// if we close unexpectedly.
			defer func(c *connection) {
				if c != nil {
					c.Close()
				}
			}(watcher.conn)

			for {
				select {
				case <-watcher.timer.C:
					// Socket expiry timer has run out. Close the connection.
					log.Debug("Socket %p (%s) inactive for too long; close it", watcher.conn, watcher.addr)
					watcher.conn.Close()
					watcher.conn = nil
				case update := <-watcher.update:
					// We've been pinged with a connection; update it and refresh the
					// timer.
					if update != watcher.conn {
						log.Debug("Manager for address %s received new socket %p; update records", watcher.addr, watcher.conn)
						watcher.conn = update
					}
					watcher.timer.Reset(c_SOCKET_EXPIRY)
				case stop := <-watcher.stop:
					// We've received a termination signal; stop managing this connection.
					if stop {
						log.Info("Connection watcher for address %s got the kill signal. Stopping.", watcher.addr)
						watcher.timer.Stop()
						watcher.conn.Close()
						watcher.conn = nil
						break
					}
				}
			}
		}(watcher)
	}

	watcher.update <- conn
}

// Return an existing open socket for the given address, or nil if no such socket
// exists.
func (t *connTable) GetConn(addr string) *connection {
	watcher, ok := t.conns[addr]
	if ok {
		log.Debug("Query connection for address %s returns %p", addr, watcher.conn)
		return watcher.conn
	} else {
		log.Debug("Query connection for address %s returns nil (no registered watcher)", addr)
		return nil
	}
}

// Close all sockets and stop socket management.
// The table cannot be restarted after Stop() has been called, and GetConn() will return nil.
func (t *connTable) Stop() {
	log.Info("Conntable %p stopped")
	t.stopped = true
	for _, watcher := range t.conns {
		watcher.stop <- true
	}
}
