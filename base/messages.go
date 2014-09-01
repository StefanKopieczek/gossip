package base

import (
	"bytes"
	"fmt"
	"strings"
)

// A representation of a SIP method.
// This is syntactic sugar around the string type, so make sure to use
// the Equals method rather than built-in equality, or you'll fall foul of case differences.
// If you're defining your own Method, uppercase is preferred but not compulsory.
type Method string

// Determine if the given method equals some other given method.
// This is syntactic sugar for case insensitive equality checking.
func (method *Method) Equals(other *Method) bool {
	if method != nil && other != nil {
		return strings.EqualFold(string(*method), string(*other))
	} else {
		return method == other
	}
}

// It's nicer to avoid using raw strings to represent methods, so the following standard
// method names are defined here as constants for convenience.
const (
	INVITE    Method = "INVITE"
	ACK       Method = "ACK"
	CANCEL    Method = "CANCEL"
	BYE       Method = "BYE"
	REGISTER  Method = "REGISTER"
	OPTIONS   Method = "OPTIONS"
	SUBSCRIBE Method = "SUBSCRIBE"
	NOTIFY    Method = "NOTIFY"
	REFER     Method = "REFER"
)

// Internal representation of a SIP message - either a Request or a Response.
type SipMessage interface {
	// Yields a flat, string representation of the SIP message suitable for sending out over the wire.
	String() string

    // Yields a short string representation of the message useful for logging.
    Short() string

    AllHeaders() []SipHeader
    HeadersByName(name string) []SipHeader

    GetBody() string
    SetBody(body string)
}

// A SIP request (c.f. RFC 3261 section 7.1).
type Request struct {
	// Which method this request is, e.g. an INVITE or a REGISTER.
	Method Method

	// The Request URI. This indicates the user to whom this request is being addressed.
	Recipient Uri

	// The version of SIP used in this message, e.g. "SIP/2.0".
	SipVersion string

	// The logical SIP headers attached to this message.
	Headers []SipHeader

	// The application data of the message.
	Body string
}

func (request *Request) String() string {
	var buffer bytes.Buffer

	// Every SIP request starts with a Request Line - RFC 2361 7.1.
	buffer.WriteString(fmt.Sprintf("%s %s %s\r\n",
		(string)(request.Method),
		request.Recipient.String(),
		request.SipVersion))

	// Construct each header in turn and add it to the message.
	for idx, header := range request.Headers {
		buffer.WriteString(header.String())

		if idx < len(request.Headers) {
			buffer.WriteString("\r\n")
		}
	}

	// If the request has a message body, add it.
    buffer.WriteString("\r\n" + request.Body)

	return buffer.String()
}

func (request *Request) Short() string {
    var buffer bytes.Buffer

    buffer.WriteString(fmt.Sprintf("%s %s %s",
        (string)(request.Method),
        request.Recipient.String(),
        request.SipVersion))

    cseqs := request.HeadersByName("CSeq")
    if len(cseqs) > 0 {
        buffer.WriteString(fmt.Sprintf(" (CSeq: %s)", (cseqs[0].(*CSeq)).String()))
    }

    return buffer.String()
}

func (request *Request) AllHeaders() []SipHeader {
    return request.Headers
}

func (request *Request) HeadersByName(name string) []SipHeader {
    result := make([]SipHeader, 0)
    for _, header := range(request.Headers) {
        // TODO: Filter headers
        result = append(result, header)
    }

    return result
}

func (request *Request) GetBody() string {
    return request.Body
}

func (request *Request) SetBody(body string) {
    request.Body = body
}

// A SIP response object  (c.f. RFC 3261 section 7.2).
type Response struct {
	// The version of SIP used in this message, e.g. "SIP/2.0".
	SipVersion string

	// The response code, e.g. 200, 401 or 500.
	// This indicates the outcome of the originating request.
	StatusCode uint16

	// The reason string provides additional, human-readable information used to provide
	// clarification or explanation of the status code.
	// This will vary between different SIP UAs, and should not be interpreted by the receiving UA.
	Reason string

	// The logical SIP headers attached to this message.
	Headers []SipHeader

	// The application data of the message.
	Body string
}

func (response *Response) String() string {
	var buffer bytes.Buffer

	// Every SIP response starts with a Status Line - RFC 2361 7.2.
	buffer.WriteString(fmt.Sprintf("%s %d %s\r\n",
		response.SipVersion,
		response.StatusCode,
		response.Reason))

	// Construct each header in turn and add it to the message.
	for idx, header := range response.Headers {
		buffer.WriteString(header.String())

		if idx < len(response.Headers) {
			buffer.WriteString("\r\n")
		}
	}

	// If the request has a message body, add it.
    buffer.WriteString("\r\n" + response.Body)

	return buffer.String()
}

func (response *Response) Short() string {
    var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("%s %d %s\r\n",
		response.SipVersion,
		response.StatusCode,
		response.Reason))

    cseqs := response.HeadersByName("CSeq")
    if len(cseqs) > 0 {
        buffer.WriteString(fmt.Sprintf(" (CSeq: %s)", (cseqs[0].(*CSeq)).String()))
    }

    return buffer.String()
}

func (response *Response) AllHeaders() []SipHeader {
    return response.Headers
}

func (response *Response) HeadersByName(name string) []SipHeader {
    result := make([]SipHeader, 0)
    for _, header := range(response.Headers) {
        // TODO: Filter headers
        result = append(result, header)
    }

    return result
}

func (response *Response) GetBody() string {
    return response.Body
}

func (response *Response) SetBody(body string) {
    response.Body = body
}

