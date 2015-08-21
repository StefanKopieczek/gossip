package testutils

import "net"
import "time"

type DummyConn struct{}

func (c *DummyConn) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (c *DummyConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (c *DummyConn) Close() error {
	return nil
}

func (c *DummyConn) LocalAddr() net.Addr {
	return nil
}

func (c *DummyConn) RemoteAddr() net.Addr {
	return nil
}

func (c *DummyConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *DummyConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *DummyConn) SetWriteDeadline(t time.Time) error {
	return nil
}
