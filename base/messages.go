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

	// Adds a header to this message.
	AddHeader(h SipHeader)

	// Returns a slice of all headers of the given type.
	// If there are no headers of the requested type, returns an empty slice.
	Headers(name string) []SipHeader
}

// A shared type for holding headers and their ordering.
type headers struct {
	// The logical SIP headers attached to this message.
	headers map[string][]SipHeader

	// The order the headers should be displayed in.
	headerOrder []string
}

func (h headers) String() string {
	buffer := bytes.Buffer{}
	// Construct each header in turn and add it to the message.
	for typeIdx, name := range h.headerOrder {
		headers := h.headers[name]
		for idx, header := range headers {
			buffer.WriteString(header.String())
			if typeIdx < len(h.headerOrder) || idx < len(headers) {
				buffer.WriteString("\r\n")
			}
		}
	}
	return buffer.String()
}

// Add the given header.
func (hs *headers) AddHeader(h SipHeader) {
	if hs.headers == nil {
		hs.headers = map[string][]SipHeader{}
		hs.headerOrder = []string{}
	}
	name := h.Name()
	if _, ok := hs.headers[name]; ok {
		hs.headers[name] = append(hs.headers[name], h)
	} else {
		hs.headers[name] = []SipHeader{h}
		hs.headerOrder = append(hs.headerOrder, name)
	}
}

// Gets some headers.
func (hs *headers) Headers(name string) []SipHeader {
	if hs.headers == nil {
		hs.headers = map[string][]SipHeader{}
		hs.headerOrder = []string{}
	}
	if headers, ok := hs.headers[name]; ok {
		return headers
	} else {
		return []SipHeader{}
	}
}

// Copy all headers of one type from one message to another.
// Appending to any headers that were already there.
func CopyHeaders(name string, from, to SipMessage) {
	for _, h := range from.Headers(name) {
		to.AddHeader(h.Copy())
	}
}

// A SIP request (c.f. RFC 3261 section 7.1).
type Request struct {
	// Which method this request is, e.g. an INVITE or a REGISTER.
	Method Method

	// The Request URI. This indicates the user to whom this request is being addressed.
	Recipient Uri

	// The version of SIP used in this message, e.g. "SIP/2.0".
	SipVersion string

	// A Request has headers.
	headers

	// The application data of the message.
	Body *string
}

func (request *Request) String() string {
	var buffer bytes.Buffer

	// Every SIP request starts with a Request Line - RFC 2361 7.1.
	buffer.WriteString(fmt.Sprintf("%s %s %s\r\n",
		(string)(request.Method),
		request.Recipient.String(),
		request.SipVersion))

	buffer.WriteString(request.headers.String())

	// If the request has a message body, add it.
	if request.Body != nil {
		buffer.WriteString("\r\n" + *request.Body)
	}

	return buffer.String()
}

// A SIP response object  (c.f. RFC 3261 section 7.2).
type Response struct {
	// The version of SIP used in this message, e.g. "SIP/2.0".
	SipVersion string

	// The response code, e.g. 200, 401 or 500.
	// This indicates the outcome of the originating request.
	StatusCode uint8

	// The reason string provides additional, human-readable information used to provide
	// clarification or explanation of the status code.
	// This will vary between different SIP UAs, and should not be interpreted by the receiving UA.
	Reason string

	// A response has headers.
	headers

	// The application data of the message.
	Body *string
}

func (response *Response) String() string {
	var buffer bytes.Buffer

	// Every SIP response starts with a Status Line - RFC 2361 7.2.
	buffer.WriteString(fmt.Sprintf("%s %d %s\r\n",
		response.SipVersion,
		response.StatusCode,
		response.Reason))

	// Write the headers.
	buffer.WriteString(response.headers.String())

	// If the request has a message body, add it.
	if response.Body != nil {
		buffer.WriteString("\r\n" + *response.Body)
	}

	return buffer.String()
}
