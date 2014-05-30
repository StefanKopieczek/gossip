package gossip

import "net"

type MessageHandler interface {
    ProcessMessage(message SipMessage)
}

type SipTransportManager interface {
    Start()
    Send(message *SipMessage)
    AddMessageHandler(handler MessageHandler)
    RemoveMessageHandler(handler MessageHandler)
    Stop()
}

type UdpTransportManager struct {
    address *net.UDPAddr
    conn *net.UDPConn
    messageHandlers []MessageHandler
}
func NewUdpTransportManager(address *net.UDPAddr) (*UdpTransportManager, error) {
    handlers := make([]MessageHandler, 0)
    manager := UdpTransportManager{address, nil, handlers}
    return &manager, nil
}
func (transport *UdpTransportManager) Start() (error) {
    var err error = nil
    transport.conn, err = net.ListenUDP("udp", transport.address)

    if (err == nil) {
        go func() {
            parser := NewMessageParser()
            for {
                pkt := make([]byte, 0)
                _, _, _ = transport.conn.ReadFromUDP(pkt)  // TODO: DO this properly.
                go func() {
                    message, _ := parser.ParseMessage(pkt) // TODO: Handle error
                    for _, handler := range(transport.messageHandlers) {
                        handler.ProcessMessage(message)
                    }
                }()
            }
        }()
    }

    return err
}
