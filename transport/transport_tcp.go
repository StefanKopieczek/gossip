package transport

import (
	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
	"github.com/stefankopieczek/gossip/parser"
)

import "net"

type Tcp struct {
	connTable
	laddr *net.TCPAddr
	in    *net.TCPListener
}

func NewTcp(addr string) (*Tcp, error) {
	laddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	tcp := Tcp{laddr: laddr}
	tcp.connTable.Init()
	return &tcp, nil
}

func (tcp *Tcp) Listen(parser parser.Parser) error {
	var err error = nil
	tcp.in, err = net.ListenTCP("tcp", tcp.laddr)

	if err == nil {
		go tcp.listen(parser)
	}

	return err
}

func (tcp *Tcp) IsStreamed() bool {
	return true
}

func (tcp *Tcp) getConn(addr string) (*net.TCPConn, error) {
	conn := tcp.connTable.GetConn(addr)

	if conn == nil {
		log.Debug("No stored connection for address %s", addr)
		raddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			return nil, err
		}
		conn, err = net.DialTCP("tcp", tcp.laddr, raddr)
		if err != nil {
			return nil, err
		}
	}

	tcp.connTable.Notify(addr, conn)
	return conn.(*net.TCPConn), nil
}

func (tcp *Tcp) Send(addr string, msg base.SipMessage) error {
	conn, err := tcp.getConn(addr)
	if err != nil {
		return err
	}

	_, err = conn.Write([]byte(msg.String()))
	return err
}

func (tcp *Tcp) listen(parser parser.Parser) {
	log.Info("Begin listening for TCP on address " + tcp.laddr.String())

	for {
		conn, err := tcp.in.Accept()
		if err != nil {
			log.Severe("Failed to accept TCP conn on address " + tcp.laddr.String() + "; " + err.Error())
			continue
		}
		go func(c *net.TCPConn) {
			buffer := make([]byte, c_BUFSIZE)
			for {
				num, err := conn.Read(buffer)
				if err != nil {
					log.Severe("Failed to read from TCP buffer: " + err.Error())
					continue
				}

				pkt := append([]byte(nil), buffer[:num]...)
				parser.Write(pkt)
			}
		}(conn.(*net.TCPConn))
	}
}

func (tcp *Tcp) Stop() {
	tcp.connTable.Stop()
}
