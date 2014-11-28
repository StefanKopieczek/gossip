package transaction

import "testing"

// Tests we can start/stop a transaction manager repeatedly on the same port.
func TestStop(t *testing.T) {
	loops := 5
	for i := 0; i < loops; i++ {
		m, err := NewManager("tcp", "localhost:12345")
		if err != nil {
			t.Fatalf("Failed to start manager on loop %v: %v\n", i, err)
		}

		m.Stop()
	}
}
