package gossip

import "fmt"

type Method string

const (
    INVITE  Method = "INVITE"
    ACK     Method = "ACK"
    CANCEL  Method = "CANCEL"
    BYE     Method = "BYE"
    OPTIONS Method = "OPTIONS"
)

type SipMessage interface {
    String()
}

// A RequestLine is required at the start of every SIP request - RFC 3261 7.1.
type RequestLine struct {
    method Method
    uri SipUri
    sipVersion string
}
func (requestLine *RequestLine) String() (string) {
    return fmt.Sprintf("%s %s %s", (string)(requestLine.method),
        requestLine.uri.String(), requestLine.sipVersion)
}

// A StatusLine is required at the start of every SIP response - RFC 3261 7.2.
type StatusLine struct {
    sipVersion string
    statusCode uint8
    reason string
}
func (statusLine *StatusLine) String() (string) {
    return fmt.Sprintf("%s %d %s", statusLine.sipVersion,
        statusLine.statusCode, statusLine.reason)
}
