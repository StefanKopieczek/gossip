package base

import (
	"github.com/weave-lab/gossip/utils"
)

import "bytes"
import "fmt"
import "strconv"
import "strings"

// Whitespace recognised by SIP protocol.
const c_ABNF_WS = " \t"

// A single logical header from a SIP message.
type SipHeader interface {
	// Produce the string representation of the header.
	String() string

	// Produce the name of the header (e.g. "To", "Via")
	Name() string

	// Produce an exact copy of this header.
	Copy() SipHeader
}

// A URI from any schema (e.g. sip:, tel:, callto:)
type Uri interface {
	// Determine if the two URIs are equal according to the rules in RFC 3261 s. 19.1.4.
	Equals(other Uri) bool

	// Produce the string representation of the URI.
	String() string

	// Produce an exact copy of this URI.
	Copy() Uri
}

// A URI from a schema suitable for inclusion in a Contact: header.
// The only such URIs are sip/sips URIs and the special wildcard URI '*'.
type ContactUri interface {
	Uri

	// Return true if and only if the URI is the special wildcard URI '*'; that is, if it is
	// a WildcardUri struct.
	IsWildcard() bool
}

// A SIP or SIPS URI, including all params and URI header params.
type SipUri struct {
	// True if and only if the URI is a SIPS URI.
	IsEncrypted bool

	// The user part of the URI: the 'joe' in sip:joe@bloggs.com
	// This is a pointer, so that URIs without a user part can have 'nil'.
	User *string

	// The password field of the URI. This is represented in the URI as joe:hunter2@bloggs.com.
	// Note that if a URI has a password field, it *must* have a user field as well.
	// This is a pointer, so that URIs without a password field can have 'nil'.
	// Note that RFC 3261 strongly recommends against the use of password fields in SIP URIs,
	// as they are fundamentally insecure.
	Password *string

	// The host part of the URI. This can be a domain, or a string representation of an IP address.
	Host string

	// The port part of the URI. This is optional, and so is represented here as a pointer type.
	Port *uint16

	// Any parameters associated with the URI.
	// These are used to provide information about requests that may be constructed from the URI.
	// (For more details, see RFC 3261 section 19.1.1).
	// These appear as a semicolon-separated list of key=value pairs following the host[:port] part.
	// Note that not all keys have an associated value, so the values of the map may be nil.
	UriParams Params

	// Any headers to be included on requests constructed from this URI.
	// These appear as a '&'-separated list at the end of the URI, introduced by '?'.
	// Although the values of the map are pointers, they will never be nil in practice as the parser
	// guarantees to not return nil values for header elements in SIP URIs.
	// You should not set the values of headers to nil.
	Headers Params
}

// Copy the Sip URI.
func (uri *SipUri) Copy() Uri {
	var user, password *string
	var port *uint16
	if uri.User != nil {
		temp := *uri.User
		user = &temp
	}
	if uri.Password != nil {
		temp := *uri.Password
		password = &temp
	}
	if uri.Port != nil {
		temp := *uri.Port
		port = &temp
	}

	return &SipUri{
		uri.IsEncrypted,
		user,
		password,
		uri.Host,
		port,
		uri.UriParams.Copy(),
		uri.Headers.Copy(),
	}
}

// IsWildcard() always returns 'false' for SIP URIs as they are not equal to the wildcard '*' URI.
// This method is required since SIP URIs are valid in Contact: headers.
func (uri *SipUri) IsWildcard() bool {
	return false
}

// Determine if the SIP URI is equal to the specified URI according to the rules laid down in RFC 3261 s. 19.1.4.
// TODO: The Equals method is not currently RFC-compliant; fix this!
func (uri *SipUri) Equals(otherUri Uri) bool {
	otherPtr, ok := otherUri.(*SipUri)
	if !ok {
		return false
	}

	other := *otherPtr
	result := uri.IsEncrypted == other.IsEncrypted &&
		utils.StrPtrEq(uri.User, other.User) &&
		utils.StrPtrEq(uri.Password, other.Password) &&
		uri.Host == other.Host &&
		utils.Uint16PtrEq(uri.Port, other.Port)

	if !result {
		return false
	}

	if !ParamsEqual(uri.UriParams, other.UriParams) {
		return false
	}

	if !ParamsEqual(uri.Headers, other.Headers) {
		return false
	}

	return true
}

// Generates the string representation of a SipUri struct.
func (uri *SipUri) String() string {
	var buffer bytes.Buffer

	// Compulsory protocol identifier.
	if uri.IsEncrypted {
		buffer.WriteString("sips")
		buffer.WriteString(":")
	} else {
		buffer.WriteString("sip")
		buffer.WriteString(":")
	}

	// Optional userinfo part.
	if uri.User != nil {
		buffer.WriteString(*uri.User)

		if uri.Password != nil {
			buffer.WriteString(":")
			buffer.WriteString(*uri.Password)
		}

		buffer.WriteString("@")
	}

	// Compulsory hostname.
	buffer.WriteString(uri.Host)

	// Optional port number.
	if uri.Port != nil {
		buffer.WriteString(":")
		buffer.WriteString(strconv.Itoa(int(*uri.Port)))
	}

	buffer.WriteString(ParamsToString(uri.UriParams, ';', ';'))
	buffer.WriteString(ParamsToString(uri.Headers, '?', '&'))

	return buffer.String()
}

// The special wildcard URI used in Contact: headers in REGISTER requests when expiring all registrations.
type WildcardUri struct{}

// Copy the wildcard URI. Not hard!
func (uri WildcardUri) Copy() Uri { return uri }

// Always returns 'true'.
func (uri WildcardUri) IsWildcard() bool {
	return true
}

// Always returns '*' - the representation of a wildcard URI in a SIP message.
func (uri WildcardUri) String() string {
	return "*"
}

// Determines if this wildcard URI equals the specified other URI.
// This is true if and only if the other URI is also a wildcard URI.
func (uri WildcardUri) Equals(other Uri) bool {
	switch other.(type) {
	case WildcardUri:
		return true
	default:
		return false
	}
}

// Generic list of parameters on a header.
type Params map[string]*string

// Copy a list of params.
func (p Params) Copy() Params {
	dup := make(map[string]*string, len(p))
	for k, v := range p {
		if v != nil {
			s := *v
			dup[k] = &s
		} else {
			dup[k] = nil
		}
	}
	return dup
}

// Encapsulates a header that gossip does not natively support.
// This allows header data that is not understood to be parsed by gossip and relayed to the parent application.
type GenericHeader struct {
	// The name of the header.
	HeaderName string

	// The contents of the header, including any parameters.
	// This is transparent data that is not natively understood by gossip.
	Contents string
}

// Convert the header to a flat string representation.
func (header *GenericHeader) String() string {
	return header.HeaderName + ": " + header.Contents
}

// Pull out the header name.
func (h *GenericHeader) Name() string {
	return h.HeaderName
}

// Copy the header.
func (h *GenericHeader) Copy() SipHeader {
	return &GenericHeader{h.HeaderName, h.Contents}
}

type ToHeader struct {
	// The display name from the header - this is a pointer type as it is optional.
	DisplayName *string

	Address Uri

	// Any parameters present in the header.
	Params Params
}

func (to *ToHeader) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("To: ")

	if to.DisplayName != nil {
		buffer.WriteString(fmt.Sprintf("\"%s\" ", *to.DisplayName))
	}

	buffer.WriteString(fmt.Sprintf("<%s>", to.Address))
	buffer.WriteString(ParamsToString(to.Params, ';', ';'))

	return buffer.String()
}

func (h *ToHeader) Name() string { return "To" }

// Copy the header. A little tricky due to string pointers.
func (h *ToHeader) Copy() SipHeader {
	var name *string
	if h.DisplayName != nil {
		temp := *h.DisplayName
		name = &temp
	}
	return &ToHeader{name, h.Address.Copy(), h.Params.Copy()}
}

type FromHeader struct {
	// The display name from the header - this is a pointer type as it is optional.
	DisplayName *string

	Address Uri

	// Any parameters present in the header.
	Params Params
}

func (from *FromHeader) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("From: ")

	if from.DisplayName != nil {
		buffer.WriteString(fmt.Sprintf("\"%s\" ", *from.DisplayName))
	}

	buffer.WriteString(fmt.Sprintf("<%s>", from.Address))
	buffer.WriteString(ParamsToString(from.Params, ';', ';'))

	return buffer.String()
}

func (h *FromHeader) Name() string { return "From" }

// Copy the header. A little tricky due to string pointers.
func (h *FromHeader) Copy() SipHeader {
	var name *string
	if h.DisplayName != nil {
		temp := *h.DisplayName
		name = &temp
	}
	return &FromHeader{name, h.Address.Copy(), h.Params.Copy()}
}

type ContactHeader struct {
	// The display name from the header - this is a pointer type as it is optional.
	DisplayName *string

	Address ContactUri

	// Any parameters present in the header.
	Params Params
}

func (contact *ContactHeader) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("Contact: ")

	if contact.DisplayName != nil {
		buffer.WriteString(fmt.Sprintf("\"%s\" ", *contact.DisplayName))
	}

	switch contact.Address.(type) {
	case *WildcardUri:
		// Treat the Wildcard URI separately as it must not be contained in < > angle brackets.
		buffer.WriteString("*")
	default:
		buffer.WriteString(fmt.Sprintf("<%s>", contact.Address.String()))
	}

	buffer.WriteString(ParamsToString(contact.Params, ';', ';'))

	return buffer.String()
}

func (h *ContactHeader) Name() string { return "Contact" }

// Copy the header. A little tricky due to string pointers.
func (h *ContactHeader) Copy() SipHeader {
	var name *string
	if h.DisplayName != nil {
		temp := *h.DisplayName
		name = &temp
	}
	return &ContactHeader{name, h.Address.Copy().(ContactUri), h.Params.Copy()}
}

type CallId string

func (callId CallId) String() string {
	return "Call-Id: " + (string)(callId)
}

func (h *CallId) Name() string { return "Call-Id" }

func (h *CallId) Copy() SipHeader {
	temp := *h
	return &temp
}

type CSeq struct {
	SeqNo      uint32
	MethodName Method
}

func (cseq *CSeq) String() string {
	return fmt.Sprintf("CSeq: %d %s", cseq.SeqNo, cseq.MethodName)
}

func (h *CSeq) Name() string { return "CSeq" }

func (h *CSeq) Copy() SipHeader { return &CSeq{h.SeqNo, h.MethodName} }

type MaxForwards uint32

func (maxForwards MaxForwards) String() string {
	return fmt.Sprintf("Max-Forwards: %d", ((int)(maxForwards)))
}

func (h MaxForwards) Name() string { return "Max-Forwards" }

func (h MaxForwards) Copy() SipHeader { return h }

type ContentLength uint32

func (contentLength ContentLength) String() string {
	return fmt.Sprintf("Content-Length: %d", ((int)(contentLength)))
}

func (h ContentLength) Name() string { return "Content-Length" }

func (h ContentLength) Copy() SipHeader { return h }

type ViaHeader []*ViaHop

// A single component in a Via header.
// Via headers are composed of several segments of the same structure, added by successive nodes in a routing chain.
type ViaHop struct {
	// E.g. 'SIP'.
	ProtocolName string

	// E.g. '2.0'.
	ProtocolVersion string
	Transport       string
	Host            string

	// The port for this via hop. This is stored as a pointer type, since it is an optional field.
	Port *uint16

	Params Params
}

func (hop *ViaHop) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("%s/%s/%s %s",
		hop.ProtocolName, hop.ProtocolVersion,
		hop.Transport,
		hop.Host))
	if hop.Port != nil {
		buffer.WriteString(fmt.Sprintf(":%d", *hop.Port))
	}

	buffer.WriteString(ParamsToString(hop.Params, ';', ';'))

	return buffer.String()
}

// Return an exact copy of this ViaHop.
func (hop *ViaHop) Copy() *ViaHop {
	var port *uint16 = nil
	if hop.Port != nil {
		temp := *hop.Port
		port = &temp
	}
	return &ViaHop{
		hop.ProtocolName,
		hop.ProtocolVersion,
		hop.Transport,
		hop.Host,
		port,
		hop.Params.Copy(),
	}
}

func (via ViaHeader) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("Via: ")
	for idx, hop := range via {
		buffer.WriteString(hop.String())
		if idx != len(via)-1 {
			buffer.WriteString(", ")
		}
	}

	return buffer.String()
}

func (h ViaHeader) Name() string { return "Via" }

func (h ViaHeader) Copy() SipHeader {
	dup := make([]*ViaHop, 0, len(h))
	for _, hop := range h {
		dup = append(dup, hop.Copy())
	}
	return ViaHeader(dup)
}

type RequireHeader struct {
	Options []string
}

func (header *RequireHeader) String() string {
	return fmt.Sprintf("Require: %s",
		strings.Join(header.Options, ", "))
}

func (h *RequireHeader) Name() string { return "Require" }

func (h *RequireHeader) Copy() SipHeader {
	dup := make([]string, len(h.Options))
	copy(h.Options, dup)
	return &RequireHeader{dup}
}

type SupportedHeader struct {
	Options []string
}

func (header *SupportedHeader) String() string {
	return fmt.Sprintf("Supported: %s",
		strings.Join(header.Options, ", "))
}

func (h *SupportedHeader) Name() string { return "Supported" }

func (h *SupportedHeader) Copy() SipHeader {
	dup := make([]string, len(h.Options))
	copy(h.Options, dup)
	return &SupportedHeader{dup}
}

type ProxyRequireHeader struct {
	Options []string
}

func (header *ProxyRequireHeader) String() string {
	return fmt.Sprintf("Proxy-Require: %s",
		strings.Join(header.Options, ", "))
}

func (h *ProxyRequireHeader) Name() string { return "Proxy-Require" }

func (h *ProxyRequireHeader) Copy() SipHeader {
	dup := make([]string, len(h.Options))
	copy(h.Options, dup)
	return &ProxyRequireHeader{dup}
}

// 'Unsupported:' is a SIP header type - this doesn't indicate that the
// header itself is not supported by gossip!
type UnsupportedHeader struct {
	Options []string
}

func (header *UnsupportedHeader) String() string {
	return fmt.Sprintf("Unsupported: %s",
		strings.Join(header.Options, ", "))
}

func (h *UnsupportedHeader) Name() string { return "Unsupported" }

func (h *UnsupportedHeader) Copy() SipHeader {
	dup := make([]string, len(h.Options))
	copy(h.Options, dup)
	return &UnsupportedHeader{dup}
}

// Utility method for converting a map of parameters to a flat string representation.
// Takes the map of parameters, and start and end characters (e.g. '?' and '&').
// It is assumed that key/value pairs are always represented as "key=value".
// Note that this method does not escape special characters - that should be done before calling this method.
func ParamsToString(params Params, start uint8, sep uint8) string {
	var buffer bytes.Buffer
	first := true
	for key, value := range params {
		if first {
			buffer.WriteString(fmt.Sprintf("%c", start))
			first = false
		} else {
			buffer.WriteString(fmt.Sprintf("%c", sep))
		}
		if value == nil {
			buffer.WriteString(fmt.Sprintf("%s", key))
		} else if strings.ContainsAny(*value, c_ABNF_WS) {
			buffer.WriteString(fmt.Sprintf("%s=\"%s\"", key, *value))
		} else {
			buffer.WriteString(fmt.Sprintf("%s=%s", key, *value))
		}
	}

	return buffer.String()
}

// Check if two maps of parameters are equal in the sense of having the same keys with the same values.
// This does not rely on any ordering of the keys of the map in memory.
func ParamsEqual(a Params, b Params) bool {
	if len(a) != len(b) {
		return false
	}

	for key, a_val := range a {
		b_val, ok := b[key]
		if !ok {
			return false
		}
		if !utils.StrPtrEq(a_val, b_val) {
			return false
		}
	}

	return true
}
