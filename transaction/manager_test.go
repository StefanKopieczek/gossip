package transaction

import (
	"runtime"
	"testing"
)

// Tests we can start/stop a transaction manager repeatedly on the same port.
func TestStop(t *testing.T) {
	loops := 5
	goroutines := runtime.NumGoroutine()
	for i := 0; i < loops; i++ {
		m, err := NewManager("tcp", "localhost:12345")
		if err != nil {
			t.Fatalf("Failed to start manager on loop %v: %v\n", i, err)
		}

		m.Stop()

		// Check no goroutines still running.
		n := runtime.NumGoroutine()
		if n != goroutines {
			t.Errorf("%v goroutines still running after manager closed on loop %v.", n, i)
		}
	}
	trace := make([]byte, 8192)
	count := runtime.Stack(trace, true)
	t.Log(string(trace[:count]))
}
