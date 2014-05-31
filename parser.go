package gossip

import "bytes"
import "fmt"
import "strings"
import "strconv"
import "unicode"
import "unicode/utf8"

type MessageParser interface {
    ParseMessage(rawData []byte) (SipMessage, error)
}

var knownMethods map[string]bool = map[string]bool {
    "INVITE"    :  true,
    "CANCEL"    :  true,
    "BYE"       :  true,
    "REGISTER"  :  true,
    "SUBSCRIBE" :  true,
    "NOTIFY"    :  true,
    "OPTIONS"   :  true,
}

type parserFunc func (rawData []byte) (SipMessage, error)
func (f *parserFunc) ParseMessage(rawData []byte) (SipMessage, error) {
    return (*f)(rawData)
}

func NewMessageParser() (MessageParser) {
    var f parserFunc = parseMessage
    return &f
}

func parseMessage(rawData []byte) (SipMessage, error) {
    contents := strings.Split(string(rawData), "\r\n")
    if isRequest(contents) {
        return parseRequest(contents)
    } else if isResponse(contents) {
        return parseResponse(contents)
    }

    return nil, fmt.Errorf("transmission beginnng '%s' is not a SIP message",
        contents[0])
}

// Heuristic to determine if the given transmission looks like a SIP request.
// It is guaranteed that any RFC3261-compliant request will pass this test,
// but invalid messages may not necessarily be rejected.
func isRequest(contents []string) (bool) {
    requestLine := contents[0]

    // SIP request lines contain precisely three spaces.
    if strings.Count(requestLine, " ") != 3 {
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

    // SIP status lines contain at least three spaces.
    if strings.Count(statusLine, " ") < 3 {
        return false
    }

    // Check that the version string starts with SIP.
    versionString := statusLine[:strings.Index(statusLine, " ")]
    return versionString[:3] == "SIP"
}

func parseRequest(contents []string) (*Request, error) {
    var request Request
    var err error

    request.Method, request.Uri, request.SipVersion, err =
        parseRequestLine(contents[0])
    if (err != nil) {
        return nil, err
    }

    var consumed int
    request.headers, consumed, err = parseHeaders(contents[1:])
    if (err != nil) {
        return nil, err
    }

    bodyText := strings.Join(contents[2 + consumed:], "\r\n")
    request.Body = &bodyText

    return &request, err
}

func parseResponse(contents []string) (*Response, error) {
    var response Response
    var err error

    response.SipVersion, response.StatusCode, response.Reason, err =
        parseStatusLine(contents[0])
    if (err != nil) {
        return nil, err
    }

    var consumed int
    response.headers, consumed, err = parseHeaders(contents[1:])
    if (err != nil) {
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
    statusCodeRaw, err := strconv.ParseInt(parts[1], 10, 8)
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
    endOfUriPart := strings.Index(uriStr, "@")
    if (endOfUriPart != -1) {
        // A user-info part is present. These take the form:
        //     user [ ":" password ] "@"
        endOfUsernamePart := strings.Index(uriStr, ":")
        if (endOfUsernamePart > endOfUriPart) {
            endOfUsernamePart = -1
        }

        if (endOfUsernamePart == -1) {
            // No password component; the whole of the user-info part before
            // the '@' is a username.
            user := uriStr[:endOfUriPart]
            uri.User = &user
        } else {
            user := uriStr[:endOfUsernamePart]
            pwd := uriStr[endOfUsernamePart+1 : endOfUriPart]
            uri.User = &user
            uri.Password = &pwd
        }
        uriStr = uriStr[endOfUriPart+1:]
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

func parseKeyValuePairs(source string,
                        start uint8, sep uint8, end uint8) (
    pairs map[string]string, consumed int, err error) {

    pairs = make(map[string]string)

    consumed += 1
    source = source[1:]

    for {
        valIdx := strings.Index(source, "=") + 1
        if valIdx == -1 {
            err = fmt.Errorf("missing value following key: '%s'", source)
        }

        key := source[:valIdx-1]
        source := source[valIdx:]
        consumed += valIdx
        sepIdx := strings.Index(source, string(sep))
        endIdx := strings.Index(source, string(end))

        var value string
        if sepIdx == -1 {
            // No final separator, so the entire tail is the value part.
            value = source
            consumed += len(source)
            return
        } else if endIdx != -1 && endIdx < sepIdx {
            value = source[:endIdx]
            consumed += endIdx
            return
        } else {
            value = source[:sepIdx+1]
            source = source[sepIdx+1:]
            consumed += sepIdx + 1
        }

        pairs[key] = value
    }
}

func parseHeaders(contents[] string) (
    headers []SipHeader, consumed int, err error) {
    headers = make([]SipHeader, 0)
    for {
        headerText, lines := getNextHeaderBlock(contents[consumed:])
        if consumed == 0 {
            // End of header section
            return
        }

        var someHeaders []SipHeader
        someHeaders, err = parseHeaderSection(headerText)
        if (err != nil) {
            return
        }
        headers = append(headers, someHeaders...)
        consumed += lines
    }

    return
}

func parseHeaderSection(headerText string) (headers []SipHeader, err error) {
    colonIdx := strings.Index(headerText, ":")
    if colonIdx == -1 {
        err = fmt.Errorf("Field name with no value in header: %s", headerText)
        return
    }

    // TODO

    return
}

func parseAddressValue(addressText string) {
   return // TODO
}


func getNextHeaderBlock(contents[] string) (headerText string, consumed int) {
    var buffer bytes.Buffer
    buffer.WriteString(strings.TrimSpace(contents[0]))
    for consumed = 1; consumed < len(contents); consumed+=1 {
        firstChar, _ := utf8.DecodeRuneInString(contents[consumed])
        if !unicode.IsSpace(firstChar) {
            break
        } else if contents[consumed] == "\r\n" {
            break
        }

        buffer.WriteString(" " + strings.TrimSpace(contents[consumed]))
    }

    return
}
