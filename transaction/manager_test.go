package transaction

import (
	"testing"
	"time"
)

// Tests we can start/stop a transaction manager repeatedly on the same port.
func TestStop(t *testing.T) {
	loops := 5
	for i := 0; i < loops; i++ {
		m, err := NewManager("tcp", "localhost:12345")
		if err != nil {
			t.Fatalf("Failed to start manager on loop %v: %v\n", i, err)
		}

		<-time.After(10 * time.Millisecond)
		m.Stop()
	}
}
