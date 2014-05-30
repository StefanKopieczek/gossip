package gossip

import "bytes"
import "fmt"
import "net"
import "strconv"

type SipHeader interface {
    String() (string)
}

type Uri interface {
    String() (string)
}

type SipUri struct {
    IsEncrypted bool
    User *string
    Password *string
    Host string
    Port *uint8
    UriParams map[string]string
    Headers map[string]string
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

    // Zero or more URI parameters.
    // Form ;key1=value1;key2=value2;key3=value3
    for key, value := range(uri.UriParams) {
        buffer.WriteString(";")
        buffer.WriteString(key)
        buffer.WriteString("=")
        buffer.WriteString(value)
    }

    // Optional header section.
    // Has form ?key1=value1&key2=value2&key3=value3
    firstHeader := true
    for key, value := range(uri.Headers) {
        if firstHeader {
            buffer.WriteString("?")
        } else {
            buffer.WriteString("&")
        }

        buffer.WriteString(key)
        buffer.WriteString("=")
        buffer.WriteString(value)
    }

    return buffer.String()
}

type ToHeader struct {
    displayName string
    uri Uri
    tag *string
}
func (to *ToHeader) String() (string) {
    result := fmt.Sprintf("To: \"%s\" <$s>", to.displayName, to.uri.String())

    if (to.tag != nil) {
        result += ";tag=" + *to.tag
    }

    return result
}

type FromHeader struct {
    displayName string
    uri Uri
    tag string
}
func (from *FromHeader) String() (string) {
    return fmt.Sprintf("From: \"%s\" <%s>;tag=%s",
        from.displayName, from.uri.String(), from.tag)
}

type CallId string
func (callId *CallId) String() (string) {
    return (string)(*callId)
}

type CSeq uint32
func (cseq *CSeq) String() (string) {
    return fmt.Sprintf("CSeq: %d", ((int)(*cseq)))
}

type MaxForwards uint32
func (maxForwards *MaxForwards) Strip() (string) {
    return fmt.Sprintf("Max-Forwards: %d", ((int)(*maxForwards)))
}

type ViaHeader struct {
    transport string
    address net.Addr
    port uint8
    branch *string
    received *net.IP
}
func (via *ViaHeader) String() (string) {
    result := fmt.Sprintf("Via: %s %s:%d", via.transport,
                          via.address.String(), via.port)
    if (via.received != nil) {
        result += ";received=" + via.received.String()
    }
    if (via.branch != nil) {
        result += ";branch=" + *via.branch
    }

    return result
}

type ContactHeader struct  {
    uri SipUri
}
func (contactHeader *ContactHeader) String() (string) {
    return fmt.Sprintf("Contact: <%s>", contactHeader.uri.String())
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
