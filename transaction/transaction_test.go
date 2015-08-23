package transaction

import (
	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/transport"
	"testing"
)

// UTs for transaction layer.

// Dummy transport manager.
type dummyTransport struct {
	listenReqs chan string
	messages   chan sentMessage
	toTM       chan base.SipMessage
}

type sentMessage struct {
	addr string
	msg  base.SipMessage
}

func newDummyTransport() *dummyTransport {
	return &dummyTransport{
		listenReqs: make(chan string, 5),
		messages:   make(chan sentMessage, 5),
		toTM:       make(chan base.SipMessage, 5),
	}
}

// Implement transport.Manager interface.
func (t *dummyTransport) Listen(address string) error {
	t.listenReqs <- address
	return nil
}

func (t *dummyTransport) Send(addr string, message base.SipMessage) error {
	t.messages <- sentMessage{addr, message}
	return nil
}

func (t *dummyTransport) Stop() {}

func (t *dummyTransport) GetChannel() transport.Listener {
	return t.toTM
}

// Confirm transaction manager requests for transport to listen.
func TestListenRequest(t *testing.T) {
	trans := newDummyTransport()
	m, err := NewManager(trans, "1.1.1.1")
	if err != nil {
		t.Fatalf("Error creating TM: %v", err)
	}

	addr := <-trans.listenReqs
	if addr != "1.1.1.1" {
		t.Fatalf("Created TM with addr 1.1.1.1 but were asked to listen on %v", addr)
	}

	m.Stop()
}

// Invite request comes in, TU rejects with 503.
func TestServerInvite503(t *testing.T) {
	trans := newDummyTransport()
	m, err := NewManager(trans, "1.1.1.1")
	if err != nil {
		t.Fatalf("Error creating TM: %v", err)
	}

	tu := m.Requests()

	// Send in INVITE.
	invite := base.NewRequest(
		base.INVITE,
		&base.SipUri{
			Host:      "wonderland.com",
			User:      base.String{"alice"},
			Password:  base.NoString{},
			UriParams: base.NewParams(),
			Headers:   base.NewParams(),
		},
		"SIP/2.0",
		[]base.SipHeader{
			&base.ToHeader{
				DisplayName: base.NoString{},
				Address: &base.SipUri{
					Host:      "wonderland.com",
					User:      base.String{"alice"},
					Password:  base.NoString{},
					UriParams: base.NewParams(),
					Headers:   base.NewParams(),
				},
				Params: base.NewParams(),
			},
			&base.FromHeader{
				DisplayName: base.NoString{},
				Address: &base.SipUri{
					Host:      "teaparty.wl",
					User:      base.String{"hatter"},
					Password:  base.NoString{},
					UriParams: base.NewParams(),
					Headers:   base.NewParams(),
				},
				Params: base.NewParams(),
			},
			&base.ViaHeader{
				&base.ViaHop{
					ProtocolName:    "SIP",
					ProtocolVersion: "2.0",
					Transport:       "UDP",
					Host:            "teaparty.wl",
					Port:            nil,
					Params:          base.NewParams().Add("branch", base.String{"branch-1"}),
				},
			},
			base.CallId("call-id-1"),
		},
		"",
	)
	trans.toTM <- invite

	// Receive INVITE at TU, unchanged.
	tx := <-tu
	exp := invite.String()
	got := tx.Origin().String()
	if exp != got {
		t.Fatalf("Received incorrect request at TU:\nExpected:\n%v\n\nGot:\n%v",
			exp, got)
	}

	// Receive automatic 100 Trying.j
	resp := base.NewResponse(
		"SIP/2.0",
		100, "Trying",
		[]base.SipHeader{
			&base.ViaHeader{
				&base.ViaHop{
					ProtocolName:    "SIP",
					ProtocolVersion: "2.0",
					Transport:       "UDP",
					Host:            "teaparty.wl",
					Port:            nil,
					Params:          base.NewParams().Add("branch", base.String{"branch-1"}),
				},
			},
			&base.FromHeader{
				DisplayName: base.NoString{},
				Address: &base.SipUri{
					Host:      "teaparty.wl",
					User:      base.String{"hatter"},
					Password:  base.NoString{},
					UriParams: base.NewParams(),
					Headers:   base.NewParams(),
				},
				Params: base.NewParams(),
			},
			&base.ToHeader{
				DisplayName: base.NoString{},
				Address: &base.SipUri{
					Host:      "wonderland.com",
					User:      base.String{"alice"},
					Password:  base.NoString{},
					UriParams: base.NewParams(),
					Headers:   base.NewParams(),
				},
				Params: base.NewParams(),
			},
			base.CallId("call-id-1"),
		},
		"",
	)

	// Should receive the response out of the transport.
	recv := <-trans.messages
	if recv.addr != "teaparty.wl:5060" {
		t.Errorf("Should have received response on teaparty.wl:5060, actually received on %v", recv.addr)
	}

	exp = resp.String()
	got = recv.msg.String()
	if exp != got {
		t.Fatalf("Received incorrect response at transport:\nExpected:\n%v\n\nGot:\n%v",
			exp, got)
	}

	// Send response from TU.
	resp = base.NewResponse(
		"SIP/2.0",
		503, "Service Unavailable",
		[]base.SipHeader{
			&base.ToHeader{
				DisplayName: base.NoString{},
				Address: &base.SipUri{
					Host:      "wonderland.com",
					User:      base.String{"alice"},
					Password:  base.NoString{},
					UriParams: base.NewParams(),
					Headers:   base.NewParams(),
				},
				Params: base.NewParams(),
			},
			&base.FromHeader{
				DisplayName: base.NoString{},
				Address: &base.SipUri{
					Host:      "teaparty.wl",
					User:      base.String{"hatter"},
					Password:  base.NoString{},
					UriParams: base.NewParams(),
					Headers:   base.NewParams(),
				},
				Params: base.NewParams(),
			},
			&base.ViaHeader{
				&base.ViaHop{
					ProtocolName:    "SIP",
					ProtocolVersion: "2.0",
					Transport:       "UDP",
					Host:            "teaparty.wl",
					Port:            nil,
					Params:          base.NewParams().Add("branch", base.String{"branch-1"}),
				},
			},
			base.CallId("call-id-1"),
		},
		"",
	)
	tx.Respond(resp)

	// Should receive the response out of the transport.
	recv = <-trans.messages
	if recv.addr != "teaparty.wl:5060" {
		t.Errorf("Should have received response on teaparty.wl:5060, actually received on %v", recv.addr)
	}

	exp = resp.String()
	got = recv.msg.String()
	if exp != got {
		t.Fatalf("Received incorrect response at transport:\nExpected:\n%v\n\nGot:\n%v",
			exp, got)
	}

	m.Stop()
}
