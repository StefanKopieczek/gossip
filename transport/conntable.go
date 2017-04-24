package transport

import (
	"time"

	"github.com/remodoy/gossip/log"
	"github.com/remodoy/gossip/timing"
)

// Fields of connTable should only be modified by the dedicated goroutine called by Init().
// All other callers should use connTable's associated public methods to access it.
type connTable struct {
	conns        map[string]*connWatcher
	connRequests chan *connRequest
	updates      chan *connUpdate
	expiries     chan string
	stop         chan bool
	stopped      bool
}

type connWatcher struct {
	addr       string
	conn       *connection
	timer      timing.Timer
	expiryTime time.Time
	expiry     chan<- string
	stop       chan bool
}

// Create a new connection table.
func (t *connTable) Init() {
	log.Info("Init conntable %p", t)
	t.conns = make(map[string]*connWatcher)
	t.connRequests = make(chan *connRequest)
	t.updates = make(chan *connUpdate)
	t.expiries = make(chan string)
	t.stop = make(chan bool)
	go t.manage()
}

// Management loop for the connTable.
// Handles notifications of connection updates, expiries of connections, and
// the termination of the routine.
func (t *connTable) manage() {
	for {
		select {
		case request := <-t.connRequests:
			watcher := t.conns[request.addr]
			if watcher != nil {
				request.responseChan <- watcher.conn
			} else {
				request.responseChan <- nil
			}
		case update := <-t.updates:
			t.handleUpdate(update)
		case addr := <-t.expiries:
			if t.conns[addr].expiryTime.Before(time.Now()) {
				log.Debug("Conntable %p notified that the watcher for address %s has expired. Remove it.", t, addr)
				t.conns[addr].stop <- true
				t.conns[addr].conn.Close()
				delete(t.conns, addr)
			} else {
                // Due to a race condition, the socket has been updated since this expiry happened.
                // Ignore the expiry since we already have a new socket for this address.
                log.Warn("Ignored spurious expiry for address %s in conntable %p", t, addr)
            }
		case <-t.stop:
			log.Info("Conntable %p stopped")
			t.stopped = true
			for _, watcher := range t.conns {
				watcher.stop <- true
				watcher.conn.Close()
			}
			break
		}
	}
}

// Push a connection to the connection table, registered under a specific address.
// If it is a new connection, start the socket expiry timer.
// If it is a known connection, restart the timer.
func (t *connTable) Notify(addr string, conn *connection) {
	if t.stopped {
		log.Debug("Ignoring conn notification for address %s after table stop.", addr)
		return
	}

	t.updates <- &connUpdate{addr, conn}
}

func (t *connTable) handleUpdate(update *connUpdate) {
	log.Debug("Update received in connTable %p for address %s", t, update.addr)
	watcher, entry_exists := t.conns[update.addr]
	if !entry_exists {
		log.Debug("No connection watcher registered for %s; spawn one", update.addr)
		watcher = &connWatcher{update.addr, update.conn, timing.NewTimer(c_SOCKET_EXPIRY), timing.Now().Add(c_SOCKET_EXPIRY), t.expiries, make(chan bool)}
		t.conns[update.addr] = watcher
		go watcher.loop()
	}

	watcher.Update(update.conn)
}

// Return an existing open socket for the given address, or nil if no such socket
// exists.
func (t *connTable) GetConn(addr string) *connection {
	responseChan := make(chan *connection)
	t.connRequests <- &connRequest{addr, responseChan}
	conn := <-responseChan

	log.Debug("Query connection for address %s returns %p", conn)
	return conn
}

// Close all sockets and stop socket management.
// The table cannot be restarted after Stop() has been called, and GetConn() will return nil.
func (t *connTable) Stop() {
	t.stop <- true
}

// Update the connection associated with a given connWatcher, and reset the
// timeout timer.
// Must only be called from the connTable goroutine (and in particular, must
// *not* be called from the connWatcher goroutine).
func (watcher *connWatcher) Update(c *connection) {
	watcher.expiryTime = timing.Now().Add(c_SOCKET_EXPIRY)
	watcher.timer.Reset(c_SOCKET_EXPIRY)
	watcher.conn = c
}

// connWatcher main loop. Waits for the connection to expire, and notifies the connTable
// when it does.
func (watcher *connWatcher) loop() {
	// We expect to close off connections explicitly, but let's be safe and clean up
	// if we close unexpectedly.
	defer func(c *connection) {
		if c != nil {
			c.Close()
		}
	}(watcher.conn)

	for {
		select {
		case <-watcher.timer.C():
			// Socket expiry timer has run out. Close the connection.
			log.Debug("Socket %p (%s) inactive for too long; close it", watcher.conn, watcher.addr)
			watcher.expiry <- watcher.addr

		case stop := <-watcher.stop:
			// We've received a termination signal; stop managing this connection.
			if stop {
				log.Info("Connection watcher for address %s got the kill signal. Stopping.", watcher.addr)
				watcher.timer.Stop()
				break
			}
		}
	}
}

type connUpdate struct {
	addr string
	conn *connection
}

type connRequest struct {
	addr         string
	responseChan chan *connection
}
