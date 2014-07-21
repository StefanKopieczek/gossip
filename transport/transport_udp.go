package transport

import (
	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
	"github.com/stefankopieczek/gossip/parser"
)

import (
	"net"
)

const c_BUFSIZE int = 65507

type UdpTransportManager struct {
	notifier
	address *net.UDPAddr
	in      *net.UDPConn
	parser  parser.Parser
}

func NewUdpTransportManager(address *net.UDPAddr) *UdpTransportManager {
	manager := UdpTransportManager{notifier{}, address, nil, nil}
	return &manager
}

func (transport *UdpTransportManager) Listen() error {
	var err error = nil
	transport.in, err = net.ListenUDP("udp", transport.address)

	if err == nil {
		go transport.listen()
	}

	return err
}

func (transport *UdpTransportManager) GetChannel() (c chan base.SipMessage) {
	return transport.notifier.getChannel()
}

func (transport *UdpTransportManager) Send(addr string, msg base.SipMessage) error {
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	var conn *net.UDPConn
	conn, err = net.DialUDP("udp", transport.address, raddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(msg.String()))
	return err
}

func (transport *UdpTransportManager) listen() {
	log.Info("Begin listening for UDP on address " + transport.address.String())
	parsedMessages := make(chan base.SipMessage, 0)
	errors := make(chan error, 0)
	transport.parser = parser.NewParser(parsedMessages, errors, false)

	go func() {
		running := true
		for running {
			select {
			case message, ok := <-parsedMessages:
				if ok {
					transport.notifier.notifyAll(message)
				} else {
					log.Info("Parser stopped in UDP Transport Manager; will stop listening")
					running = false
				}
			case err, ok := <-errors:
				if ok {
					// The parser has hit a terminal error. We need to restart it.
					log.Warn("Failed to parse SIP message", err)
					transport.parser = parser.NewParser(parsedMessages, errors, false)
				} else {
					log.Info("Parser stopped in UDP Transport Manager; will stop listening")
				}
			}
		}
	}()

	buffer := make([]byte, c_BUFSIZE)
	for {
		num, _, err := transport.in.ReadFromUDP(buffer)
		if err != nil {
			log.Severe("Failed to read from UDP buffer: " + err.Error())
			continue
		}

		pkt := append([]byte(nil), buffer[:num]...)
		transport.parser.Write(pkt)
	}
}

func (transport *UdpTransportManager) Stop() {
	transport.in.Close()
	// Also stop parser! TODO.
}
