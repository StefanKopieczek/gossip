package gossip

import "bytes"
import "fmt"
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
    UriParams map[string]*string
    Headers map[string]*string
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
    // Form ;key1=value1;key2;key3=value3
    for key, value := range(uri.UriParams) {
        buffer.WriteString(";")
        buffer.WriteString(key)
        if value != nil {
            buffer.WriteString("=")
            buffer.WriteString(*value)
        }
    }

    // Optional header section.
    // Has form ?key1=value1&key2&key3=value3
    firstHeader := true
    for key, value := range(uri.Headers) {
        if firstHeader {
            buffer.WriteString("?")
        } else {
            buffer.WriteString("&")
        }

        buffer.WriteString(key)
        if value != nil {
            buffer.WriteString("=")
            buffer.WriteString(*value)
        }
    }

    return buffer.String()
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

    for key, value := range(to.params) {
        if value != nil {
            buffer.WriteString(fmt.Sprintf(";%s=%s", key, *value))
        } else {
            buffer.WriteString(fmt.Sprintf(";%s", key))
        }
    }

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

    for key, value := range(from.params) {
        if value != nil {
            buffer.WriteString(fmt.Sprintf(";%s=%s", key, *value))
        } else {
            buffer.WriteString(fmt.Sprintf(";%s", key))
        }
    }

    return buffer.String()
}

type ContactHeader struct  {
    displayName *string
    uri SipUri
    params map[string]*string
}
func (contact *ContactHeader) String() (string) {
    var buffer bytes.Buffer
    buffer.WriteString("Contact: ")

    if (contact.displayName != nil) {
        buffer.WriteString(fmt.Sprintf("\"%s\" ", *contact.displayName))
    }

    buffer.WriteString(fmt.Sprintf("<%s>", contact.uri.String()))

    for key, value := range(contact.params) {
        if value != nil {
            buffer.WriteString(fmt.Sprintf(";%s=%s", key, *value))
        } else {
            buffer.WriteString(fmt.Sprintf(";%s", key))
        }
    }

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
    port *uint8
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

    for key, value := range(via.params) {
        if value != nil {
            buffer.WriteString(fmt.Sprintf(";%s=%s", key, *value))
        } else {
            buffer.WriteString(fmt.Sprintf(";%s", key))
        }
    }

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
