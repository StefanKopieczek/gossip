package parser

import (
	"github.com/weave-lab/gossip/base"
	"github.com/weave-lab/gossip/log"
	"github.com/weave-lab/gossip/sipuri"
)

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// The whitespace characters recognised by the Augmented Backus-Naur Form syntax
// that SIP uses (RFC 3261 S.25).
const c_ABNF_WS = " \t"

// The maximum permissible CSeq number in a SIP message (2**31 - 1).
// C.f. RFC 3261 S. 8.1.1.5.
const MAX_CSEQ = 2147483647

// The buffer size of the parser input channel.
const c_INPUT_CHAN_SIZE = 10

// A Parser converts the raw bytes of SIP messages into base.SipMessage objects.
// It allows
type Parser interface {
	// Implements io.Writer. Queues the given bytes to be parsed.
	// If the parser has terminated due to a previous fatal error, it will return n=0 and an appropriate error.
	// Otherwise, it will return n=len(p) and err=nil.
	// Note that err=nil does not indicate that the data provided is valid - simply that the data was successfully queued for parsing.
	Write(p []byte) (n int, err error)

	// Register a custom header parser for a particular header type.
	// This will overwrite any existing registered parser for that header type.
	// If a parser is not available for a header type in a message, the parser will produce a base.GenericHeader struct.
	SetHeaderParser(headerName string, headerParser HeaderParser)

	Stop()
}

// A HeaderParser is any function that turns raw header data into one or more SipHeader objects.
// The HeaderParser will receive arguments of the form ("max-forwards", "70").
// It should return a slice of headers, which should have length > 1 unless it also returns an error.
type HeaderParser func(headerName string, headerData string) (
	headers []base.SipHeader, err error)

func defaultHeaderParsers() map[string]HeaderParser {
	return map[string]HeaderParser{
		"to":             parseAddressHeader,
		"t":              parseAddressHeader,
		"from":           parseAddressHeader,
		"f":              parseAddressHeader,
		"contact":        parseAddressHeader,
		"m":              parseAddressHeader,
		"call-id":        parseCallId,
		"cseq":           parseCSeq,
		"via":            parseViaHeader,
		"v":              parseViaHeader,
		"max-forwards":   parseMaxForwards,
		"content-length": parseContentLength,
		"l":              parseContentLength,
	}
}

// Parse a SIP message by creating a parser on the fly.
// This is more costly than reusing a parser, but is necessary when we do not
// have a guarantee that all messages coming over a connection are from the
// same endpoint (e.g. UDP).
func ParseMessage(msgData []byte) (base.SipMessage, error) {
	output := make(chan base.SipMessage, 0)
	errors := make(chan error, 0)
	parser := NewParser(output, errors, false)
	defer parser.Stop()

	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		parser.Write(msgData)
		parser.Stop()
		wg.Done()
	}()

	select {
	case msg := <-output:
		wg.Wait()
		return msg, nil
	case err := <-errors:
		parser.Stop()
		wg.Wait()
		return nil, err
	}
}

// Create a new Parser.
//
// Parsed SIP messages will be sent down the 'output' chan provided.
// Any errors which force the parser to terminate will be sent down the 'errs' chan provided.
//
// If streamed=false, each Write call to the parser should contain data for one complete SIP message.

// If streamed=true, Write calls can contain a portion of a full SIP message.
// The end of one message and the start of the next may be provided in a single call to Write.
// When streamed=true, all SIP messages provided must have a Content-Length header.
// SIP messages without a Content-Length will cause the parser to permanently stop, and will result in an error on the errs chan.

// 'streamed' should be set to true whenever the caller cannot reliably identify the starts and ends of messages from the transport frames,
// e.g. when using streamed protocols such as TCP.
func NewParser(output chan<- base.SipMessage, errs chan<- error, streamed bool) Parser {
	p := parser{streamed: streamed}

	// Configure the parser with the standard set of header parsers.
	p.headerParsers = make(map[string]HeaderParser)
	for headerName, headerParser := range defaultHeaderParsers() {
		p.SetHeaderParser(headerName, headerParser)
	}

	p.output = output
	p.errs = errs

	if !streamed {
		// If we're not in streaming mode, set up a channel so the Write method can pass calculated body lengths to the parser.
		p.bodyLength = make(chan int, 1)
	}

	// Create a managed buffer to allow message data to be asynchronously provided to the parser, and
	// to allow the parser to block until enough data is available to parse.
	p.input = newParserBuffer()

	// Wait for input a line at a time, and produce SipMessages to send down p.output.
	go p.parse(streamed)

	return &p
}

type parser struct {
	headerParsers map[string]HeaderParser
	streamed      bool
	input         *parserBuffer
	bodyLength    chan int
	output        chan<- base.SipMessage
	errs          chan<- error
	terminalErr   error
	stopped       bool
}

func (p *parser) Write(data []byte) (int, error) {
	if p.terminalErr != nil {
		// The parser has stopped due to a terminal error. Return it.
		log.Fine("Parser %p ignores %d new bytes due to previous terminal error: %s", p, len(data), p.terminalErr.Error())
		return 0, p.terminalErr
	} else if p.stopped {
		return 0, fmt.Errorf("Cannot write data to stopped parser %p", p)
	}

	if !p.streamed {
		l := getBodyLength(data)
		p.bodyLength <- l
	}

	n, err := p.input.Write(data)
	if err != nil {
		return n, err
	}

	return n, nil
}

// Stop parser processing, and allow all resources to be garbage collected.
// The parser will not release its resources until Stop() is called,
// even if the parser object itself is garbage collected.
func (p *parser) Stop() {
	log.Debug("Stopping parser %p", p)
	p.stopped = true
	p.input.Stop()
	log.Debug("Parser %p stopped", p)
}

// Consume input lines one at a time, producing base.SipMessage objects and sending them down p.output.
func (p *parser) parse(requireContentLength bool) {
	var message base.SipMessage

	for {
		// Parse the StartLine.
		startLine, err := p.input.NextLine()

		if err != nil {
			log.Debug("Parser %p stopped", p)
			break
		}

		if parts, ok := isRequest(startLine); ok {
			method, recipient, sipVersion, err := parseRequestLine(parts)
			p.terminalErr = err

			message = base.NewRequest(method, recipient, sipVersion, []base.SipHeader{}, "")

		} else if parts, ok := isResponse(startLine); ok {
			sipVersion, statusCode, reason, err := parseStatusLine(parts)
			p.terminalErr = err

			message = base.NewResponse(sipVersion, statusCode, reason, []base.SipHeader{}, "")
		} else {
			p.terminalErr = fmt.Errorf("transmission beginning '%s' is not a SIP message", startLine)
		}

		if p.terminalErr != nil {
			p.terminalErr = fmt.Errorf("failed to parse first line of message: %s", p.terminalErr.Error())
			p.errs <- p.terminalErr
			break
		}

		// Parse the header section.
		// Headers can be split across lines (marked by whitespace at the start of subsequent lines),
		// so store lines into a buffer, and then flush and parse it when we hit the end of the header.
		var buffer bytes.Buffer
		headers := make([]base.SipHeader, 0)

		flushBuffer := func() {
			if buffer.Len() > 0 {
				newHeaders, err := p.parseHeader(buffer.String())
				if err == nil {
					headers = append(headers, newHeaders...)
				} else {
					log.Debug("Skipping header '%s' due to error: %s", buffer.String(), err.Error())
				}
				buffer.Reset()
			}
		}

		for {
			line, err := p.input.NextLine()

			if err != nil {
				log.Debug("Parser %p stopped", p)
				break
			}

			if len(line) == 0 {
				// We've hit the end of the header section.
				// Parse anything remaining in the buffer, then break out.
				flushBuffer()
				break
			}

			if !strings.Contains(c_ABNF_WS, string(line[0])) {
				// This line starts a new header.
				// Parse anything currently in the buffer, then store the new header line in the buffer.
				flushBuffer()
				buffer.WriteString(line)
			} else if buffer.Len() > 0 {
				// This is a continuation line, so just add it to the buffer.
				buffer.WriteString(" ")
				buffer.WriteString(line)
			} else {
				// This is a continuation line, but also the first line of the whole header section.
				// Discard it and log.
				log.Debug("Discarded unexpected continuation line '%s' at start of header block in message '%s'",
					line,
					message.Short())
			}
		}

		// Store the headers in the message object.
		for _, header := range headers {
			message.AddHeader(header)
		}

		contentLength, err := p.getContentLength(message)
		if err != nil {
			p.terminalErr = err
			p.errs <- err
			break
		}

		// Extract the message body.
		body, err := p.input.NextChunk(contentLength)

		if err != nil {
			p.terminalErr = err
			p.errs <- p.terminalErr
			log.Debug("Parsed %p stopped", p)
			break
		}

		switch message.(type) {
		case *base.Request:
			message.(*base.Request).Body = body
		case *base.Response:
			message.(*base.Response).Body = body
		default:
			log.Severe("Internal error - message %s is neither a request type nor a response type", message.Short())
		}
		p.output <- message
	}

	if !p.streamed {
		// We're in unstreamed mode, so we created a bodyLength chan which
		// needs to be closed.
		close(p.bodyLength)
	}

	p.Stop()

	return
}

func (p *parser) getContentLength(message base.SipMessage) (int, error) {

	// Determine the length of the body, so we know when to stop parsing this message.
	// Use the content-length header to identify the end of the message.
	contentLengthHeaders := message.Headers("Content-Length")
	if len(contentLengthHeaders) == 0 {
		// if streamed, content-length is required
		if p.streamed {
			return 0, fmt.Errorf("Missing required content-length header on message %s", message.Short())
		}

		// We're not in streaming mode, so the Write method should have calculated the length of the body for us.
		return <-p.bodyLength, nil

	} else if len(contentLengthHeaders) > 1 {

		// Can't handle multiple content-lengths
		var errbuf bytes.Buffer
		errbuf.WriteString("Multiple content-length headers on message ")
		errbuf.WriteString(message.Short())
		errbuf.WriteString(":\n")
		for _, header := range contentLengthHeaders {
			errbuf.WriteString("\t")
			errbuf.WriteString(header.String())
		}
		return 0, fmt.Errorf(errbuf.String())

	}

	if contentLengthHeaders[0] == nil {
		return 0, fmt.Errorf("Unexpected nil Content-Length header")
	}

	if l, ok := contentLengthHeaders[0].(*base.ContentLength); ok {

		if l == nil {
			return 0, fmt.Errorf("Unexpected nil Content-Length value")
		}

		return int(*l), nil
	}

	return 0, fmt.Errorf("Unable to get content length header")

}

// Implements ParserFactory.SetHeaderParser.
func (p *parser) SetHeaderParser(headerName string, headerParser HeaderParser) {
	headerName = strings.ToLower(headerName)
	p.headerParsers[headerName] = headerParser
}

// Calculate the size of a SIP message's body, given the entire contents of the message as a byte array.
func getBodyLength(data []byte) int {
	s := string(data)

	// Body starts with first character following a double-CRLF.
	bodyStart := strings.Index(s, "\r\n\r\n") + 4

	return len(s) - bodyStart
}

// Heuristic to determine if the given transmission looks like a SIP request.
// It is guaranteed that any RFC3261-compliant request will pass this test,
// but invalid messages may not necessarily be rejected.
func isRequest(startLine string) ([]string, bool) {

	// SIP request lines contain precisely two spaces.
	parts := strings.Split(startLine, " ")
	if len(parts) != 3 {
		return nil, false
	}

	// Check that the version string starts with SIP.
	return parts, len(parts[2]) >= 3 && strings.ToUpper(parts[2][:3]) == "SIP"
}

// Heuristic to determine if the given transmission looks like a SIP response.
// It is guaranteed that any RFC3261-compliant response will pass this test,
// but invalid messages may not necessarily be rejected.
func isResponse(startLine string) ([]string, bool) {

	// SIP status lines contain at least two spaces.
	parts := strings.Split(startLine, " ")
	if len(parts) < 3 {
		return nil, false
	}

	// Check that the version string starts with SIP.
	return parts, len(parts[0]) >= 3 && strings.ToUpper(parts[0][:3]) == "SIP"
}

// Parse the first line of a SIP request, e.g:
//   INVITE bob@example.com SIP/2.0
//   REGISTER jane@telco.com SIP/1.0
func parseRequestLine(parts []string) (
	method base.Method, recipient base.Uri, sipVersion string, err error) {

	if len(parts) != 3 {
		err = fmt.Errorf("request line should have at least 3 parts: %v", parts)
		return
	}

	method = base.Method(strings.ToUpper(parts[0]))
	recipient, err = sipuri.ParseUri(parts[1])
	sipVersion = parts[2]

	switch recipient.(type) {
	case *base.WildcardUri:
		err = fmt.Errorf("wildcard URI '*' not permitted in request line: '%v'", parts)
	}

	return
}

// Parse the first line of a SIP response, e.g:
//   SIP/2.0 200 OK
//   SIP/1.0 403 Forbidden
func parseStatusLine(parts []string) (
	sipVersion string, statusCode uint16, reasonPhrase string, err error) {

	if len(parts) < 3 {
		err = fmt.Errorf("status line has too few spaces: '%v'", parts)
		return
	}

	sipVersion = parts[0]
	statusCodeRaw, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return
	}

	statusCode = uint16(statusCodeRaw)
	reasonPhrase = strings.Join(parts[2:], "")

	return
}

// Parse a header string, producing one or more SipHeader objects.
// (SIP messages containing multiple headers of the same type can express them as a
// single header containing a comma-separated argument list).
func (p *parser) parseHeader(headerText string) ([]base.SipHeader, error) {
	log.Debug("Parser %p parsing header \"%s\"", p, headerText)

	colonIdx := strings.Index(headerText, ":")
	if colonIdx == -1 {
		return nil, fmt.Errorf("Field name with no value in header: %s", headerText)
	}

	fieldName := strings.ToLower(strings.TrimSpace(headerText[:colonIdx]))
	fieldText := strings.TrimSpace(headerText[colonIdx+1:])

	if headerParser, ok := p.headerParsers[fieldName]; ok {
		// We have a registered parser for this header type - use it.
		headers, err := headerParser(fieldName, fieldText)
		if err != nil {
			return nil, err
		}

		return headers, nil
	}

	// We have no registered parser for this header type,
	// so we encapsulate the header data in a GenericHeader struct.
	log.Debug("Parser %p has no parser for header type %s", p, fieldName)
	header := base.GenericHeader{fieldName, fieldText}

	return []base.SipHeader{&header}, nil
}

// Parse a To, From or Contact header line, producing one or more logical SipHeaders.
func parseAddressHeader(headerName string, headerText string) ([]base.SipHeader, error) {
	// assume headerName is "to", "from", "contact", "t", "f", "m":

	// Perform the actual parsing. The rest of this method is just typeclass bookkeeping.
	displayNames, uris, paramSets, err := ParseAddressValues(headerText)
	if err != nil {
		return nil, err
	}

	if len(displayNames) != len(uris) || len(uris) != len(paramSets) {
		// This shouldn't happen unless ParseAddressValues is bugged.
		err = fmt.Errorf("internal parser error: parsed param mismatch. "+
			"%d display names, %d uris and %d param sets "+
			"in %s.",
			len(displayNames), len(uris), len(paramSets),
			headerText)
		return nil, err
	}

	// Build a slice of headers of the appropriate kind, populating them with the values parsed above.
	// It is assumed that all headers returned by ParseAddressValues are of the same kind,
	// although we do not check for this below.
	headers := make([]base.SipHeader, 0, 10)
	for idx := 0; idx < len(displayNames); idx++ {
		var header base.SipHeader
		if headerName == "to" || headerName == "t" {
			if idx > 0 {
				// Only a single To header is permitted in a SIP message.
				return nil,
					fmt.Errorf("Multiple to: headers in message:\n%s: %s",
						headerName, headerText)
			}
			switch uris[idx].(type) {
			case base.WildcardUri:
				// The Wildcard '*' URI is only permitted in Contact headers.
				err = fmt.Errorf("wildcard uri not permitted in to: "+
					"header: %s", headerText)
				return nil, err
			default:
				toHeader := base.ToHeader{displayNames[idx],
					uris[idx],
					paramSets[idx]}
				header = &toHeader
			}
		} else if headerName == "from" || headerName == "f" {
			if idx > 0 {
				// Only a single From header is permitted in a SIP message.
				return nil,
					fmt.Errorf("Multiple from: headers in message:\n%s: %s",
						headerName, headerText)
			}
			switch uris[idx].(type) {
			case base.WildcardUri:
				// The Wildcard '*' URI is only permitted in Contact headers.
				err = fmt.Errorf("wildcard uri not permitted in from: "+
					"header: %s", headerText)
				return nil, err
			default:
				fromHeader := base.FromHeader{displayNames[idx],
					uris[idx],
					paramSets[idx]}
				header = &fromHeader
			}
		} else if headerName == "contact" || headerName == "m" {
			switch uris[idx].(type) {
			case base.ContactUri:
				if uris[idx].(base.ContactUri).IsWildcard() {
					if displayNames[idx] != nil || len(paramSets[idx]) > 0 {
						// Wildcard headers do not contain display names or parameters.
						err = fmt.Errorf("wildcard contact header should contain only '*' in %s",
							headerText)
						return nil, err
					}
				}
				contactHeader := base.ContactHeader{displayNames[idx],
					uris[idx].(base.ContactUri),
					paramSets[idx]}
				header = &contactHeader
			default:
				// URIs in contact headers are restricted to being either SIP URIs or 'Contact: *'.
				return nil,
					fmt.Errorf("Uri %s not valid in Contact header. Must be SIP uri or '*'", uris[idx].String())
			}
		}

		headers = append(headers, header)
	}

	return headers, nil
}

// Parse a string representation of a CSeq header, returning a slice of at most one CSeq.
func parseCSeq(headerName string, headerText string) (
	headers []base.SipHeader, err error) {
	var cseq base.CSeq

	parts := splitByWhitespace(headerText)
	if len(parts) != 2 {
		err = fmt.Errorf("CSeq field should have precisely one whitespace section: '%s'",
			headerText)
		return
	}

	var seqno uint64
	seqno, err = strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return
	}

	if seqno > MAX_CSEQ {
		err = fmt.Errorf("invalid CSeq %d: exceeds maximum permitted value "+
			"2**31 - 1", seqno)
		return
	}

	cseq.SeqNo = uint32(seqno)
	cseq.MethodName = base.Method(strings.TrimSpace(parts[1]))

	if strings.Contains(string(cseq.MethodName), ";") {
		err = fmt.Errorf("unexpected ';' in CSeq body: %s", headerText)
		return
	}

	headers = []base.SipHeader{&cseq}

	return
}

// Parse a string representation of a Call-Id header, returning a slice of at most one CallId.
func parseCallId(headerName string, headerText string) (
	headers []base.SipHeader, err error) {
	headerText = strings.TrimSpace(headerText)
	var callId base.CallId = base.CallId(headerText)

	if strings.ContainsAny(string(callId), c_ABNF_WS) {
		err = fmt.Errorf("unexpected whitespace in CallId header body '%s'", headerText)
		return
	}
	if strings.Contains(string(callId), ";") {
		err = fmt.Errorf("unexpected semicolon in CallId header body '%s'", headerText)
		return
	}
	if len(string(callId)) == 0 {
		err = fmt.Errorf("empty Call-Id body")
		return
	}

	headers = []base.SipHeader{&callId}

	return
}

// Parse a string representation of a Via header, returning a slice of at most one ViaHeader.
// Note that although Via headers may contain a comma-separated list, RFC 3261 makes it clear that
// these should not be treated as separate logical Via headers, but as multiple values on a single
// Via header.
func parseViaHeader(headerName string, headerText string) (
	headers []base.SipHeader, err error) {
	sections := strings.Split(headerText, ",")
	var via base.ViaHeader = base.ViaHeader{}
	for _, section := range sections {
		var hop base.ViaHop
		parts := strings.Split(section, "/")

		if len(parts) < 3 {
			err = fmt.Errorf("not enough protocol parts in via header: '%s'",
				parts)
			return
		}

		parts[2] = strings.Join(parts[2:], "/")

		// The transport part ends when whitespace is reached, but may also start with
		// whitespace.
		// So the end of the transport part is the first whitespace char following the
		// first non-whitespace char.
		initialSpaces := len(parts[2]) - len(strings.TrimLeft(parts[2], c_ABNF_WS))
		sentByIdx := strings.IndexAny(parts[2][initialSpaces:], c_ABNF_WS) + initialSpaces + 1
		if sentByIdx == 0 {
			err = fmt.Errorf("expected whitespace after sent-protocol part "+
				"in via header '%s'", section)
			return
		} else if sentByIdx == 1 {
			err = fmt.Errorf("empty transport field in via header '%s'", section)
			return
		}

		hop.ProtocolName = strings.TrimSpace(parts[0])
		hop.ProtocolVersion = strings.TrimSpace(parts[1])
		hop.Transport = strings.TrimSpace(parts[2][:sentByIdx-1])

		if len(hop.ProtocolName) == 0 {
			err = fmt.Errorf("no protocol name provided in via header '%s'", section)
		} else if len(hop.ProtocolVersion) == 0 {
			err = fmt.Errorf("no version provided in via header '%s'", section)
		} else if len(hop.Transport) == 0 {
			err = fmt.Errorf("no transport provided in via header '%s'", section)
		}
		if err != nil {
			return
		}

		viaBody := parts[2][sentByIdx:]

		paramsIdx := strings.Index(viaBody, ";")
		var host string
		var port *uint16
		if paramsIdx == -1 {
			// There are no header parameters, so the rest of the Via body is part of the host[:post].
			host, port, err = sipuri.ParseHostPort(viaBody)
			hop.Host = host
			hop.Port = port
			if err != nil {
				return
			}
		} else {
			host, port, err = sipuri.ParseHostPort(viaBody[:paramsIdx])
			if err != nil {
				return
			}
			hop.Host = host
			hop.Port = port

			hop.Params, _, err = sipuri.ParseParams(viaBody[paramsIdx:],
				';', ';', 0, true, true)
		}
		via = append(via, &hop)
	}

	headers = []base.SipHeader{&via}
	return
}

// Parse a string representation of a Max-Forwards header into a slice of at most one MaxForwards header object.
func parseMaxForwards(headerName string, headerText string) (
	headers []base.SipHeader, err error) {
	var maxForwards base.MaxForwards
	var value uint64
	value, err = strconv.ParseUint(strings.TrimSpace(headerText), 10, 32)
	maxForwards = base.MaxForwards(value)

	headers = []base.SipHeader{&maxForwards}
	return
}

// Parse a string representation of a Content-Length header into a slice of at most one ContentLength header object.
func parseContentLength(headerName string, headerText string) (
	headers []base.SipHeader, err error) {
	var contentLength base.ContentLength
	var value uint64
	value, err = strconv.ParseUint(strings.TrimSpace(headerText), 10, 32)
	contentLength = base.ContentLength(value)

	headers = []base.SipHeader{&contentLength}
	return
}

// ParseAddressValues parses a comma-separated list of addresses, returning
// any display names and header params, as well as the SIP URIs themselves.
// ParseAddressValues is aware of < > bracketing and quoting, and will not
// break on commas within these structures.
func ParseAddressValues(addresses string) (
	displayNames []*string, uris []base.Uri,
	headerParams []map[string]*string,
	err error) {

	prevIdx := 0
	inBrackets := false
	inQuotes := false

	// Append a comma to simplify the parsing code; we split address sections
	// on commas, so use a comma to signify the end of the final address section.
	addresses = addresses + ","

	var prevChar rune
	for idx, char := range addresses {
		if char == '<' && !inQuotes {
			inBrackets = true
		} else if char == '>' && !inQuotes {
			inBrackets = false

			// display name can have escaped quotes
		} else if char == '"' && prevChar != '\\' {
			inQuotes = !inQuotes
		} else if !inQuotes && !inBrackets && char == ',' {

			displayName, uri, params, err := ParseAddressValue(addresses[prevIdx:idx])

			if err != nil {
				return nil, nil, nil, err
			}
			prevIdx = idx + 1

			displayNames = append(displayNames, displayName)
			uris = append(uris, uri)
			headerParams = append(headerParams, params)
		}
		prevChar = char
	}

	return displayNames, uris, headerParams, nil
}

// ParseAddressValue parses an address - such as from a From, To, or
// Contact header. It returns:
//   - a pointer to the display name (or nil if there was none present)
//   - a parsed SipUri object
//   - a map containing any header parameters present
//   - the error object
// See RFC 3261 section 20.10 for details on parsing an address.
// Note that this method will not accept a comma-separated list of addresses;
// addresses in that form should be handled by ParseAddressValues.
// In form: name-addr      =  [ display-name ] LAQUOT addr-spec RAQUOT
func ParseAddressValue(addressText string) (displayName *string, uri base.Uri, headerParams map[string]*string, err error) {

	if len(addressText) == 0 {
		err = fmt.Errorf("address-type header has empty body")
		return
	}

	addressTextCopy := addressText
	addressText = strings.TrimSpace(addressText)

	firstAngleBracket := findUnescaped(addressText, '<', quotes_delim)

	// if there is a bracket, a display name may be present
	if firstAngleBracket != -1 {
		// There is a display name present. Let's parse it.

		// display-name   =  *(token LWS)/ quoted-string
		if addressText[0] == '"' {
			// The display name is within quotations.
			addressText = addressText[1:]

			// find the next quote that isn't escaped
			var nextQuote = -1
			var firstBracket = -1

			var prevChar rune

			for i, v := range addressText {
				if v == '"' && prevChar != '\\' {
					nextQuote = i
					break
				} else if v == '<' && firstBracket == -1 {
					firstBracket = i
				}
				prevChar = v
			}

			if nextQuote == -1 {
				// if we have a bracket and don't have a quote try to insert it
				if firstBracket != -1 {
					nextQuote = firstBracket
				} else {
					// Unclosed quotes - parse error.
					err = fmt.Errorf("Unclosed quotes in header text: %s",
						addressTextCopy)
					return
				}
			}

			nameField := addressText[:nextQuote]
			displayName = &nameField
			addressText = addressText[nextQuote+1:]
		} else {
			// The display name is unquoted, so match until the LAQUOT
			// TODO: only allow valid token characters and LWS
			// *(token LWS)
			nameField := strings.TrimSpace(addressText[:firstAngleBracket])
			if nameField != "" {
				displayName = &nameField
			}
			addressText = addressText[firstAngleBracket:]
		}
	}

	// Work out where the SIP URI starts and ends.
	addressText = strings.TrimSpace(addressText)
	if len(addressText) < 1 {
		err = fmt.Errorf("Address text is too short")
		return
	}

	var endOfUri int
	var startOfParams int
	if addressText[0] != '<' {
		if displayName != nil {
			// The address must be in <angle brackets> if a display name is
			// present, so this is an invalid address line.
			err = fmt.Errorf("Invalid character '%c' following display "+
				"name in address line; expected '<': %s",
				addressText[0], addressTextCopy)
			return
		}

		endOfUri = strings.Index(addressText, ";")
		if endOfUri == -1 {
			endOfUri = len(addressText)
		}
		startOfParams = endOfUri

	} else {
		addressText = addressText[1:]
		endOfUri = strings.Index(addressText, ">")
		if endOfUri <= 0 {
			err = fmt.Errorf("'<' without closing '>' in address %s",
				addressTextCopy)
			return
		}
		startOfParams = endOfUri + 1

	}

	if len(addressText) < endOfUri {
		err = fmt.Errorf("Index out of bounds (%s) Length: %d Index %d", addressText, len(addressText), endOfUri)
		return
	}

	// Now parse the SIP URI.
	uri, err = sipuri.ParseUri(addressText[:endOfUri])
	if err != nil {
		return
	}

	if startOfParams >= len(addressText) {
		return
	}

	// Finally, parse any header parameters and then return.
	addressText = addressText[startOfParams:]
	headerParams, _, err = sipuri.ParseParams(addressText, ';', ';', ',', true, true)
	if err != nil {
		fmt.Printf("error!!!! (%s) %s\n", addressText, err)
		return nil, nil, nil, err
	}
	return
}

// Extract the next logical header line from the message.
// This may run over several actual lines; lines that start with whitespace are
// a continuation of the previous line.
// Therefore also return how many lines we consumed so the parent parser can
// keep track of progress through the message.
func getNextHeaderLine(contents []string) (headerText string, consumed int) {
	if len(contents) == 0 {
		return
	}
	if len(contents[0]) == 0 {
		return
	}

	var buffer bytes.Buffer
	buffer.WriteString(contents[0])

	for consumed = 1; consumed < len(contents); consumed++ {
		firstChar, _ := utf8.DecodeRuneInString(contents[consumed])
		if !unicode.IsSpace(firstChar) {
			break
		} else if len(contents[consumed]) == 0 {
			break
		}

		buffer.WriteString(" " + strings.TrimSpace(contents[consumed]))
	}

	headerText = buffer.String()
	return
}

// A delimiter is any pair of characters used for quoting text (i.e. bulk escaping literals).
type delimiter struct {
	start rune
	end   rune
}

// Define common quote characters needed in parsing.
var quotes_delim = delimiter{'"', '"'}

// Find the first instance of the target in the given text which is not enclosed in any delimiters
// from the list provided.
func findUnescaped(text string, target rune, delims ...delimiter) int {
	return findAnyUnescaped(text, string(target), delims...)
}

// Find the first instance of any of the targets in the given text that are not enclosed in any delimiters
// from the list provided.
func findAnyUnescaped(text string, targets string, delims ...delimiter) int {
	escaped := false
	var endEscape rune

	endChars := make(map[rune]rune)
	for _, delim := range delims {
		endChars[delim.start] = delim.end
	}

	var prevChar rune
	for idx, currentChar := range text {
		if !escaped && strings.Contains(targets, string(currentChar)) {
			return idx
		}

		if escaped {
			escaped = (currentChar != endEscape && prevChar != '\\')
			prevChar = rune(text[idx])
			continue
		}

		endEscape, escaped = endChars[currentChar]
		prevChar = currentChar
	}

	return -1
}

// Splits the given string into sections, separated by one or more characters
// from c_ABNF_WS.
func splitByWhitespace(text string) []string {
	var buffer bytes.Buffer
	var inString bool = true
	result := make([]string, 0)

	for _, char := range text {
		s := string(char)
		if strings.Contains(c_ABNF_WS, s) {
			if inString {
				// First whitespace char following text; flush buffer to the results array.
				result = append(result, buffer.String())
				buffer.Reset()
			}
			inString = false
		} else {
			buffer.WriteString(s)
			inString = true
		}
	}

	if buffer.Len() > 0 {
		result = append(result, buffer.String())
	}

	return result
}
