package testutils

import (
	"time"
)

// Runs f at 10ms intervals until it returns True.
// Gives up after 100 attempts.
func Eventually(f func() bool) bool {
	attempts := 0
	for !f() {
		if attempts++; attempts >= 100 {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
	return true
}
