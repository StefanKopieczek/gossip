package gossip

import "bytes"
import "fmt"
import "strconv"
import "strings"

type SipHeader interface {
    String() string
}

type Uri interface {
    Equals(other Uri) bool
    String() string
}

type ContactUri interface {
    Equals(other Uri) bool
    String() string
    IsWildcard() bool
}

type SipUri struct {
    IsEncrypted bool
    User *string
    Password *string
    Host string
    Port *uint16
    UriParams map[string]*string
    Headers map[string]*string
}

func (uri *SipUri) IsWildcard() bool {
    return false
}

func (uri *SipUri) Equals(otherUri Uri) (bool) {
    otherPtr, ok := otherUri.(*SipUri)
    if !ok {
        return false
    }

    other := *otherPtr
    result := uri.IsEncrypted == other.IsEncrypted &&
              strPtrEq(uri.User, other.User) &&
              strPtrEq(uri.Password, other.Password) &&
              uri.Host == other.Host &&
              uint16PtrEq(uri.Port, other.Port)

    if !result {
        return false
    }

    if !paramsEqual(uri.UriParams, other.UriParams) {
        return false
    }

    if !paramsEqual(uri.Headers, other.Headers) {
        return false
    }

    return true
}

// Generates the string representation of a SipUri struct.
func (uri *SipUri) String() (string) {
    var buffer bytes.Buffer

    // Compulsory protocol identifier.
    if (uri.IsEncrypted) {
        buffer.WriteString("sips")
        buffer.WriteString(":")
    } else {
        buffer.WriteString("sip")
        buffer.WriteString(":")
    }

    // Optional userinfo part.
    if (uri.User != nil) {
        buffer.WriteString(*uri.User)

        if (uri.Password != nil) {
            buffer.WriteString(":")
            buffer.WriteString(*uri.Password)
        }

        buffer.WriteString("@")
    }

    // Compulsory hostname.
    buffer.WriteString(uri.Host)

    // Optional port number.
    if (uri.Port != nil) {
        buffer.WriteString(":")
        buffer.WriteString(strconv.Itoa(int(*uri.Port)))
    }

    buffer.WriteString(ParamsToString(uri.UriParams, ';', ';'))
    buffer.WriteString(ParamsToString(uri.Headers, '?', '&'))

    return buffer.String()
}

type WildcardUri struct{}

func (uri *WildcardUri) IsWildcard() bool {
    return true
}

func (uri *WildcardUri) String() string {
    return "*"
}

func (uri *WildcardUri) Equals(other Uri) bool {
    switch other.(type) {
    case *WildcardUri:
        return true
    default:
        return false
    }
}

type GenericHeader struct {
    headerName string
    contents string
}
func (header *GenericHeader) String() (string) {
    return header.headerName + ": " + header.contents
}

type ToHeader struct {
    displayName *string
    uri Uri
    params map[string]*string
}
func (to *ToHeader) String() (string) {
    var buffer bytes.Buffer
    buffer.WriteString("To: ")

    if (to.displayName != nil) {
        buffer.WriteString(fmt.Sprintf("\"%s\" ", *to.displayName))
    }

    buffer.WriteString(fmt.Sprintf("<%s>", to.uri))
    buffer.WriteString(ParamsToString(to.params, ';', ';'))

    return buffer.String()
}

type FromHeader struct {
    displayName *string
    uri Uri
    params map[string]*string
}
func (from *FromHeader) String() (string) {
    var buffer bytes.Buffer
    buffer.WriteString("From: ")

    if (from.displayName != nil) {
        buffer.WriteString(fmt.Sprintf("\"%s\" ", *from.displayName))
    }

    buffer.WriteString(fmt.Sprintf("<%s>", from.uri))
    buffer.WriteString(ParamsToString(from.params, ';', ';'))

    return buffer.String()
}

type ContactHeader struct  {
    displayName *string
    uri ContactUri
    params map[string]*string
}
func (contact *ContactHeader) String() (string) {
    var buffer bytes.Buffer
    buffer.WriteString("Contact: ")

    if (contact.displayName != nil) {
        buffer.WriteString(fmt.Sprintf("\"%s\" ", *contact.displayName))
    }

    switch contact.uri.(type) {
    case *WildcardUri:
        buffer.WriteString("*")
    default:
        buffer.WriteString(fmt.Sprintf("<%s>", contact.uri.String()))
    }

    buffer.WriteString(ParamsToString(contact.params, ';', ';'))

    return buffer.String()
}

type CallId string
func (callId *CallId) String() (string) {
    return "Call-Id: " + (string)(*callId)
}

type CSeq struct {
    SeqNo uint32
    MethodName Method
}
func (cseq *CSeq) String() (string) {
    return fmt.Sprintf("CSeq: %d %s", cseq.SeqNo, cseq.MethodName)
}

type MaxForwards uint32
func (maxForwards *MaxForwards) String() (string) {
    return fmt.Sprintf("Max-Forwards: %d", ((int)(*maxForwards)))
}

type ContentLength uint32
func (contentLength *ContentLength) String() (string) {
    return fmt.Sprintf("Content-Length: %d", ((int)(*contentLength)))
}

type ViaHeader struct {
    protocolName string
    protocolVersion string
    transport string
    host string
    port *uint16
    params map[string]*string
}
func (via *ViaHeader) String() (string) {
    var buffer bytes.Buffer
    buffer.WriteString(fmt.Sprintf("Via: %s/%s/%s %s",
                                   via.protocolName, via.protocolVersion,
                                   via.transport,
                                   via.host))
    if via.port != nil {
        buffer.WriteString(fmt.Sprintf(":%d", *via.port))
    }

    buffer.WriteString(ParamsToString(via.params, ';', ';'))

    return buffer.String()
}


type RequireHeader struct {
    options []string
}
func (header *RequireHeader) String() (string) {
    return fmt.Sprintf("Require: %s",
        joinStrings(", ", header.options...))
}

type SupportedHeader struct {
    options []string
}
func (header *SupportedHeader) String() (string) {
    return fmt.Sprintf("Supported: %s",
        joinStrings(", ", header.options...))
}

type ProxyRequireHeader struct {
    options []string
}
func (header *ProxyRequireHeader) String() (string) {
    return fmt.Sprintf("Proxy-Require: %s",
        joinStrings(", ", header.options...))
}

type UnsupportedHeader struct {
    options []string
}
func (header *UnsupportedHeader) String() (string) {
    return fmt.Sprintf("Unsupported: %s",
        joinStrings(", ", header.options...))
}

func ParamsToString(params map[string]*string, start uint8, sep uint8) (
        string) {
    var buffer bytes.Buffer
    first := true
    for key, value := range(params) {
        if first {
            buffer.WriteString(fmt.Sprintf("%c", start))
            first = false
        } else {
            buffer.WriteString(fmt.Sprintf("%c", sep))
        }
        if value == nil {
            buffer.WriteString(fmt.Sprintf("%s", key))
        } else if strings.ContainsAny(*value, ABNF_WS) {
            buffer.WriteString(fmt.Sprintf("%s=\"%s\"", key, *value))
        } else {
            buffer.WriteString(fmt.Sprintf("%s=%s", key, *value))
        }
    }

    return buffer.String()
}

func paramsEqual(a map[string]*string, b map[string]*string) bool {
    if len(a) != len(b) {
        return false
    }

    for key, a_val := range(a) {
        b_val, ok := b[key]
        if !ok {
            return false
        }
        if !strPtrEq(a_val, b_val) {
            return false
        }
    }

    return true
}
