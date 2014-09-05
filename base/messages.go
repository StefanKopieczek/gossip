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

    Headers() []SipHeader
    HeadersWithName(name string) []SipHeader
    AddHeader(header SipHeader)
    RemoveHeader(header SipHeader) error

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
	headers map[string][]SipHeader

	// The order the headers should be displayed in.
	headerOrder []string

	// The application data of the message.
	Body string
}

func NewRequest(method Method, recipient Uri, sipVersion string, headers []SipHeader, body string) (request *Request) {
    request = new(Request)
    request.Method = method
    request.Recipient = recipient
    request.headers = make(map[string][]SipHeader)
    request.headerOrder = make([]string, 0)
    request.Body = body

    for _, header := range headers {
        request.AddHeader(header)
    }

    return
}

func (request *Request) String() string {
	var buffer bytes.Buffer

	// Every SIP request starts with a Request Line - RFC 2361 7.1.
	buffer.WriteString(fmt.Sprintf("%s %s %s\r\n",
		(string)(request.Method),
		request.Recipient.String(),
		request.SipVersion))

	// Construct each header in turn and add it to the message.
	for typeIdx, name := range request.headerOrder {
		headers := request.headers[name]
		for idx, header := range headers {
			buffer.WriteString(header.String())
			if typeIdx < len(request.headerOrder) || idx < len(headers) {
				buffer.WriteString("\r\n")
			}
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

    cseqs := request.HeadersWithName("CSeq")
    if len(cseqs) > 0 {
        buffer.WriteString(fmt.Sprintf(" (CSeq: %s)", (cseqs[0].(*CSeq)).String()))
    }

    return buffer.String()
}

func (request *Request) Headers() []SipHeader {
    headers := make([]SipHeader, 0)
    for _, key := range(request.headerOrder) {
        headers = append(headers, request.headers[key]...)
    }

    return headers
}

func (request *Request) HeadersWithName(name string) []SipHeader {
    name = strings.ToLower(name)
    return request.headers[name]
}

func (request *Request) AddHeader(header SipHeader) {
    name := strings.ToLower(header.Name())
    headersOfSameType, isMatch := request.headers[name]
    if !isMatch || len(headersOfSameType) == 0 {
        request.headers[name] = []SipHeader{header}
        request.headerOrder = append(request.headerOrder, name)
    } else {
        request.headers[name] = append(headersOfSameType, header)
    }
}

func (request *Request) RemoveHeader(header SipHeader) error {
    errNoMatch := fmt.Errorf("cannot remove header '%s' from request '%s' as it is not present",
                             header.String(), request.Short())
    name := header.Name()

    headersOfSameType, isMatch := request.headers[name]

    if !isMatch || len(headersOfSameType) == 0 {
        return errNoMatch
    }

    found := false
    for idx, hdr := range headersOfSameType {
        if hdr == header {
            request.headers[name] = append(headersOfSameType[:idx], headersOfSameType[idx+1:]...)
            found = true
            break
        }
    }
    if !found {
        return errNoMatch
    }

    if len(request.headers[name]) == 0 {
        // The header we removed was the only one of its type.
        // Tidy up the header structure by removing the empty list value from the header map,
        // and removing the entry from the headerOrder list.
        delete(request.headers, name)

        for idx, entry := range request.headerOrder {
            if entry == name {
                request.headerOrder = append(request.headerOrder[:idx], request.headerOrder[idx+1:]...)
            }
        }
    }

    return nil
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
	headers map[string][]SipHeader

	// The order the headers should be displayed in.
	headerOrder []string

	// The application data of the message.
	Body string
}

func NewResponse(sipVersion string, statusCode uint16, reason string, headers []SipHeader, body string) (response *Response) {
    response = new(Response)
    response.SipVersion = sipVersion
    response.StatusCode = statusCode
    response.Reason = reason
    response.Body = body
    response.headers = make(map[string][]SipHeader)
    response.headerOrder = make([]string, 0)

    for _, header := range headers {
        response.AddHeader(header)
    }

    return
}

func (response *Response) String() string {
	var buffer bytes.Buffer

	// Every SIP response starts with a Status Line - RFC 2361 7.2.
	buffer.WriteString(fmt.Sprintf("%s %d %s\r\n",
		response.SipVersion,
		response.StatusCode,
		response.Reason))

	// Construct each header in turn and add it to the message.
	for typeIdx, name := range response.headerOrder {
		headers := response.headers[name]
		for idx, header := range headers {
			buffer.WriteString(header.String())
			if typeIdx < len(response.headerOrder) || idx < len(headers) {
				buffer.WriteString("\r\n")
			}
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

    cseqs := response.HeadersWithName("CSeq")
    if len(cseqs) > 0 {
        buffer.WriteString(fmt.Sprintf(" (CSeq: %s)", (cseqs[0].(*CSeq)).String()))
    }

    return buffer.String()
}

func (response *Response) Headers() []SipHeader {
    headers := make([]SipHeader, 0)
    for _, key := range(response.headerOrder) {
        headers = append(headers, response.headers[key]...)
    }

    return headers
}

func (response *Response) HeadersWithName(name string) []SipHeader {
    name = strings.ToLower(name)
    return response.headers[name]
}

func (response *Response) AddHeader(header SipHeader) {
    name := strings.ToLower(header.Name())
    headersOfSameType, isMatch := response.headers[name]
    if !isMatch || len(headersOfSameType) == 0 {
        response.headers[name] = []SipHeader{header}
        response.headerOrder = append(response.headerOrder, name)
    } else {
        response.headers[name] = append(headersOfSameType, header)
    }
}

func (response *Response) RemoveHeader(header SipHeader) error {
    errNoMatch := fmt.Errorf("cannot remove header '%s' from response '%s' as it is not present",
                             header.String(), response.Short())
    name := header.Name()

    headersOfSameType, isMatch := response.headers[name]

    if !isMatch || len(headersOfSameType) == 0 {
        return errNoMatch
    }

    found := false
    for idx, hdr := range headersOfSameType {
        if hdr == header {
            response.headers[name] = append(headersOfSameType[:idx], headersOfSameType[idx+1:]...)
            found = true
            break
        }
    }
    if !found {
        return errNoMatch
    }

    if len(response.headers[name]) == 0 {
        // The header we removed was the only one of its type.
        // Tidy up the header structure by removing the empty list value from the header map,
        // and removing the entry from the headerOrder list.
        delete(response.headers, name)

        for idx, entry := range response.headerOrder {
            if entry == name {
                response.headerOrder = append(response.headerOrder[:idx], response.headerOrder[idx+1:]...)
            }
        }
    }

    return nil
}

func (response *Response) GetBody() string {
    return response.Body
}

func (response *Response) SetBody(body string) {
    response.Body = body
}

