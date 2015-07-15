package sipuri

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/weave-lab/gossip/base"
)

// The whitespace characters recognised by the Augmented Backus-Naur Form syntax
// that SIP uses (RFC 3261 S.25).
const c_ABNF_WS = " \t"

// Character escaping requirements Section 19.1.2
/*
Follows RFC 2396

reserved    = ";" | "/" | "?" | ":" | "@" | "&" | "=" | "+" | "$" | ","

19.1.2 Character Escaping Requirements

                                                       dialog
                                          reg./redir. Contact/
              default  Req.-URI  To  From  Contact   R-R/Route  external
user          --          o      o    o       o          o         o
password      --          o      o    o       o          o         o
host          --          m      m    m       m          m         m
port          (1)         o      -    -       o          o         o
user-param    ip          o      o    o       o          o         o
method        INVITE      -      -    -       -          -         o
maddr-param   --          o      -    -       o          o         o
ttl-param     1           o      -    -       o          -         o
transp.-param (2)         o      -    -       o          o         o
lr-param      --          o      -    -       -          o         o
other-param   --          o      o    o       o          o         o
headers       --          -      -    -       o          -         o

*/

// parseUri converts a string representation of a URI into a Uri object.
// If the URI is malformed, or the URI schema is not recognised, an error is returned.
// URIs have the general form of schema:address.
func ParseUri(uriStr string) (uri base.Uri, err error) {
	if strings.TrimSpace(uriStr) == "*" {
		// Wildcard '*' URI used in the Contact headers of REGISTERs when unregistering.
		return base.WildcardUri{}, nil
	}

	colonIdx := strings.Index(uriStr, ":")
	if colonIdx == -1 {
		err = fmt.Errorf("no ':' in URI %s", uriStr)
		return
	}

	var protocol = strings.ToLower(uriStr[:colonIdx])
	switch protocol {
	case "sip", "sips":
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

	colonIdx := strings.Index(uriStr, ":")
	if colonIdx == -1 {
		err = fmt.Errorf("no ':' in URI %s", uriStr)
		return
	}

	var protocol = strings.ToLower(uriStr[:colonIdx])
	uriStr = uriStr[colonIdx+1:]

	protocol = strings.ToLower(protocol)
	// URI should start 'sip' or 'sips'. Check the first 3 chars.
	switch protocol {
	case "sip":
	case "sips":
		uri.IsEncrypted = true
	default:
		err = fmt.Errorf("Unknown protocol %s", protocol)
		return
	}

	// Store the original URI in case we need to print it in an error.
	uriStrCopy := uriStr

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

	uri.Host, uri.Port, err = ParseHostPort(uriStr[:endOfUriPart])
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
		uriParams, n, err = ParseParams(uriStr, ';', ';', '?', true, true)
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
	headers, n, err = ParseParams(uriStr, '?', '&', 0, true, false)
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
func ParseHostPort(rawText string) (host string, port *uint16, err error) {
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

// addressText, ';', ';', ',', true, true
func ParseParams(source string, start rune, sep uint8, end uint8, quoteValues bool, permitSingletons bool) (
	params map[string]*string, consumed int, err error) {

	params = make(map[string]*string)

	if len(source) == 0 {
		// Key-value section is completely empty; return defaults.
		return
	}

	// Ensure the starting character is correct.
	for i, v := range source {
		if start == 0 {
			break
		}

		if v == start {
			consumed = i + len([]byte(string(v)))
			break
		}

		// skip LWS
		if unicode.IsSpace(v) {
			continue
		}

		err = fmt.Errorf("expected %c at start of key-value section; got %c. section was %s",
			start, source[0], source)
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
