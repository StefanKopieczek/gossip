package transport

import "testing"
import (
	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
	"github.com/stefankopieczek/gossip/parser"
	"github.com/stefankopieczek/gossip/testutils"
	"github.com/stefankopieczek/gossip/time"
)

var c_LOG_LEVEL = log.WARN

func TestAAAASetup(t *testing.T) {
	time.MockMode = true
	log.SetDefaultLogLevel(c_LOG_LEVEL)
}

// Test that we can store and retrieve a connection.
func TestBasicStorage(t *testing.T) {
	var table connTable
	table.Init()
	defer table.Stop()

	conn := makeTestConn()
	table.Notify("foo", conn)
	sleep(time.Millisecond * 20)

	if table.GetConn("foo") != conn {
		t.FailNow()
	}
}

// Test that after the configured expiry time, a stored connection expires and
// is nulled out.
func TestBasicExpiry(t *testing.T) {
	var table connTable
	table.Init()
	// defer table.Stop()  // This currently panics due to issue #13.

	table.Notify("bar", makeTestConn())
	time.Elapse(c_SOCKET_EXPIRY)
	time.Elapse(time.Second)
	sleep(time.Millisecond * 20)

	if table.GetConn("bar") != nil {
		t.FailNow()
	}
}

// Test that two different connections can be stored at the same time.
func TestDoubleStorage(t *testing.T) {
	var table connTable
	table.Init()
	defer table.Stop()

	conn1 := makeTestConn()
	table.Notify("foo", conn1)
	conn2 := makeTestConn()
	table.Notify("bar", conn2)
	sleep(time.Millisecond * 20)

	if table.GetConn("foo") != conn1 {
		t.FailNow()
	} else if table.GetConn("bar") != conn2 {
		t.FailNow()
	}
}

// Test that we can push an update to a stored connection, and that when we
// retrieve the connection we get the updated value.
func TestUpdate(t *testing.T) {
	var table connTable
	table.Init()
	defer table.Stop()
	table.Notify("foo", makeTestConn())
	conn2 := makeTestConn()
	table.Notify("foo", conn2)
	sleep(time.Millisecond * 20)

	if table.GetConn("foo") != conn2 {
		t.FailNow()
	}
}

// Test that if a connection is allowed to expire, we can store the same connection
// again with the same key and retrieve it.
func TestReuse1(t *testing.T) {
	var table connTable
	table.Init()
	defer table.Stop() // This currently panics due to issue #13.

	conn := makeTestConn()
	table.Notify("foo", conn)
	time.Elapse(c_SOCKET_EXPIRY)
	time.Elapse(time.Second)

	table.Notify("foo", conn)
	sleep(time.Millisecond * 20)
	if table.GetConn("foo") != conn {
		t.FailNow()
	}
}

// Test that if a connection is allowed to expire, we can store a different connection
// with the same key and then retrieve it.
func TestReuse2(t *testing.T) {
	var table connTable
	table.Init()
	defer table.Stop() // This currently panics due to issue #13.

	table.Notify("foo", makeTestConn())
	time.Elapse(c_SOCKET_EXPIRY)
	time.Elapse(time.Second)

	conn2 := makeTestConn()
	table.Notify("foo", conn2)
	sleep(time.Millisecond * 20)
	if table.GetConn("foo") != conn2 {
		t.FailNow()
	}
}

// Construct a dummy connection object to use to populate the connTable for tests.
func makeTestConn() *connection {
	parsedMessages := make(chan base.SipMessage)
	errors := make(chan error)
	streamed := true
	return &connection{
		&testutils.DummyConn{},
		true,
		parser.NewParser(parsedMessages, errors, streamed),
		parsedMessages,
		errors,
		make(chan base.SipMessage),
	}
}

// Utility method to sleep for a given time.
// This is a bit clunky because we have to temporarily turn off mock mode in the time library.
func sleep(d time.Duration) {
	oldMockMode := time.MockMode
	time.MockMode = false
	<-time.NewTimer(d).C()
	time.MockMode = oldMockMode
}
