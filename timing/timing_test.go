package timing

// Tests for the mock timing module.

import (
	"testing"
	"time"
)

func TestTimer(t *testing.T) {
	MockMode = true
	timer := NewTimer(5 * time.Second)
	done := make(chan struct{})

	go func() {
		<-timer.C()
		done <- struct{}{}
	}()

	Elapse(5 * time.Second)
	<-done
}

func TestTwoTimers(t *testing.T) {
	MockMode = true
	timer1 := NewTimer(5 * time.Second)
	done1 := make(chan struct{})

	timer2 := NewTimer(5 * time.Millisecond)
	done2 := make(chan struct{})

	go func() {
		<-timer1.C()
		done1 <- struct{}{}
	}()

	go func() {
		<-timer2.C()
		done2 <- struct{}{}
	}()

	Elapse(5 * time.Millisecond)
	<-done2

	Elapse(9995 * time.Millisecond)
	<-done1
}

func TestAfter(t *testing.T) {
	MockMode = true
	done := make(chan struct{})
	c := After(5 * time.Second)

	go func() {
		<-c
		done <- struct{}{}
	}()

	Elapse(5 * time.Second)
	<-done
}

func TestAfterFunc(t *testing.T) {
	MockMode = true
	done := make(chan struct{})
	AfterFunc(5*time.Second,
		func() {
			done <- struct{}{}
		})

	Elapse(5 * time.Second)
	<-done
}
