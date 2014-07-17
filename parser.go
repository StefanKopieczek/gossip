package gossip

import "bytes"
import "fmt"
import "strings"
import "strconv"
import "unicode"
import "unicode/utf8"

// The whitespace characters recognised by the Augmented Backus-Naur Form syntax
// that SIP uses (RFC 3261 S.25).
const ABNF_WS = " \t"

// The maximum permissible CSeq number in a SIP message (2**31 - 1).
// C.f. RFC 3261 S. 8.1.1.5.
const MAX_CSEQ = 2147483647

// A MessageParser converts the raw bytes of a SIP message into an internal gossip.SipMessage.
// This will be either a Request or a Response struct.
type MessageParser interface {
	// ParseMessage converts the given raw message data into either a Request or a Response.
	ParseMessage(rawData []byte) (SipMessage, error)

	// Register a parser for the given header type.
	// This allows you to add support for new header types, or override existing parsing behaviour.
	// The headerName should be a string of the form 'from', or 'via'. Case is irrelevant.
	SetHeaderParser(headerName string, headerParser HeaderParser)
}

// A HeaderParser is any function that turns raw header data into one or more SipHeader objects.
// The HeaderParser will receive arguments of the form ("max-forwards", "70").
// It should return a slice of headers, which should have length > 1 unless it also returns an error.
type HeaderParser func(headerName string, headerData string) (
	headers []SipHeader, err error)

type parserImpl struct {
	headerParsers map[string]HeaderParser
}

// Create a new MessageParser.
func NewMessageParser() MessageParser {
	var parser parserImpl
	parser.headerParsers = make(map[string]HeaderParser)
	headerParsers := map[string]HeaderParser{
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
	for headerName, headerParser := range headerParsers {
		parser.SetHeaderParser(headerName, headerParser)
	}

	return &parser
}

// See MessageParser.SetHeaderParser.
func (parser *parserImpl) SetHeaderParser(headerName string,
	headerParser HeaderParser) {
	headerName = strings.ToLower(headerName)
	parser.headerParsers[headerName] = headerParser
}

// See MessageParser.ParseMessage.
func (parser *parserImpl) ParseMessage(rawData []byte) (SipMessage, error) {
	contents := strings.Split(string(rawData), "\r\n")
	if isRequest(contents) {
		return parser.parseRequest(contents)
	} else if isResponse(contents) {
		return parser.parseResponse(contents)
	}

	return nil, fmt.Errorf("transmission beginnng '%s' is not a SIP message", contents[0])
}

// Heuristic to determine if the given transmission looks like a SIP request.
// It is guaranteed that any RFC3261-compliant request will pass this test,
// but invalid messages may not necessarily be rejected.
func isRequest(contents []string) bool {
	requestLine := contents[0]

	// SIP request lines contain precisely two spaces.
	if strings.Count(requestLine, " ") != 2 {
		return false
	}

	// Check that the version string starts with SIP.
	versionString := strings.ToUpper(strings.Split(requestLine, " ")[2])
	return versionString[:3] == "SIP"
}

// Heuristic to determine if the given transmission looks like a SIP response.
// It is guaranteed that any RFC3261-compliant response will pass this test,
// but invalid messages may not necessarily be rejected.
func isResponse(contents []string) bool {
	statusLine := contents[0]

	// SIP status lines contain at least two spaces.
	if strings.Count(statusLine, " ") < 2 {
		return false
	}

	// Check that the version string starts with SIP.
	versionString := statusLine[:strings.Index(statusLine, " ")]
	return versionString[:3] == "SIP"
}

func (parser *parserImpl) parseRequest(contents []string) (*Request, error) {
	var request Request
	var err error

	// Parse the Request Line of the message.
	request.Method, request.Recipient, request.SipVersion, err = parseRequestLine(contents[0])
	if err != nil {
		return nil, err
	}

	// Parse all headers on the message.
	// Record how many lines are consumed so that we may identify the start of the application data.
	var consumed int
	request.Headers, consumed, err = parser.parseHeaders(contents[1:])
	if err != nil {
		return nil, err
	}

	// If the request contains no application data then it should end immediately with double-CRLF.
	// We're splitting on CRLF, so there should be at least two more lines at this stage; if there
	// are exactly two we've reached the end of the message.
	if len(contents) == consumed+2 {
		return &request, err
	} else if len(contents) == consumed+1 {
		err = fmt.Errorf("Request beginning '%s' has no CRLF at end of headers",
			contents[0])
		return nil, err
	} else if len(contents) <= consumed {
		err = fmt.Errorf("Internal error: consumed %d lines processing request "+
			"beginning '%s' but message length was %d lines!",
			consumed, len(contents), contents[0])
		return nil, err
	}

	bodyText := strings.Join(contents[2+consumed:], "\r\n")
	request.Body = &bodyText

	return &request, err
}

func (parser *parserImpl) parseResponse(contents []string) (*Response, error) {
	var response Response
	var err error

	// Parse the status line of the message.
	response.SipVersion, response.StatusCode, response.Reason, err = parseStatusLine(contents[0])
	if err != nil {
		return nil, err
	}

	// Parse all headers on the message.
	// Record how many lines are consumed so that we can identify the start of the application data.
	var consumed int
	response.Headers, consumed, err = parser.parseHeaders(contents[1:])
	if err != nil {
		return nil, err
	}

	// If the request contains no application data then it should end immediately with double-CRLF.
	// We're splitting on CRLF, so there should be at least two more lines at this stage; if there
	// are exactly two we've reached the end of the message.
	if len(contents) == consumed+2 {
		return &response, err
	} else if len(contents) == consumed+1 {
		err = fmt.Errorf("Response beginning '%s' has no CRLF at end of headers", contents[0])
		return nil, err
	} else if len(contents) <= consumed {
		err = fmt.Errorf("Internal error: consumed %d lines processing response "+
			"beginning '%s' but message length was %d lines!",
			consumed, len(contents), contents[0])
		return nil, err
	}

	bodyText := strings.Join(contents[2+consumed:], "\r\n")
	response.Body = &bodyText

	return &response, err
}

// Parse the first line of a SIP request, e.g:
//   INVITE bob@example.com SIP/2.0
//   REGISTER jane@telco.com SIP/1.0
func parseRequestLine(requestLine string) (
	method Method, recipient Uri, sipVersion string, err error) {
	parts := strings.Split(requestLine, " ")
	if len(parts) != 3 {
		err = fmt.Errorf("request line should have 2 spaces: '%s'", requestLine)
		return
	}

	method = Method(strings.ToUpper(parts[0]))
	recipient, err = ParseUri(parts[1])
	sipVersion = parts[2]

    switch recipient.(type) {
    case *WildcardUri:
        err = fmt.Errorf("wildcard URI '*' not permitted in request line: '%s'", requestLine)
    }

	return
}

// Parse the first line of a SIP response, e.g:
//   SIP/2.0 200 OK
//   SIP/1.0 403 Forbidden
func parseStatusLine(statusLine string) (
	sipVersion string, statusCode uint8, reasonPhrase string, err error) {
	parts := strings.Split(statusLine, " ")
	if len(parts) < 3 {
		err = fmt.Errorf("status line has too few spaces: '%s'", statusLine)
		return
	}

	sipVersion = parts[0]
	statusCodeRaw, err := strconv.ParseUint(parts[1], 10, 8)
	statusCode = uint8(statusCodeRaw)
	reasonPhrase = strings.Join(parts[2:], "")

	return
}

// parseUri converts a string representation of a URI into a Uri object.
// If the URI is malformed, or the URI schema is not recognised, an error is returned.
// URIs have the general form of schema:address.
func ParseUri(uriStr string) (uri Uri, err error) {
	if strings.TrimSpace(uriStr) == "*" {
		// Wildcard '*' URI used in the Contact headers of REGISTERs when unregistering.
		return &WildcardUri{}, nil
	}

	colonIdx := strings.Index(uriStr, ":")
	if colonIdx == -1 {
		err = fmt.Errorf("no ':' in URI %s", uriStr)
		return
	}

	switch strings.ToLower(uriStr[:colonIdx]) {
	case "sip":
		var sipUri SipUri
		sipUri, err = ParseSipUri(uriStr)
		uri = &sipUri
	case "sips":
		// SIPS URIs have the same form as SIP uris, so we use the same parser.
		var sipUri SipUri
		sipUri, err = ParseSipUri(uriStr)
		uri = &sipUri
	default:
		err = fmt.Errorf("Unsupported URI schema %s", uriStr[:colonIdx])
	}

	return
}

// ParseSipUri converts a string representation of a SIP or SIPS URI into a SipUri object.
func ParseSipUri(uriStr string) (uri SipUri, err error) {
	// Store off the original URI in case we need to print it in an error.
	uriStrCopy := uriStr

	// URI should start 'sip' or 'sips'. Check the first 3 chars.
	if strings.ToLower(uriStr[:3]) != "sip" {
		err = fmt.Errorf("invalid SIP uri protocol name in '%s'", uriStrCopy)
		return
	}
	uriStr = uriStr[3:]

	if strings.ToLower(uriStr[0:1]) == "s" {
		// URI started 'sips', so it's encrypted.
		uri.IsEncrypted = true
		uriStr = uriStr[1:]
	}

	// The 'sip' or 'sips' protocol name should be followed by a ':' character.
	if uriStr[0] != ':' {
		err = fmt.Errorf("no ':' after protocol name in SIP uri '%s'", uriStrCopy)
		return
	}
	uriStr = uriStr[1:]

	// SIP URIs may contain a user-info part, ending in a '@'.
	// This is the only place '@' may occur, so we can use it to check for the
	// existence of a user-info part.
	endOfUserInfoPart := strings.Index(uriStr, "@")
	if endOfUserInfoPart != -1 {
		// A user-info part is present. These take the form:
		//     user [ ":" password ] "@"
		endOfUsernamePart := strings.Index(uriStr, ":")
		if endOfUsernamePart > endOfUserInfoPart {
			endOfUsernamePart = -1
		}

		if endOfUsernamePart == -1 {
			// No password component; the whole of the user-info part before
			// the '@' is a username.
			user := uriStr[:endOfUserInfoPart]
			uri.User = &user
		} else {
			user := uriStr[:endOfUsernamePart]
			pwd := uriStr[endOfUsernamePart+1 : endOfUserInfoPart]
			uri.User = &user
			uri.Password = &pwd
		}
		uriStr = uriStr[endOfUserInfoPart+1:]
	}

	// A ';' indicates the beginning of a URI params section, and the end of the URI itself.
	endOfUriPart := strings.Index(uriStr, ";")
	if endOfUriPart == -1 {
		// There are no URI parameters, but there might be header parameters (introduced by '?').
		endOfUriPart = strings.Index(uriStr, "?")
	}
	if endOfUriPart == -1 {
		// There are no parameters at all. The URI ends after the host[:port] part.
		endOfUriPart = len(uriStr)
	}

	uri.Host, uri.Port, err = parseHostPort(uriStr[:endOfUriPart])
	uriStr = uriStr[endOfUriPart:]
	if err != nil || len(uriStr) == 0 {
		return
	}

	// Now parse any URI parameters.
	// These are key-value pairs separated by ';'.
	// They end at the end of the URI, or at the start of any URI headers
	// which may be present (denoted by an initial '?').
	var uriParams map[string]*string
	var n int
	if uriStr[0] == ';' {
		uriParams, n, err = parseParams(uriStr, ';', ';', '?', true, true)
		if err != nil {
			return
		}
	} else {
		uriParams, n = map[string]*string{}, 0
	}
	uri.UriParams = uriParams
	uriStr = uriStr[n:]

	// Finally parse any URI headers.
	// These are key-value pairs, starting with a '?' and separated by '&'.
	var headers map[string]*string
	headers, n, err = parseParams(uriStr, '?', '&', 0, true, false)
	if err != nil {
		return
	}
	uri.Headers = headers
	uriStr = uriStr[n:]
	if len(uriStr) > 0 {
		err = fmt.Errorf("internal error: parse of SIP uri ended early! '%s'",
			uriStrCopy)
		return // Defensive return
	}

	return
}

// Parse a text representation of a host[:port] pair.
// The port may or may not be present, so we represent it with a *uint16,
// and return 'nil' if no port was present.
func parseHostPort(rawText string) (host string, port *uint16, err error) {
	colonIdx := strings.Index(rawText, ":")
	if colonIdx == -1 {
		host = rawText
		return
	}

	// Surely there must be a better way..!
	var portRaw64 uint64
	var portRaw16 uint16
	host = rawText[:colonIdx]
	portRaw64, err = strconv.ParseUint(rawText[colonIdx+1:], 10, 16)
	portRaw16 = uint16(portRaw64)
	port = &portRaw16

	return
}

// General utility method for parsing 'key=value' parameters.
// Takes a string (source), ensures that it begins with the 'start' character provided,
// and then parses successive key/value pairs separated with 'sep',
// until either 'end' is reached or there are no characters remaining.
// A map of keys to values will be returned, along with the number of characters consumed.
// Provide 0 for start or end to indicate that there is no starting/ending delimiter.
// If quoteValues is true, values can be enclosed in double-quotes which will be validated by the
// parser and omitted from the returned map.
// If permitSingletons is true, keys with no values are permitted.
// These will result in a nil value in the returned map.
func parseParams(source string,
	start uint8, sep uint8, end uint8,
	quoteValues bool, permitSingletons bool) (
	params map[string]*string, consumed int, err error) {

	params = make(map[string]*string)

	if len(source) == 0 {
		// Key-value section is completely empty; return defaults.
		return
	}

	// Ensure the starting character is correct.
	if start != 0 {
		if source[0] != start {
			err = fmt.Errorf("expected %c at start of key-value section; got %c. section was %s",
				start, source[0], source)
			return
		}
		consumed++
	}

	// Statefully parse the given string one character at a time.
	var buffer bytes.Buffer
	var key string
	parsingKey := true // false implies we are parsing a value
	inQuotes := false
parseLoop:
	for ; consumed < len(source); consumed++ {
		switch source[consumed] {
		case end:
			if inQuotes {
				// We read an end character, but since we're inside quotations we should
				// treat it as a literal part of the value.
				buffer.WriteString(string(end))
				continue
			}

			break parseLoop

		case sep:
			if inQuotes {
				// We read a separator character, but since we're inside quotations
				// we should treat it as a literal part of the value.
				buffer.WriteString(string(sep))
				continue
			}
			if parsingKey && permitSingletons {
				params[buffer.String()] = nil
			} else if parsingKey {
				err = fmt.Errorf("Singleton param '%s' when parsing params which disallow singletons: \"%s\"",
					buffer.String(), source)
				return
			} else {
				value := buffer.String()
				params[key] = &value
			}
			buffer.Reset()
			parsingKey = true

		case '"':
			if !quoteValues {
				// We hit a quote character, but since quoting is turned off we treat it as a literal.
				buffer.WriteString("\"")
				continue
			}

			if parsingKey {
				// Quotes are never allowed in keys.
				err = fmt.Errorf("Unexpected '\"' in parameter key in params \"%s\"", source)
				return
			}

			if !inQuotes && buffer.Len() != 0 {
				// We hit an initial quote midway through a value; that's not allowed.
				err = fmt.Errorf("unexpected '\"' in params \"%s\"", source)
				return
			}

			if inQuotes &&
				consumed != len(source)-1 &&
				source[consumed+1] != sep {
				// We hit an end-quote midway through a value; that's not allowed.
				err = fmt.Errorf("unexpected character %c after quoted param in \"%s\"",
					source[consumed+1], source)

				return
			}

			inQuotes = !inQuotes

		case '=':
			if buffer.Len() == 0 {
				err = fmt.Errorf("Key of length 0 in params \"%s\"", source)
				return
			}
			if !parsingKey {
				err = fmt.Errorf("Unexpected '=' char in value token: \"%s\"", source)
				return
			}
			key = buffer.String()
			buffer.Reset()
			parsingKey = false

		default:
			if !inQuotes && strings.Contains(ABNF_WS, string(source[consumed])) {
				// Skip unquoted whitespace.
				continue
			}

			buffer.WriteString(string(source[consumed]))
		}
	}

	// The param string has ended. Check that it ended in a valid place, and then store off the
	// contents of the buffer.
	if inQuotes {
		err = fmt.Errorf("Unclosed quotes in parameter string: %s", source)
	} else if parsingKey && permitSingletons {
		params[buffer.String()] = nil
	} else if parsingKey {
		err = fmt.Errorf("Singleton param '%s' when parsing params which disallow singletons: \"%s\"",
			buffer.String(), source)
	} else {
		value := buffer.String()
		params[key] = &value
	}
	return
}

// Extract the headers from a string representation of a SIP message.
// Return the parsed headers, the number of lines consumed, and any error.
func (parser *parserImpl) parseHeaders(contents []string) (
	headers []SipHeader, consumed int, err error) {
	headers = make([]SipHeader, 0)
	for {
		// Separate out the lines corresponding to the first header.
		headerText, lines := getNextHeaderLine(contents[consumed:])
		if lines == 0 {
			// End of header section
			return
		}

		// Parse this header block, producing one or more logical headers.
		// (SIP headers of the same type can be expressed as a comma-separated argument list).
		var someHeaders []SipHeader
		someHeaders, err = parser.parseHeader(headerText)
		if err != nil {
			return
		}
		headers = append(headers, someHeaders...)
		consumed += lines
	}

	return
}

// Parse a header string, producing one or more SipHeader objects.
// (SIP messages containing multiple headers of the same type can express them as a
// single header containing a comma-separated argument list).
func (parser *parserImpl) parseHeader(headerText string) (
	headers []SipHeader, err error) {
	colonIdx := strings.Index(headerText, ":")
	if colonIdx == -1 {
		err = fmt.Errorf("Field name with no value in header: %s", headerText)
		return
	}

	fieldName := strings.ToLower(strings.TrimSpace(headerText[:colonIdx]))
	fieldText := strings.TrimSpace(headerText[colonIdx+1:])
	if headerParser, ok := parser.headerParsers[fieldName]; ok {
		// We have a registered parser for this header type - use it.
		return headerParser(fieldName, fieldText)
	} else {
		// We have no registered parser for this header type,
		// so we encapsulate the header data in a GenericHeader struct.
		header := GenericHeader{fieldName, fieldText}
		headers = []SipHeader{&header}
		return
	}

	return
}

// Parse a To, From or Contact header line, producing one or more logical SipHeaders.
func parseAddressHeader(headerName string, headerText string) (
	headers []SipHeader, err error) {
	switch headerName {
	case "to", "from", "contact", "t", "f", "m":
		var displayNames []*string
		var uris []Uri
		var paramSets []map[string]*string

		// Perform the actual parsing. The rest of this method is just typeclass bookkeeping.
		displayNames, uris, paramSets, err = parseAddressValues(headerText)

		if err != nil {
			return
		}
		if len(displayNames) != len(uris) || len(uris) != len(paramSets) {
			// This shouldn't happen unless parseAddressValues is bugged.
			err = fmt.Errorf("internal parser error: parsed param mismatch. "+
				"%d display names, %d uris and %d param sets "+
				"in %s.",
				len(displayNames), len(uris), len(paramSets),
				headerText)
			return
		}

		// Build a slice of headers of the appropriate kind, populating them with the values parsed above.
		// It is assumed that all headers returned by parseAddressValues are of the same kind,
		// although we do not check for this below.
		for idx := 0; idx < len(displayNames); idx++ {
			var header SipHeader
			if headerName == "to" || headerName == "t" {
				if idx > 0 {
					// Only a single To header is permitted in a SIP message.
					return nil,
						fmt.Errorf("Multiple to: headers in message:\n%s: %s",
							headerName, headerText)
				}
				switch uris[idx].(type) {
				case *WildcardUri:
					// The Wildcard '*' URI is only permitted in Contact headers.
					err = fmt.Errorf("wildcard uri not permitted in to: "+
						"header: %s", headerText)
					return
				default:
					toHeader := ToHeader{displayNames[idx],
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
				case *WildcardUri:
					// The Wildcard '*' URI is only permitted in Contact headers.
					err = fmt.Errorf("wildcard uri not permitted in from: "+
						"header: %s", headerText)
					return
				default:
					fromHeader := FromHeader{displayNames[idx],
						uris[idx],
						paramSets[idx]}
					header = &fromHeader
				}
			} else if headerName == "contact" || headerName == "m" {
				switch uris[idx].(type) {
				case ContactUri:
					if uris[idx].(ContactUri).IsWildcard() {
						if displayNames[idx] != nil || len(paramSets[idx]) > 0 {
							// Wildcard headers do not contain display names or parameters.
							err = fmt.Errorf("wildcard contact header should contain only '*' in %s",
								headerText)
							return
						}
					}
					contactHeader := ContactHeader{displayNames[idx],
						uris[idx].(ContactUri),
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
	}

	return
}

// Parse a string representation of a CSeq header, returning a slice of at most one CSeq.
func parseCSeq(headerName string, headerText string) (
	headers []SipHeader, err error) {
	var cseq CSeq

	parts := strings.Split(headerText, " ")
	if len(parts) != 2 {
		err = fmt.Errorf("CSeq field should have precisely one space: '%s'",
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
	cseq.MethodName = Method(strings.TrimSpace(parts[1]))

	if strings.Contains(string(cseq.MethodName), ";") {
		err = fmt.Errorf("unexpected ';' in CSeq body: %s", headerText)
		return
	}

	headers = []SipHeader{&cseq}

	return
}

// Parse a string representation of a Call-Id header, returning a slice of at most one CallId.
func parseCallId(headerName string, headerText string) (
	headers []SipHeader, err error) {
	headerText = strings.TrimSpace(headerText)
	var callId CallId = CallId(headerText)

	if strings.ContainsAny(string(callId), ABNF_WS) {
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

	headers = []SipHeader{&callId}

	return
}

// Parse a string representation of a Via header, returning a slice of at most one ViaHeader.
// Note that although Via headers may contain a comma-separated list, RFC 3261 makes it clear that
// these should not be treated as separate logical Via headers, but as multiple values on a single
// Via header.
func parseViaHeader(headerName string, headerText string) (
	headers []SipHeader, err error) {
	sections := strings.Split(headerText, ",")
	var via ViaHeader = ViaHeader{}
	for _, section := range sections {
		var entry ViaHop
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
		initialSpaces := len(parts[2]) - len(strings.TrimLeft(parts[2], ABNF_WS))
		sentByIdx := strings.IndexAny(parts[2][initialSpaces:], ABNF_WS) + initialSpaces + 1
		if sentByIdx == 0 {
			err = fmt.Errorf("expected whitespace after sent-protocol part "+
				"in via header '%s'", section)
			return
		} else if sentByIdx == 1 {
			err = fmt.Errorf("empty transport field in via header '%s'", section)
			return
		}

		entry.protocolName = strings.TrimSpace(parts[0])
		entry.protocolVersion = strings.TrimSpace(parts[1])
		entry.transport = strings.TrimSpace(parts[2][:sentByIdx-1])

		if len(entry.protocolName) == 0 {
			err = fmt.Errorf("no protocol name provided in via header '%s'", section)
		} else if len(entry.protocolVersion) == 0 {
			err = fmt.Errorf("no version provided in via header '%s'", section)
		} else if len(entry.transport) == 0 {
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
			host, port, err = parseHostPort(viaBody)
			entry.host = host
			entry.port = port
			if err != nil {
				return
			}
		} else {
			host, port, err = parseHostPort(viaBody[:paramsIdx])
			if err != nil {
				return
			}
			entry.host = host
			entry.port = port

			entry.params, _, err = parseParams(viaBody[paramsIdx:],
				';', ';', 0, true, true)
		}
		via = append(via, &entry)
	}

	headers = []SipHeader{&via}
	return
}

// Parse a string representation of a Max-Forwards header into a slice of at most one MaxForwards header object.
func parseMaxForwards(headerName string, headerText string) (
	headers []SipHeader, err error) {
	var maxForwards MaxForwards
	var value uint64
	value, err = strconv.ParseUint(strings.TrimSpace(headerText), 10, 32)
	maxForwards = MaxForwards(value)

	headers = []SipHeader{&maxForwards}
	return
}

// Parse a string representation of a Content-Length header into a slice of at most one ContentLength header object.
func parseContentLength(headerName string, headerText string) (
	headers []SipHeader, err error) {
	var contentLength ContentLength
	var value uint64
	value, err = strconv.ParseUint(strings.TrimSpace(headerText), 10, 32)
	contentLength = ContentLength(value)

	headers = []SipHeader{&contentLength}
	return
}

// parseAddressValues parses a comma-separated list of addresses, returning
// any display names and header params, as well as the SIP URIs themselves.
// parseAddressValues is aware of < > bracketing and quoting, and will not
// break on commas within these structures.
func parseAddressValues(addresses string) (
	displayNames []*string, uris []Uri,
	headerParams []map[string]*string,
	err error) {

	prevIdx := 0
	inBrackets := false
	inQuotes := false

	// Append a comma to simplify the parsing code; we split address sections
	// on commas, so use a comma to signify the end of the final address section.
	addresses = addresses + ","

	for idx, char := range addresses {
		if char == '<' && !inQuotes {
			inBrackets = true
		} else if char == '>' && !inQuotes {
			inBrackets = false
		} else if char == '"' {
			inQuotes = !inQuotes
		} else if !inQuotes && !inBrackets && char == ',' {
			var displayName *string
			var uri Uri
			var params map[string]*string
			displayName, uri, params, err =
				parseAddressValue(addresses[prevIdx:idx])
			if err != nil {
				return
			}
			prevIdx = idx + 1

			displayNames = append(displayNames, displayName)
			uris = append(uris, uri)
			headerParams = append(headerParams, params)
		}
	}

	return
}

// parseAddressValue parses an address - such as from a From, To, or
// Contact header. It returns:
//   - a pointer to the display name (or nil if there was none present)
//   - a parsed SipUri object
//   - a map containing any header parameters present
//   - the error object
// See RFC 3261 section 20.10 for details on parsing an address.
// Note that this method will not accept a comma-separated list of addresses;
// addresses in that form should be handled by parseAddressValues.
func parseAddressValue(addressText string) (
	displayName *string, uri Uri,
	headerParams map[string]*string,
	err error) {

	if len(addressText) == 0 {
		err = fmt.Errorf("address-type header has empty body")
		return
	}

	addressTextCopy := addressText
	addressText = strings.TrimSpace(addressText)

	firstAngleBracket := findUnescaped(addressText, '<', quotes_delim)
	firstSpace := findAnyUnescaped(addressText, ABNF_WS, quotes_delim, angles_delim)
	if firstAngleBracket != -1 && firstSpace != -1 &&
		firstSpace < firstAngleBracket {
		// There is a display name present. Let's parse it.
		if addressText[0] == '"' {
			// The display name is within quotations.
			addressText = addressText[1:]
			nextQuote := strings.Index(addressText, "\"")

			if nextQuote == -1 {
				// Unclosed quotes - parse error.
				err = fmt.Errorf("Unclosed quotes in header text: %s",
					addressTextCopy)
				return
			}

			nameField := addressText[:nextQuote]
			displayName = &nameField
			addressText = addressText[nextQuote+1:]
		} else {
			// The display name is unquoted, so match until the next whitespace
			// character.
			nameField := addressText[:firstSpace]
			displayName = &nameField
			addressText = addressText[firstSpace+1:]
		}
	}

	// Work out where the SIP URI starts and ends.
	addressText = strings.TrimSpace(addressText)
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
		if endOfUri == 0 {
			err = fmt.Errorf("'<' without closing '>' in address %s",
				addressTextCopy)
			return
		}
		startOfParams = endOfUri + 1

	}

	// Now parse the SIP URI.
	uri, err = ParseUri(addressText[:endOfUri])
	if err != nil {
		return
	}

	if startOfParams >= len(addressText) {
		return
	}

	// Finally, parse any header parameters and then return.
	addressText = addressText[startOfParams:]
	headerParams, _, err = parseParams(addressText, ';', ';', ',', true, true)
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
	start uint8
	end   uint8
}

// Define common quote characters needed in parsing.
var quotes_delim = delimiter{'"', '"'}
var angles_delim = delimiter{'<', '>'}

// Find the first instance of the target in the given text which is not enclosed in any delimiters
// from the list provided.
func findUnescaped(text string, target uint8, delims ...delimiter) int {
	return findAnyUnescaped(text, string(target), delims...)
}

// Find the first instance of any of the targets in the given text that are not enclosed in any delimiters
// from the list provided.
func findAnyUnescaped(text string, targets string, delims ...delimiter) int {
	escaped := false
	var endEscape uint8 = 0

	endChars := make(map[uint8]uint8)
	for _, delim := range delims {
		endChars[delim.start] = delim.end
	}

	for idx := 0; idx < len(text); idx++ {
		if !escaped && strings.Contains(targets, string(text[idx])) {
			return idx
		}

		if escaped {
			escaped = (text[idx] != endEscape)
			continue
		} else {
			endEscape, escaped = endChars[text[idx]]
		}
	}

	return -1
}
