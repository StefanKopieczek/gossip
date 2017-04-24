package test

import (
	"github.com/remodoy/gossip/transaction"
	"github.com/remodoy/gossip/transport"
	"testing"
)

// Tests we can start/stop a transaction manager repeatedly on the same port.
func TestStop(t *testing.T) {
	loops := 5
	for i := 0; i < loops; i++ {
		transport, err := transport.NewManager("udp")
		if err != nil {
			t.Fatalf("Failed to start transport manager on loop %v: %v\n", i, err)
		}

		m, err := transaction.NewManager(transport, "localhost:12345")
		if err != nil {
			t.Fatalf("Failed to start transaction manager on loop %v: %v\n", i, err)
		}

		m.Stop()
	}
}
