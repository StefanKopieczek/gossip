package gossip

import "fmt"
import "net"

type SipHeader interface {
    String() (string)
}

type Uri interface {
    String() (string)
}

type SipUri interface {
    String() (string)
    HostPart() (string)
    DomainPart() (string)
}

type PlainSipUri struct {
    host string
    domain string
}
func (uri *PlainSipUri) String() (string) {
    return fmt.Sprintf("sip:%s@%s", uri.host, uri.domain)
}
func (uri *PlainSipUri) HostPart() (string) {
    return uri.host
}
func (uri *PlainSipUri) DomainPart() (string) {
    return uri.domain
}

type SipsUri struct {
    host string
    domain string
}
func (uri *SipsUri) String() (string) {
    return fmt.Sprintf("sips:%s@%s", uri.host, uri.domain)
}
func (uri *SipsUri) HostPart() (string) {
    return uri.host
}
func (uri *SipsUri) DomainPart() (string) {
    return uri.domain
}

type TelUri struct {
    // TODO
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
