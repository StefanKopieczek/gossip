package gossip

import (
    "bytes"
    "fmt"
)

type Method string

const (
    INVITE   Method = "INVITE"
    ACK      Method = "ACK"
    CANCEL   Method = "CANCEL"
    BYE      Method = "BYE"
    REGISTER Method = "REGISTER"
    OPTIONS  Method = "OPTIONS"
)

type SipMessage interface {
    String()
}

type Request struct {
    method Method
    uri SipUri
    sipVersion string
    headers []SipHeader
    body *string
}
func (request *Request) String() (string) {
    var buffer bytes.Buffer

    // Every SIP request starts with a Request Line - RFC 2361 7.1.
    buffer.WriteString(fmt.Sprintf("%s %s %s\r\n",
        (string)(request.method),
        request.uri.String(),
        request.sipVersion))

    // Construct each header in turn and add it to the message.
    for idx, header := range(request.headers) {
        buffer.WriteString(header.String())

        if (idx < len(request.headers)) {
            buffer.WriteString("\r\n")
        }
    }

    // If the request has a message body, add it.
    if (request.body != nil) {
        buffer.WriteString(*request.body)
    }

    return buffer.String()
}

type Response struct {
    sipVersion string
    statusCode uint8
    reason string
    headers []SipHeader
    body *string
}
func (response *Response) String() (string) {
    var buffer bytes.Buffer

    // Every SIP response starts with a Status Line - RFC 2361 7.2.
    buffer.WriteString(fmt.Sprintf("%s %d %s",
        response.sipVersion,
        response.statusCode,
        response.reason))

    // Construct each header in turn and add it to the message.
    for idx, header := range(response.headers) {
        buffer.WriteString(header.String())

        if (idx < len(response.headers)) {
            buffer.WriteString("\r\n")
        }
    }

    // If the request has a message body, add it.
    if (response.body != nil) {
        buffer.WriteString(*response.body)
    }

    return buffer.String()
}
