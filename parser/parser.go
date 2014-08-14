package parser

import (
    "github.com/stefankopieczek/gossip/base"
    "github.com/stefankopieczek/gossip/log"
    "github.com/stefankopieczek/gossip/utils"
)

import (
    "bytes"
    "fmt"
    "strings"
    "strconv"
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

    // TODO: Parser.Stop()
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
    p := parser{streamed:streamed}

    // Configure the parser with the standard set of header parsers.
    p.headerParsers = make(map[string]HeaderParser)
	for headerName, headerParser := range defaultHeaderParsers() {
		p.SetHeaderParser(headerName, headerParser)
	}

    p.input = make(chan string, c_INPUT_CHAN_SIZE)
    p.output = output
    p.errs = errs

    if !streamed {
        // If we're not in streaming mode, set up a channel so the Write method can pass calculated body lengths to the parser.
        p.bodyLengths.Init()
    }

    // Continually chunk input into individual lines, and then feed them into the parser proper.
    lines := make(chan string)
    go pipeLines(p.input, lines)

    // Wait for input a line at a time, and produce SipMessages to send down p.output.
    go p.parse(lines, streamed)

	return &p
}

type parser struct {
    headerParsers map[string]HeaderParser
    streamed bool
    input chan string
    bodyLengths utils.ElasticChan
    output chan<- base.SipMessage
    errs chan<- error
    terminalErr error
}

func (p *parser) Write(data []byte) (n int, err error) {
    if p.terminalErr != nil {
        // The parser has stopped due to a terminal error. Return it.
        log.Fine("Parser %p ignores %d new bytes due to previous terminal error: %s", p, len(data), p.terminalErr.Error())
        return 0, p.terminalErr
    }

    if !p.streamed {
        p.bodyLengths.In <- getBodyLength(data)
    }

    p.input <- string(data)
    return len(data), nil
}

// Consume input lines one at a time, producing base.SipMessage objects and sending them down p.output.
func (p *parser) parse(lines <- chan string, requireContentLength bool) {
    var message base.SipMessage

    for {
        // Parse the StartLine.
        startLine := <- lines
        if isRequest(startLine) {
            var request base.Request
            request.Method, request.Recipient, request.SipVersion, p.terminalErr = parseRequestLine(startLine)
            message = &request
        } else if isResponse(startLine) {
            var response base.Response
            response.SipVersion, response.StatusCode, response.Reason, p.terminalErr = parseStatusLine(startLine)
            message = &response
        } else {
            p.terminalErr = fmt.Errorf("transmission beginning '%s' is not a SIP message", startLine)
        }

        if p.terminalErr != nil {
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
            line := <- lines
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

        // Store the headers in the message object - bit of static-typing busywork here.
        switch message.(type) {
        case *base.Request:
            message.(*base.Request).Headers = headers
        case *base.Response:
            message.(*base.Response).Headers = headers
        default:
            log.Severe("Internal error - message %s is neither a request type nor a response type", message.Short())
        }

        var contentLength int

        // Determine the length of the body, so we know when to stop parsing this message.
        if p.streamed {
            // Use the content-length header to identify the end of the message.
            contentLengthHeaders := message.HeadersByName("Content-Length")
            if len(contentLengthHeaders) == 0 {
                p.terminalErr = fmt.Errorf("Missing required content-length header on message %s", message.Short())
                p.errs <- p.terminalErr
                break
            } else if len(contentLengthHeaders) > 1 {
                var errbuf bytes.Buffer
                errbuf.WriteString("Multiple content-length headers on message ")
                errbuf.WriteString(message.Short())
                errbuf.WriteString(":\n")
                for _, header := range(contentLengthHeaders) {
                    errbuf.WriteString("\t")
                    errbuf.WriteString(header.String())
                }
                p.terminalErr = fmt.Errorf(errbuf.String())
                p.errs <- p.terminalErr
                break
            }

            contentLength = int(*(contentLengthHeaders[0].(*base.ContentLength)))
        } else {
            // We're not in streaming mode, so the Write method should have calculated the length of the body for us.
            contentLength = (<- p.bodyLengths.Out).(int)
        }

        // Now just copy off lines into the body until we've met the required content length.
        // Since the message should end with a CRLF, we'll always need an integer number of lines.
        var line string
        for buffer.Len() < contentLength {
            line = <- lines
            buffer.WriteString(line)
            buffer.WriteString("\r\n")
        }

        result := buffer.String()
        if len(result) > 2 {
            result = result[:len(result)-2]
        }

        if len(result) != contentLength {
            p.errs <- fmt.Errorf("Final line of message %s was unexpectedly long: '%s'", message.Short(), line)
        }

        switch message.(type) {
        case *base.Request:
            message.(*base.Request).Body = result
        case *base.Response:
            message.(*base.Response).Body = result
        default:
            log.Severe("Internal error - message %s is neither a request type nor a response type", message.Short())
        }
        p.output <- message
    }

    return
}

// Implements ParserFactory.SetHeaderParser.
func (p *parser) SetHeaderParser(headerName string, headerParser HeaderParser) {
	headerName = strings.ToLower(headerName)
	p.headerParsers[headerName] = headerParser
}

func pipeLines(textIn <- chan string, linesOut chan<- string) {
    defer close(linesOut)

    var buffer bytes.Buffer
    toSend := make([]string, 0)

    // Inline the function for handling input strings as we call it
    // from several places below.
    processInput := func(input string) {
        lines := strings.Split(input, "\r\n")
        buffer.WriteString(lines[0])
        toSend = append(toSend, buffer.String())
        buffer.Reset()

        for idx, line := range(lines[1:]) {
            if idx == len(lines) - 1 {
                break
            }
            toSend = append(toSend, line)
        }

        if len(lines) > 1 {
            buffer.WriteString(lines[len(lines) - 1])
        }
    }

    for {
        if len(toSend) > 0 {
            // We have lines to send, so try to both send and receive.
            select {
            case latest, ok := <- textIn:
                // New input data received.
                if !ok {
                    break
                }
                processInput(latest)
            case linesOut <- toSend[0]:
                toSend = toSend[1:]
            }
        } else {
            // We have no lines queued to send, so just wait for more input.
            latest, ok := <- textIn
            if !ok {
                break
            }
            processInput(latest)
        }
    }

    // The input channel got closed, so wait for all output to be consumed and then terminate.
    for _, line := range(toSend) {
        linesOut <- line
    }
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
func isRequest(startLine string) bool {
	// SIP request lines contain precisely two spaces.
	if strings.Count(startLine, " ") != 2 {
		return false
	}

	// Check that the version string starts with SIP.
    parts := strings.Split(startLine, " ")
    if len(parts) < 3 {
        return false
    } else if len(parts[2]) < 3 {
        return false
    } else {
        return strings.ToUpper(parts[2][:3]) == "SIP"
    }
}

// Heuristic to determine if the given transmission looks like a SIP response.
// It is guaranteed that any RFC3261-compliant response will pass this test,
// but invalid messages may not necessarily be rejected.
func isResponse(startLine string) bool {
	// SIP status lines contain at least two spaces.
	if strings.Count(startLine, " ") < 2 {
		return false
	}

	// Check that the version string starts with SIP.
    parts := strings.Split(startLine, " ")
    if len(parts) < 3 {
        return false
    } else if len(parts[0]) < 3 {
        return false
    } else {
        return strings.ToUpper(parts[0][:3]) == "SIP"
    }
}

// Parse the first line of a SIP request, e.g:
//   INVITE bob@example.com SIP/2.0
//   REGISTER jane@telco.com SIP/1.0
func parseRequestLine(requestLine string) (
	method base.Method, recipient base.Uri, sipVersion string, err error) {
	parts := strings.Split(requestLine, " ")
	if len(parts) != 3 {
		err = fmt.Errorf("request line should have 2 spaces: '%s'", requestLine)
		return
	}

	method = base.Method(strings.ToUpper(parts[0]))
	recipient, err = ParseUri(parts[1])
	sipVersion = parts[2]

    switch recipient.(type) {
    case *base.WildcardUri:
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
func ParseUri(uriStr string) (uri base.Uri, err error) {
	if strings.TrimSpace(uriStr) == "*" {
		// Wildcard '*' URI used in the Contact headers of REGISTERs when unregistering.
		return &base.WildcardUri{}, nil
	}

	colonIdx := strings.Index(uriStr, ":")
	if colonIdx == -1 {
		err = fmt.Errorf("no ':' in URI %s", uriStr)
		return
	}

	switch strings.ToLower(uriStr[:colonIdx]) {
	case "sip":
		var sipUri base.SipUri
		sipUri, err = ParseSipUri(uriStr)
		uri = &sipUri
	case "sips":
		// SIPS URIs have the same form as SIP uris, so we use the same parser.
		var sipUri base.SipUri
		sipUri, err = ParseSipUri(uriStr)
		uri = &sipUri
	default:
		err = fmt.Errorf("Unsupported URI schema %s", uriStr[:colonIdx])
	}

	return
}

// ParseSipUri converts a string representation of a SIP or SIPS URI into a SipUri object.
func ParseSipUri(uriStr string) (uri base.SipUri, err error) {
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
			if !inQuotes && strings.Contains(c_ABNF_WS, string(source[consumed])) {
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

// Parse a header string, producing one or more SipHeader objects.
// (SIP messages containing multiple headers of the same type can express them as a
// single header containing a comma-separated argument list).
func (p *parser) parseHeader(headerText string) (headers []base.SipHeader, err error) {
    log.Debug("Parser %p parsing header \"%s\"", p, headerText)
    headers = make([]base.SipHeader, 0)

	colonIdx := strings.Index(headerText, ":")
	if colonIdx == -1 {
		err = fmt.Errorf("Field name with no value in header: %s", headerText)
		return
	}

	fieldName := strings.ToLower(strings.TrimSpace(headerText[:colonIdx]))
	fieldText := strings.TrimSpace(headerText[colonIdx+1:])
	if headerParser, ok := p.headerParsers[fieldName]; ok {
		// We have a registered parser for this header type - use it.
        headers, err = headerParser(fieldName, fieldText)
	} else {
		// We have no registered parser for this header type,
		// so we encapsulate the header data in a GenericHeader struct.
        log.Debug("Parser %p has no parser for header type %s", p, fieldName)
		header := base.GenericHeader{fieldName, fieldText}
		headers = []base.SipHeader{&header}
	}

    return
}

// Parse a To, From or Contact header line, producing one or more logical SipHeaders.
func parseAddressHeader(headerName string, headerText string) (
	headers []base.SipHeader, err error) {
	switch headerName {
	case "to", "from", "contact", "t", "f", "m":
		var displayNames []*string
		var uris []base.Uri
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
			var header base.SipHeader
			if headerName == "to" || headerName == "t" {
				if idx > 0 {
					// Only a single To header is permitted in a SIP message.
					return nil,
						fmt.Errorf("Multiple to: headers in message:\n%s: %s",
							headerName, headerText)
				}
				switch uris[idx].(type) {
				case *base.WildcardUri:
					// The Wildcard '*' URI is only permitted in Contact headers.
					err = fmt.Errorf("wildcard uri not permitted in to: "+
						"header: %s", headerText)
					return
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
				case *base.WildcardUri:
					// The Wildcard '*' URI is only permitted in Contact headers.
					err = fmt.Errorf("wildcard uri not permitted in from: "+
						"header: %s", headerText)
					return
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
							return
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
	}

	return
}

// Parse a string representation of a CSeq header, returning a slice of at most one CSeq.
func parseCSeq(headerName string, headerText string) (
	headers []base.SipHeader, err error) {
	var cseq base.CSeq

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
			host, port, err = parseHostPort(viaBody)
			hop.Host = host
			hop.Port = port
			if err != nil {
				return
			}
		} else {
			host, port, err = parseHostPort(viaBody[:paramsIdx])
			if err != nil {
				return
			}
			hop.Host = host
			hop.Port = port

			hop.Params, _, err = parseParams(viaBody[paramsIdx:],
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

// parseAddressValues parses a comma-separated list of addresses, returning
// any display names and header params, as well as the SIP URIs themselves.
// parseAddressValues is aware of < > bracketing and quoting, and will not
// break on commas within these structures.
func parseAddressValues(addresses string) (
	displayNames []*string, uris []base.Uri,
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
			var uri base.Uri
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
	displayName *string, uri base.Uri,
	headerParams map[string]*string,
	err error) {

	if len(addressText) == 0 {
		err = fmt.Errorf("address-type header has empty body")
		return
	}

	addressTextCopy := addressText
	addressText = strings.TrimSpace(addressText)

	firstAngleBracket := findUnescaped(addressText, '<', quotes_delim)
	firstSpace := findAnyUnescaped(addressText, c_ABNF_WS, quotes_delim, angles_delim)
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
