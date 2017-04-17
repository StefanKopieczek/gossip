package transport

import (
	"os"
	"testing"
	"time"
)

import (
	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
	"github.com/stefankopieczek/gossip/parser"
	"github.com/stefankopieczek/gossip/testutils"
	"github.com/stefankopieczek/gossip/timing"
)

var c_LOG_LEVEL = log.WARN

func TestMain(m *testing.M) {
	timing.MockMode = true
	log.SetDefaultLogLevel(c_LOG_LEVEL)
	os.Exit(m.Run())
}

// Test that we can store and retrieve a connection.
func TestBasicStorage(t *testing.T) {
	var table connTable
	table.Init()
	defer table.Stop()

	conn := makeTestConn()
	table.Notify("foo", conn)

	// Ensure it has been created.
	if !testutils.Eventually(func() bool { return table.GetConn("foo") == conn }) {
		t.FailNow()
	}
}

// Test that after the configured expiry time, a stored connection expires and
// is nulled out.
func TestBasicExpiry(t *testing.T) {
	var table connTable
	table.Init()
	defer table.Stop() // This currently panics due to issue #13.

	table.Notify("bar", makeTestConn())

	// Ensure it has been created.
	if !testutils.Eventually(func() bool { return table.GetConn("bar") != nil }) {
		t.FailNow()
	}

	timing.Elapse(c_SOCKET_EXPIRY)
	timing.Elapse(time.Nanosecond)

	// Ensure it has been expired.
	if !testutils.Eventually(func() bool { return table.GetConn("bar") == nil }) {
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

	// Ensure foo has been created.
	if !testutils.Eventually(func() bool { return table.GetConn("foo") == conn1 }) {
		t.FailNow()
	}

	// Ensure bar has been created.
	if !testutils.Eventually(func() bool { return table.GetConn("bar") == conn2 }) {
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

	if !testutils.Eventually(func() bool { return table.GetConn("foo") == conn2 }) {
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

	// Wait for connection to definitely exist.
	if !testutils.Eventually(func() bool { return table.GetConn("foo") != nil }) {
		t.Log("Connection foo couldn't be retrieved")
		t.FailNow()
	}

	timing.Elapse(c_SOCKET_EXPIRY)
	timing.Elapse(time.Nanosecond)

	// Wait for connection to definitely expire.
	if !testutils.Eventually(func() bool { return table.GetConn("foo") == nil }) {
		t.Log("Connection foo was never nil")
		t.FailNow()
	}

	// Re-store and retrieve.
	table.Notify("foo", conn)
	if table.GetConn("foo") != conn {
		t.Log("Re-stored connection wasn't the one we put in")
		t.FailNow()
	}
}

// Test that if a connection is allowed to expire, we can store a different connection
// with the same key and then retrieve it.
func TestReuse2(t *testing.T) {
	var table connTable
	table.Init()
	defer table.Stop()

	table.Notify("foo", makeTestConn())

	// Wait for connection to definitely exist.
	if !testutils.Eventually(func() bool { return table.GetConn("foo") != nil }) {
		t.Log("Connection foo couldn't be retrieved")
		t.FailNow()
	}

	timing.Elapse(c_SOCKET_EXPIRY)
	timing.Elapse(time.Nanosecond)

	// Wait for connection to definitely expire.
	if !testutils.Eventually(func() bool { return table.GetConn("foo") == nil }) {
		t.Log("Connection foo was never nil")
		t.FailNow()
	}

	conn2 := makeTestConn()
	table.Notify("foo", conn2)
	if table.GetConn("foo") != conn2 {
		t.Log("Re-stored connection wasn't the one we put in")
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
