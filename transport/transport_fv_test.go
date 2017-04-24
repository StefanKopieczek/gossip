package transport

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/remodoy/gossip/base"
)

type endpoint struct {
	host string
	port uint16
}

var alice endpoint = endpoint{"127.0.0.1", 10862}
var bob endpoint = endpoint{"127.0.0.1", 10863}

func TestMassUDP(t *testing.T) {
	NUM_MSGS := 10000
	from, _ := NewManager("udp")
	to, _ := NewManager("udp")
	to.Listen(fmt.Sprintf("%s:%d", alice.host, alice.port))
	receiver := to.GetChannel()

	receivedIDs := make([]int, 0)
	result := make(chan bool)

	go func() {
	recvloop:
		for {
			select {
			case msg, ok := <-receiver:
				if !ok {
					break recvloop
				}
				id, _ := strconv.ParseInt(msg.GetBody(), 10, 32)
				receivedIDs = append(receivedIDs, int(id))
				if len(receivedIDs) >= NUM_MSGS {
					break recvloop
				}
			case <-time.After(time.Second / 10):
				break recvloop
			}
		}

		if !(len(receivedIDs) == NUM_MSGS) {
			t.Errorf("Error - received an unexpected number of messages: %d", len(receivedIDs))
		} else {
			seenSet := make(map[int]bool)
			for _, val := range receivedIDs {
				if _, match := seenSet[val]; match {
					t.Errorf("Duplicate message received: %d", val)
				}
				seenSet[val] = true
			}
			if len(seenSet) != NUM_MSGS {
				t.Errorf("Unexpected number of messages received: %d", len(seenSet))
			}
		}
		result <- true
	}()

	go func() {
		uri := base.SipUri{User: base.String{"alice"}, Host: "127.0.0.1", Port: nil, UriParams: base.NewParams(), Headers: base.NewParams()}
		for ii := 1; ii <= NUM_MSGS; ii++ {
			from.Send(fmt.Sprintf("%s:%d", alice.host, alice.port),
				base.NewRequest(base.ACK, &uri, "SIP/2.0",
					[]base.SipHeader{base.ContentLength(len(fmt.Sprintf("%d", ii)))},
					fmt.Sprintf("%d", ii)))

			if ii%100 == 0 {
				<-time.After(time.Millisecond)
			}
		}
	}()

	_ = <-result
	return
}

func sendAndCheckReceipt(from Manager, to string,
	receiver chan base.SipMessage,
	msg base.SipMessage, timeout time.Duration) bool {
	from.Send(to, msg)

	select {
	case msgIn, ok := <-receiver:
		if !ok {
			return false
		}

		return msgIn.String() == msg.String()

	case <-time.After(timeout):
		return false
	}
}
