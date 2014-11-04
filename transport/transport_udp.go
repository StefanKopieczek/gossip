package transport

import (
	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
	"github.com/stefankopieczek/gossip/parser"
)

import (
	"net"
)

type Udp struct {
	listeningPoints []*net.UDPConn
	output          chan base.SipMessage
}

func NewUdp(output chan base.SipMessage) (*Udp, error) {
	newUdp := Udp{listeningPoints: make([]*net.UDPConn, 0), output: output}
	return &newUdp, nil
}

func (udp *Udp) Listen(address string) error {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return err
	}

	lp, err := net.ListenUDP("udp", addr)

	if err == nil {
		udp.listeningPoints = append(udp.listeningPoints, lp)
		go udp.listen(lp)
	}

	return err
}

func (udp *Udp) IsStreamed() bool {
	return false
}

func (udp *Udp) Send(addr string, msg base.SipMessage) error {
	log.Debug("Sending message %s to %s", msg.Short(), addr)
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	var conn *net.UDPConn
	conn, err = net.DialUDP("udp", nil, raddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(msg.String()))

	return err
}

func (udp *Udp) listen(conn *net.UDPConn) {
	log.Info("Begin listening for UDP on address %s", conn.LocalAddr())

	buffer := make([]byte, c_BUFSIZE)
	for {
		num, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Severe("Failed to read from UDP buffer: " + err.Error())
			continue
		}

		pkt := append([]byte(nil), buffer[:num]...)
		go func() {
			msg, err := parser.ParseMessage(pkt)
			if err != nil {
				log.Warn("Failed to parse SIP message: %s", err.Error())
			} else {
				udp.output <- msg
			}
		}()
	}
}

func (udp *Udp) Stop() {
	// TODO: Close all listening points.
}
