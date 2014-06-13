package gossip

import "bytes"
import "fmt"
import "strings"
import "strconv"
import "unicode"
import "unicode/utf8"

const WS = " \t" // TODO add all whitespace.

type MessageParser interface {
    ParseMessage(rawData []byte) (SipMessage, error)
    SetHeaderParser(headerName string, headerParser HeaderParser) ()
}

type HeaderParser func(headerName string, headerData string) (
    headers []SipHeader, err error)

var knownMethods map[string]bool = map[string]bool {
    "INVITE"    :  true,
    "CANCEL"    :  true,
    "BYE"       :  true,
    "REGISTER"  :  true,
    "SUBSCRIBE" :  true,
    "NOTIFY"    :  true,
    "OPTIONS"   :  true,
}

type parserImpl struct {
    headerParsers map[string]HeaderParser
}

func NewMessageParser() (MessageParser) {
    var parser parserImpl
    parser.headerParsers = make(map[string]HeaderParser)
    headerParsers := map[string]HeaderParser {
        "to"             : parseAddressHeader,
        "t"              : parseAddressHeader,
        "from"           : parseAddressHeader,
        "f"              : parseAddressHeader,
        "contact"        : parseAddressHeader,
        "m"              : parseAddressHeader,
        "call-id"        : parseCallId,
        "cseq"           : parseCSeq,
        "via"            : parseViaHeader,
        "v"              : parseViaHeader,
        "max-forwards"   : parseMaxForwards,
        "content-length" : parseContentLength,
        "l"              : parseContentLength,
        "content-type"   : parseContentType,
        "c"              : parseContentType,
    }
    for headerName, headerParser := range(headerParsers) {
        parser.SetHeaderParser(headerName, headerParser)
    }

    return &parser
}

func (parser *parserImpl) SetHeaderParser(headerName string,
                                          headerParser HeaderParser) {
    headerName = strings.ToLower(headerName)
    parser.headerParsers[headerName] = headerParser
}

func (parser *parserImpl) ParseMessage(rawData []byte) (SipMessage, error) {
    contents := strings.Split(string(rawData), "\r\n")
    if isRequest(contents) {
        return parser.parseRequest(contents)
    } else if isResponse(contents) {
        return parser.parseResponse(contents)
    }

    return nil, fmt.Errorf("transmission beginnng '%s' is not a SIP message",
        contents[0])
}

// Heuristic to determine if the given transmission looks like a SIP request.
// It is guaranteed that any RFC3261-compliant request will pass this test,
// but invalid messages may not necessarily be rejected.
func isRequest(contents []string) (bool) {
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
func isResponse(contents []string) (bool) {
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

    request.Method, request.Uri, request.SipVersion, err =
        parseRequestLine(contents[0])
    if (err != nil) {
        return nil, err
    }

    var consumed int
    request.Headers, consumed, err = parser.parseHeaders(contents[1:])
    if (err != nil) {
        return nil, err
    }

    if len(contents) == consumed + 2 {
        return &request, err
    } else if len(contents) == consumed + 1 {
        err = fmt.Errorf("Request beginning '%s' has no CRLF at end of headers",
                         contents[0])
        return nil, err
    } else if len(contents) <= consumed {
        err = fmt.Errorf(
            "Internal error: consumed %d lines processing request " +
            "beginning '%s' but message length was %d lines!",
            consumed, len(contents), contents[0])
        return nil, err
    }


    bodyText := strings.Join(contents[2 + consumed:], "\r\n")
    request.Body = &bodyText

    return &request, err
}

func (parser *parserImpl) parseResponse(contents []string) (*Response, error) {
    var response Response
    var err error

    response.SipVersion, response.StatusCode, response.Reason, err =
        parseStatusLine(contents[0])
    if (err != nil) {
        return nil, err
    }

    var consumed int
    response.Headers, consumed, err = parser.parseHeaders(contents[1:])
    if (err != nil) {
        return nil, err
    }

    fmt.Printf("--> %d", consumed)
    if len(contents) == consumed + 2 {
        fmt.Printf("--> %d", consumed)
        return &response, err
    } else if len(contents) == consumed + 1 {
        err = fmt.Errorf("Response beginning '%s' has no CRLF at end of headers",
                         contents[0])
        return nil, err
    } else if len(contents) <= consumed {
        err = fmt.Errorf(
            "Internal error: consumed %d lines processing response " +
            "beginning '%s' but message length was %d lines!",
            consumed, len(contents), contents[0])
        return nil, err
    }

    bodyText := strings.Join(contents[2 + consumed:], "\r\n")
    response.Body = &bodyText

    return &response, err
}

func parseRequestLine(requestLine string) (
    method Method, uri SipUri, sipVersion string, err error) {
    parts := strings.Split(requestLine, " ")
    if len(parts) != 3 {
        err = fmt.Errorf("request line should have 3 spaces: '%s'", requestLine)
        return
    }

    var m string = strings.ToUpper(parts[0])
    method = Method(m)
    uri, err = parseSipUri(parts[1])
    sipVersion = parts[2]

    return
}

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

func parseSipUri(uriStr string) (uri SipUri, err error) {
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
        err = fmt.Errorf("no ':' after protocol name in SIP uri '%s'",
                         uriStrCopy)
        return
    }
    uriStr = uriStr[1:]

    // SIP URIs may contain a user-info part, ending in a '@'.
    // This is the only place '@' may occur, so we can use it to check for the
    // existence of a user-info part.
    endOfUserInfoPart := strings.Index(uriStr, "@")
    if (endOfUserInfoPart != -1) {
        // A user-info part is present. These take the form:
        //     user [ ":" password ] "@"
        endOfUsernamePart := strings.Index(uriStr, ":")
        if (endOfUsernamePart > endOfUserInfoPart) {
            endOfUsernamePart = -1
        }

        if (endOfUsernamePart == -1) {
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

    endOfUriPart := strings.Index(uriStr, ";")
    if (endOfUriPart == -1) {
        endOfUriPart = len(uriStr) - 1
    }

    uri.Host, uri.Port, err = parseHostPort(uriStr[:endOfUriPart+1])
    uriStr = uriStr[endOfUriPart+1:]
    if err != nil || len(uriStr) == 0 {
        return
    }

    // Now parse any URI parameters.
    // These are key-value pairs separated by ';'.
    // They end at the end of the URI, or at the start of any URI headers
    // which may be present (denoted by an initial '?').
    uriParams, n, err := parseKeyValuePairs(uriStr, ';', ';', '?')
    if (err != nil) {
        return
    }
    uri.UriParams = uriParams
    uriStr = uriStr[n:]

    // Finally parse any URI headers.
    // These are key-value pairs, starting with a '?' and separated by '&'.
    headers, n, err := parseKeyValuePairs(uriStr, '?', '&', 0)
    if (err != nil) {
        return
    }
    uri.Headers = headers
    uriStr = uriStr[n:]
    if (len(uriStr) > 0) {
        err = fmt.Errorf("internal error: parse of SIP uri ended early! '%s'",
            uriStrCopy)
        return // Defensive return
    }

    return
}

func parseHostPort(rawText string) (host string, port *uint8, err error) {
    colonIdx := strings.Index(rawText, ":")
    if (colonIdx == -1) {
        host = rawText
        return
    }

    // Surely there must be a better way..!
    var portRaw64 uint64
    var portRaw8 uint8
    host = rawText[:colonIdx]
    portRaw64, err = strconv.ParseUint(rawText[colonIdx+1:], 10, 8)
    portRaw8 = uint8(portRaw64)
    port = &portRaw8

    return
}

func parseKeyValuePairs(source string,
                        start uint8, sep uint8, end uint8) (
    pairs map[string]*string, consumed int, err error) {

    pairs = make(map[string]*string)

    if len(source) == 0 {
        // Key-value section is completely empty; return defaults.
        return
    }

    if start != 0 {
        if source[0] != start {
            err = fmt.Errorf("expected %c at start of key-value section; " +
                             "got %c. section was %s", start, source[0], source)
            return
        }
        consumed += 1
        source = source[1:]
    }

    for {
        equalsIdx := strings.Index(source, "=")
        sepIdx := strings.Index(source, string(sep))
        endIdx := strings.Index(source, string(end))

        if equalsIdx != -1 &&
           (equalsIdx < sepIdx || sepIdx == -1) &&
           (equalsIdx < endIdx || endIdx == -1) {
            // We have a key value pair. Parse it.
            key := source[:equalsIdx]
            source = source[equalsIdx + 1:]
            consumed += equalsIdx + 1

            if sepIdx != -1 && (sepIdx < endIdx || endIdx == -1) {
                // There are more tokens to come, so consume the value up to
                // the separator and then move the loop on to the next token.
                value := source[:sepIdx]
                consumed += sepIdx + 1
                source = source[sepIdx + 1:]
                pairs[key] = &value
                continue
            } else {
                // That was the last token, so consume the value up to the end
                // of the section and then exit.
                if endIdx == -1 {
                    endIdx = len(source)
                    consumed -= 1 // No end token to consume.
                }
                value := source[:endIdx]
                consumed += endIdx
                pairs[key] = &value
                return
            }
        } else {
            // We have a key with no value. This is valid, so parse it.
            if sepIdx != -1 && (sepIdx < endIdx || endIdx == -1) {
                // There are more tokens to come, so consume the key up to
                // the separator and then move the loop on to the next token.
                key := source[:sepIdx]
                consumed += sepIdx + 1
                source = source[sepIdx + 1:]
                pairs[key] = nil
                continue
            } else {
                // That was the last token, so consume the value up to the end
                // of the section and then exit.
                if endIdx == -1 {
                    endIdx = len(source)
                    consumed -= 1 // No end token to consume.
                }
                key := source[:endIdx]
                consumed += endIdx
                pairs[key] = nil
                return
            }
        }
    }
}

func (parser *parserImpl) parseHeaders(contents[] string) (
    headers []SipHeader, consumed int, err error) {
    headers = make([]SipHeader, 0)
    for {
        headerText, lines := getNextHeaderBlock(contents[consumed:])
        if lines == 0 {
            // End of header section
            return
        }

        var someHeaders []SipHeader
        someHeaders, err = parser.parseHeaderSection(headerText)
        if (err != nil) {
            return
        }
        headers = append(headers, someHeaders...)
        consumed += lines
    }

    return
}

func (parser *parserImpl) parseHeaderSection(headerText string) (
        headers []SipHeader, err error) {
    colonIdx := strings.Index(headerText, ":")
    if colonIdx == -1 {
        err = fmt.Errorf("Field name with no value in header: %s", headerText)
        return
    }

    fieldName := strings.ToLower(strings.TrimSpace(headerText[:colonIdx]))
    fieldText := strings.TrimSpace(headerText[colonIdx+1:])
    if headerParser, ok := parser.headerParsers[fieldName]; ok {
        return headerParser(fieldName, fieldText)
    } else {
        header := GenericHeader{fieldName, fieldText}
        headers = []SipHeader{&header}
        return
    }


    return
}

func parseAddressHeader(headerName string, headerText string) (
        headers []SipHeader, err error) {
    switch (headerName) {
    case "to", "from", "contact", "t", "f", "m":
        var displayNames []*string
        var uris []SipUri
        var paramSets []map[string]*string
        displayNames, uris, paramSets, err = parseAddressValues(headerText)

        if (err != nil) {
            return
        }
        if (len(displayNames) != len(uris) || len(uris) != len(paramSets)) {
            err = fmt.Errorf("internal parser error: parsed param mismatch. " +
                             "%d display names, %d uris and %d param sets " +
                             "in %s.",
                             len(displayNames), len(uris), len(paramSets),
                             headerText)
            return
        }

        for idx := 0; idx < len(displayNames); idx++ {
            var header SipHeader
            if (headerName == "to" || headerName == "t") {
                toHeader := ToHeader{displayNames[idx],
                                     &uris[idx],
                                     paramSets[idx]}
                header = &toHeader
            } else if (headerName == "from" || headerName == "f") {
                fromHeader := FromHeader{displayNames[idx],
                                         &uris[idx],
                                         paramSets[idx]}
                header = &fromHeader
            } else if (headerName == "contact" || headerName == "m") {
                contactHeader := ContactHeader{displayNames[idx],
                                               uris[idx],
                                               paramSets[idx]}
                header = &contactHeader
            }

            headers = append(headers, header)
        }
    }

    return
}

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

    cseq.SeqNo = uint32(seqno)
    cseq.MethodName = Method(strings.TrimSpace(strings.ToUpper(parts[1])))

    headers = []SipHeader { &cseq }

    return
}

func parseCallId(headerName string, headerText string) (
        headers []SipHeader, err error) {
    headerText = strings.TrimSpace(headerText)
    var callId CallId = CallId(headerText)
    headers = []SipHeader { &callId }

    return
}

func parseContentType(headerName string, headerText string) (
        headers []SipHeader, err error) {
    return // TODO SMK
}

func parseViaHeader(headerName string, headerText string) (
        headers []SipHeader, err error) {
    sections := strings.Split(headerText, ",")
    for _, section := range(sections) {
        sectionCopy := section
        var via ViaHeader
        sentByIdx := strings.IndexAny(section, WS) + 1
        if (sentByIdx == -1) {
            err = fmt.Errorf("expected whitespace after sent-protocol part " +
                             "in via header '%s'", sectionCopy)
            return
        }

        sentProtocolParts := strings.Split(section[:sentByIdx-1], "/")
        if len(sentProtocolParts) != 3 {
            err = fmt.Errorf("unexpected number of protocol parts in via " +
            "header; expected 3, got %d: '%s'", len(sentProtocolParts),
            sectionCopy)
            return
        }

        via.protocolName = strings.TrimSpace(sentProtocolParts[0])
        via.protocolVersion = strings.TrimSpace(sentProtocolParts[1])
        via.transport = strings.TrimSpace(sentProtocolParts[2])

        paramsIdx := strings.Index(section, ";")
        var host string
        var port *uint8
        if paramsIdx == -1 {
            host, port, err = parseHostPort(section[sentByIdx:])
            via.host = host
            via.port = port
        } else {
            host, port, err = parseHostPort(section[sentByIdx:paramsIdx])
            if err != nil {
                return
            }
            via.host = host
            via.port = port

            via.params, _, err = parseKeyValuePairs(section[paramsIdx:],
                                                    ';', ';', 0)
        }
        headers = append(headers, &via)
    }

    return
}

func parseMaxForwards(headerName string, headerText string) (
        headers []SipHeader, err error) {
    var maxForwards MaxForwards
    var value uint64
    value, err = strconv.ParseUint(strings.TrimSpace(headerText), 10, 32)
    maxForwards = MaxForwards(value)

    headers = []SipHeader { &maxForwards }
    return
}

func parseContentLength(headerName string, headerText string) (
        headers []SipHeader, err error) {
    var contentLength ContentLength
    var value uint64
    value, err = strconv.ParseUint(strings.TrimSpace(headerText), 10, 32)
    contentLength = ContentLength(value)

    headers = []SipHeader { &contentLength }
    return
}

// parseAddressValues parses a comma-separated list of addresses, returning
// any display names and header params, as well as the SIP URIs themselves.
// parseAddressValues is aware of < > bracketing and quoting, and will not
// break on commas within these structures.
func parseAddressValues(addresses string) (
        displayNames []*string, uris []SipUri,
        headerParams []map[string]*string,
        err error) {

    prevIdx := 0
    inBrackets := false
    inQuotes := false

    // Append a comma to simplify the parsing code; we split address sections
    // on commas, so use a comma to signify the end of an address section.
    addresses = addresses + ","

    for idx, char := range(addresses) {
        if char == '<' {
            inBrackets = true
        } else if char == '>' {
            inBrackets = false
        } else if char == '"' {
            inQuotes = !inQuotes
        } else if !inQuotes && !inBrackets && char == ',' {
            var displayName *string
            var uri SipUri
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
// addresses in this form should be handled by parseAddressValues.
func parseAddressValue(addressText string) (
        displayName *string, uri SipUri,
        headerParams map[string]*string,
        err error) {

    addressTextCopy := addressText
    addressText = strings.TrimSpace(addressText)

    firstAngleBracket := strings.Index(addressText, "<")
    firstSpace := strings.IndexAny(addressText, WS)
    if (firstAngleBracket != -1 && firstSpace != -1 &&
        firstSpace < firstAngleBracket) {
        // There is a display name present. Let's parse it.
        if addressText[0] == '"' {
            // The display name is within quotations.
            addressText = addressText[1:]
            nextQuote := strings.Index(addressText, "\"")

            if (nextQuote == -1) {
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
    if addressText[0] != '<' {
        if displayName != nil {
            // The address must be in <angle brackets> if a display name is
            // present, so this is an invalid address line.
            err = fmt.Errorf("Invalid character '%c' following display " +
                             "name in address line; expected '<': %s",
                             addressText[0], addressTextCopy)
            return
        }

        endOfUri = strings.Index(addressText, ";")
        if endOfUri == -1 {
            endOfUri = len(addressText)
        }

    } else {
        addressText = addressText[1:]
        endOfUri = strings.Index(addressText, ">")
        if endOfUri == -1 {
            err = fmt.Errorf("'<' without closing '>' in address %s",
                             addressTextCopy)
            return
        }

    }

    // Now parse the SIP URI.
    uri, err = parseSipUri(addressText[:endOfUri])
    if (err != nil) {
        return
    }
    addressText = addressText[endOfUri+1:]

    // Finally, parse any header parameters and then return.
    headerParams, _, err = parseKeyValuePairs(addressText, ';', ';', ',')
    return
}


func getNextHeaderBlock(contents[] string) (headerText string, consumed int) {
    var buffer bytes.Buffer
    for consumed = 0; consumed < len(contents); consumed++ {
        firstChar, _ := utf8.DecodeRuneInString(contents[consumed])
        if consumed > 0 && !unicode.IsSpace(firstChar) {
            break
        } else if len(contents[consumed]) == 0 {
            break
        }

        buffer.WriteString(" " + strings.TrimSpace(contents[consumed]))
    }

    headerText = buffer.String()
    return
}
