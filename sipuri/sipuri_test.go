package sipuri

import (
	"github.com/weave-lab/gossip/base"
	"github.com/weave-lab/gossip/log"
)

import (
	"fmt"
	"strconv"
	"testing"
)

// Level of logs output during testing.
var c_LOG_LEVEL = log.INFO

var testsRun int
var testsPassed int

type input interface {
	String() string
	evaluate() result
}
type result interface {
	// Slight unpleasantness: equals is asymmetrical and should be called on an
	// expected value with the true result as the target.
	// This is necessary in order for the reason strings to come out right.
	equals(other result) (equal bool, reason string)
}
type test struct {
	args     input
	expected result
}

func doTests(tests []test, t *testing.T) {
	for _, test := range tests {
		testsRun++
		output := test.args.evaluate()
		pass, reason := test.expected.equals(output)
		if !pass {
			t.Errorf("Failure on input \"%s\" : %s", test.args.String(), reason)
		} else {
			testsPassed++
		}
	}
}

// Pass and fail placeholders
var fail error = fmt.Errorf("A bad thing happened.")
var pass error = nil

// Need to define immutable variables in order to pointer to them.
var alice = "alice"
var aliceAddr = "sip:alice@wonderland.com"
var aliceAddrQuot = "<sip:alice@wonderland.com>"
var aliceAddrQuotSp = "<sip: alice@wonderland.com>"
var aliceTitle = "Alice"
var aliceLiddell = "Alice Liddell"
var bar string = "bar"
var barQuote string = "\"bar\""
var barQuote2 string = "\"bar"
var barQuote3 string = "bar\""
var barBaz string = "bar;baz"
var baz string = "baz"
var bob string = "bob"
var boop string = "boop"
var b string = "b"
var empty string = ""
var hatter = "hatter"
var hunter2 string = "Hunter2"
var madHatter string = "Madison Hatter"
var port5060 uint16 = uint16(5060)
var kat string = "kat"
var ui16_5 uint16 = uint16(5)
var ui16_5060 = uint16(5060)
var ui16_9 uint16 = uint16(9)

func TestAAAASetup(t *testing.T) {
	log.SetDefaultLogLevel(c_LOG_LEVEL)
}

func TestParams(t *testing.T) {
	doTests([]test{
		// TEST: parseParams
		test{&paramInput{";foo=bar", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &bar}, 8}},
		test{&paramInput{";foo=", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &empty}, 5}},
		test{&paramInput{";foo", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": nil}, 4}},
		test{&paramInput{";foo=bar!hello", ';', ';', '!', false, true}, &paramResult{pass, map[string]*string{"foo": &bar}, 8}},
		test{&paramInput{";foo!hello", ';', ';', '!', false, true}, &paramResult{pass, map[string]*string{"foo": nil}, 4}},
		test{&paramInput{";foo=!hello", ';', ';', '!', false, true}, &paramResult{pass, map[string]*string{"foo": &empty}, 5}},
		test{&paramInput{";foo=bar!h;l!o", ';', ';', '!', false, true}, &paramResult{pass, map[string]*string{"foo": &bar}, 8}},
		test{&paramInput{";foo!h;l!o", ';', ';', '!', false, true}, &paramResult{pass, map[string]*string{"foo": nil}, 4}},
		test{&paramInput{"foo!h;l!o", ';', ';', '!', false, true}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{"foo;h;l!o", ';', ';', '!', false, true}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{";foo=bar;baz=boop", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": &boop}, 17}},
		test{&paramInput{";foo=bar;baz=boop!lol", ';', ';', '!', false, true}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": &boop}, 17}},
		test{&paramInput{";foo=bar;baz", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": nil}, 12}},
		test{&paramInput{";foo;baz=boop", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": nil, "baz": &boop}, 13}},
		test{&paramInput{";foo=bar;baz=boop;a=b", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": &boop, "a": &b}, 21}},
		test{&paramInput{";foo;baz=boop;a=b", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": nil, "baz": &boop, "a": &b}, 17}},
		test{&paramInput{";foo=bar;baz;a=b", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": nil, "a": &b}, 16}},
		test{&paramInput{";foo=bar;baz=boop;a", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": &boop, "a": nil}, 19}},
		test{&paramInput{";foo=bar;baz=;a", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": &empty, "a": nil}, 15}},
		test{&paramInput{";foo=;baz=bob;a", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &empty, "baz": &bob, "a": nil}, 15}},
		test{&paramInput{"foo=bar", ';', ';', 0, false, true}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{"$foo=bar", '$', ',', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &bar}, 8}},
		test{&paramInput{"$foo", '$', ',', 0, false, true}, &paramResult{pass, map[string]*string{"foo": nil}, 4}},
		test{&paramInput{"$foo=bar!hello", '$', ',', '!', false, true}, &paramResult{pass, map[string]*string{"foo": &bar}, 8}},
		test{&paramInput{"$foo#hello", '$', ',', '#', false, true}, &paramResult{pass, map[string]*string{"foo": nil}, 4}},
		test{&paramInput{"$foo=bar!h;,!o", '$', ',', '!', false, true}, &paramResult{pass, map[string]*string{"foo": &bar}, 8}},
		test{&paramInput{"$foo!h;l!,", '$', ',', '!', false, true}, &paramResult{pass, map[string]*string{"foo": nil}, 4}},
		test{&paramInput{"foo!h;l!o", '$', ',', '!', false, true}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{"foo,h,l!o", '$', ',', '!', false, true}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{"$foo=bar,baz=boop", '$', ',', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": &boop}, 17}},
		test{&paramInput{"$foo=bar;baz", '$', ',', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &barBaz}, 12}},
		test{&paramInput{"$foo=bar,baz=boop!lol", '$', ',', '!', false, true}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": &boop}, 17}},
		test{&paramInput{"$foo=bar,baz", '$', ',', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": nil}, 12}},
		test{&paramInput{"$foo=,baz", '$', ',', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &empty, "baz": nil}, 9}},
		test{&paramInput{"$foo,baz=boop", '$', ',', 0, false, true}, &paramResult{pass, map[string]*string{"foo": nil, "baz": &boop}, 13}},
		test{&paramInput{"$foo=bar,baz=boop,a=b", '$', ',', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": &boop, "a": &b}, 21}},
		test{&paramInput{"$foo,baz=boop,a=b", '$', ',', 0, false, true}, &paramResult{pass, map[string]*string{"foo": nil, "baz": &boop, "a": &b}, 17}},
		test{&paramInput{"$foo=bar,baz,a=b", '$', ',', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": nil, "a": &b}, 16}},
		test{&paramInput{"$foo=bar,baz=boop,a", '$', ',', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": &boop, "a": nil}, 19}},
		test{&paramInput{";foo", ';', ';', 0, false, false}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{";foo=", ';', ';', 0, false, false}, &paramResult{pass, map[string]*string{"foo": &empty}, 5}},
		test{&paramInput{";foo=bar;baz=boop", ';', ';', 0, false, false}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": &boop}, 17}},
		test{&paramInput{";foo=bar;baz", ';', ';', 0, false, false}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{";foo;bar=baz", ';', ';', 0, false, false}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{";foo=;baz=boop", ';', ';', 0, false, false}, &paramResult{pass, map[string]*string{"foo": &empty, "baz": &boop}, 14}},
		test{&paramInput{";foo=bar;baz=", ';', ';', 0, false, false}, &paramResult{pass, map[string]*string{"foo": &bar, "baz": &empty}, 13}},
		test{&paramInput{"$foo=bar,baz=,a=b", '$', ',', 0, false, true}, &paramResult{pass,
			map[string]*string{"foo": &bar, "baz": &empty, "a": &b}, 17}},
		test{&paramInput{"$foo=bar,baz,a=b", '$', ',', 0, false, false}, &paramResult{fail, map[string]*string{}, 17}},
		test{&paramInput{";foo=\"bar\"", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &barQuote}, 10}},
		test{&paramInput{";foo=\"bar", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &barQuote2}, 9}},
		test{&paramInput{";foo=bar\"", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo": &barQuote3}, 9}},
		test{&paramInput{";\"foo\"=bar", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"\"foo\"": &bar}, 10}},
		test{&paramInput{";foo\"=bar", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"foo\"": &bar}, 9}},
		test{&paramInput{";\"foo=bar", ';', ';', 0, false, true}, &paramResult{pass, map[string]*string{"\"foo": &bar}, 9}},
		test{&paramInput{";foo=\"bar\"", ';', ';', 0, true, true}, &paramResult{pass, map[string]*string{"foo": &bar}, 10}},
		test{&paramInput{";foo=\"ba\"r", ';', ';', 0, true, true}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{";foo=ba\"r", ';', ';', 0, true, true}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{";foo=bar\"", ';', ';', 0, true, true}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{";foo=\"bar", ';', ';', 0, true, true}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{";\"foo\"=bar", ';', ';', 0, true, true}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{";\"foo=bar", ';', ';', 0, true, true}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{";foo\"=bar", ';', ';', 0, true, true}, &paramResult{fail, map[string]*string{}, 0}},
		test{&paramInput{";foo=\"bar;baz\"", ';', ';', 0, true, true}, &paramResult{pass, map[string]*string{"foo": &barBaz}, 14}},
		test{&paramInput{";foo=\"bar;baz\";a=b", ';', ';', 0, true, true}, &paramResult{pass, map[string]*string{"foo": &barBaz, "a": &b}, 18}},
		test{&paramInput{";foo=\"bar;baz\";a", ';', ';', 0, true, true}, &paramResult{pass, map[string]*string{"foo": &barBaz, "a": nil}, 16}},
		test{&paramInput{";foo=bar", ';', ';', 0, true, true}, &paramResult{pass, map[string]*string{"foo": &bar}, 8}},
		test{&paramInput{";foo=", ';', ';', 0, true, true}, &paramResult{pass, map[string]*string{"foo": &empty}, 5}},
		test{&paramInput{";foo=\"\"", ';', ';', 0, true, true}, &paramResult{pass, map[string]*string{"foo": &empty}, 7}},
	}, t)
}

func TestSipUris(t *testing.T) {
	doTests([]test{
		test{sipUriInput("sip:bob@example.com"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com"}}},
		test{sipUriInput("sip:bob@192.168.0.1"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "192.168.0.1"}}},
		test{sipUriInput("sip:bob:Hunter2@example.com"), &sipUriResult{pass, base.SipUri{User: &bob, Password: &hunter2, Host: "example.com"}}},
		test{sipUriInput("sips:bob:Hunter2@example.com"), &sipUriResult{pass, base.SipUri{IsEncrypted: true, User: &bob, Password: &hunter2,
			Host: "example.com"}}},
		test{sipUriInput("sips:bob@example.com"), &sipUriResult{pass, base.SipUri{IsEncrypted: true, User: &bob, Host: "example.com"}}},
		test{sipUriInput("sip:example.com"), &sipUriResult{pass, base.SipUri{Host: "example.com"}}},
		test{sipUriInput("example.com"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("bob@example.com"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:5060"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5060}}},
		test{sipUriInput("sip:bob@88.88.88.88:5060"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "88.88.88.88", Port: &ui16_5060}}},
		test{sipUriInput("sip:bob:Hunter2@example.com:5060"), &sipUriResult{pass, base.SipUri{User: &bob, Password: &hunter2,
			Host: "example.com", Port: &ui16_5060}}},
		test{sipUriInput("sip:bob@example.com:5"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5}}},
		test{sipUriInput("sip:bob@example.com;foo=bar"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com",
			UriParams: map[string]*string{"foo": &bar}}}},
		test{sipUriInput("sip:bob@example.com:5060;foo=bar"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5060,
			UriParams: map[string]*string{"foo": &bar}}}},
		test{sipUriInput("sip:bob@example.com:5;foo"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5,
			UriParams: map[string]*string{"foo": nil}}}},
		test{sipUriInput("sip:bob@example.com:5;foo;baz=bar"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5,
			UriParams: map[string]*string{"foo": nil, "baz": &bar}}}},
		test{sipUriInput("sip:bob@example.com:5;baz=bar;foo"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5,
			UriParams: map[string]*string{"foo": nil, "baz": &bar}}}},
		test{sipUriInput("sip:bob@example.com:5;foo;baz=bar;a=b"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5,
			UriParams: map[string]*string{"foo": nil, "baz": &bar, "a": &b}}}},
		test{sipUriInput("sip:bob@example.com:5;baz=bar;foo;a=b"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5,
			UriParams: map[string]*string{"foo": nil, "baz": &bar, "a": &b}}}},
		test{sipUriInput("sip:bob@example.com?foo=bar"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com",
			Headers: map[string]*string{"foo": &bar}}}},
		test{sipUriInput("sip:bob@example.com?foo="), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com",
			Headers: map[string]*string{"foo": &empty}}}},
		test{sipUriInput("sip:bob@example.com:5060?foo=bar"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5060,
			Headers: map[string]*string{"foo": &bar}}}},
		test{sipUriInput("sip:bob@example.com:5?foo=bar"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5,
			Headers: map[string]*string{"foo": &bar}}}},
		test{sipUriInput("sips:bob@example.com:5?baz=bar&foo=&a=b"), &sipUriResult{pass, base.SipUri{IsEncrypted: true, User: &bob, Host: "example.com", Port: &ui16_5,
			Headers: map[string]*string{"baz": &bar, "a": &b,
				"foo": &empty}}}},
		test{sipUriInput("sip:bob@example.com:5?baz=bar&foo&a=b"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:5?foo"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:50?foo"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:50?foo=bar&baz"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com;foo?foo=bar"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com",
			UriParams: map[string]*string{"foo": nil},
			Headers:   map[string]*string{"foo": &bar}}}},
		test{sipUriInput("sip:bob@example.com:5060;foo?foo=bar"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5060,
			UriParams: map[string]*string{"foo": nil},
			Headers:   map[string]*string{"foo": &bar}}}},
		test{sipUriInput("sip:bob@example.com:5;foo?foo=bar"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5,
			UriParams: map[string]*string{"foo": nil},
			Headers:   map[string]*string{"foo": &bar}}}},
		test{sipUriInput("sips:bob@example.com:5;foo?baz=bar&a=b&foo="), &sipUriResult{pass, base.SipUri{IsEncrypted: true, User: &bob,
			Host: "example.com", Port: &ui16_5,
			UriParams: map[string]*string{"foo": nil},
			Headers: map[string]*string{"baz": &bar, "a": &b,
				"foo": &empty}}}},
		test{sipUriInput("sip:bob@example.com:5;foo?baz=bar&foo&a=b"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:5;foo?foo"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:50;foo?foo"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:50;foo?foo=bar&baz"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com;foo=baz?foo=bar"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com",
			UriParams: map[string]*string{"foo": &baz},
			Headers:   map[string]*string{"foo": &bar}}}},
		test{sipUriInput("sip:bob@example.com:5060;foo=baz?foo=bar"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5060,
			UriParams: map[string]*string{"foo": &baz},
			Headers:   map[string]*string{"foo": &bar}}}},
		test{sipUriInput("sip:bob@example.com:5;foo=baz?foo=bar"), &sipUriResult{pass, base.SipUri{User: &bob, Host: "example.com", Port: &ui16_5,
			UriParams: map[string]*string{"foo": &baz},
			Headers:   map[string]*string{"foo": &bar}}}},
		test{sipUriInput("sips:bob@example.com:5;foo=baz?baz=bar&a=b"), &sipUriResult{pass, base.SipUri{IsEncrypted: true, User: &bob, Host: "example.com", Port: &ui16_5,
			UriParams: map[string]*string{"foo": &baz},
			Headers:   map[string]*string{"baz": &bar, "a": &b}}}},
		test{sipUriInput("sip:bob@example.com:5;foo=baz?baz=bar&foo&a=b"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:5;foo=baz?foo"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:50;foo=baz?foo"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:50;foo=baz?foo=bar&baz"), &sipUriResult{fail, base.SipUri{}}},
	}, t)
}

func TestHostPort(t *testing.T) {
	doTests([]test{
		test{hostPortInput("example.com"), &hostPortResult{pass, "example.com", nil}},
		test{hostPortInput("192.168.0.1"), &hostPortResult{pass, "192.168.0.1", nil}},
		test{hostPortInput("abc123"), &hostPortResult{pass, "abc123", nil}},
		test{hostPortInput("example.com:5060"), &hostPortResult{pass, "example.com", &ui16_5060}},
		test{hostPortInput("example.com:9"), &hostPortResult{pass, "example.com", &ui16_9}},
		test{hostPortInput("192.168.0.1:5060"), &hostPortResult{pass, "192.168.0.1", &ui16_5060}},
		test{hostPortInput("192.168.0.1:9"), &hostPortResult{pass, "192.168.0.1", &ui16_9}},
		test{hostPortInput("abc123:5060"), &hostPortResult{pass, "abc123", &ui16_5060}},
		test{hostPortInput("abc123:9"), &hostPortResult{pass, "abc123", &ui16_9}},
		// TODO IPV6, c.f. IPv6reference in RFC 3261 s25
	}, t)
}

type paramInput struct {
	paramString      string
	start            rune
	sep              uint8
	end              uint8
	quoteValues      bool
	permitSingletons bool
}

func (data *paramInput) String() string {
	return fmt.Sprintf("paramString=\"%s\", start=%c, sep=%c, end=%c, quoteValues=%b, permitSingletons=%b",
		data.paramString, data.start, data.sep, data.end, data.quoteValues, data.permitSingletons)
}
func (data *paramInput) evaluate() result {
	output, consumed, err := ParseParams(data.paramString, data.start, data.sep, data.end, data.quoteValues, data.permitSingletons)
	return &paramResult{err, output, consumed}
}

type paramResult struct {
	err      error
	params   map[string]*string
	consumed int
}

func (expected *paramResult) equals(other result) (equal bool, reason string) {
	actual := *(other.(*paramResult))
	if expected.err == nil && actual.err != nil {
		return false, fmt.Sprintf("unexpected error: %s", actual.err.Error())
	} else if expected.err != nil && actual.err == nil {
		return false, fmt.Sprintf("unexpected success: got \"%s\"", base.ParamsToString(actual.params, '$', '-'))
	} else if actual.err == nil && !base.ParamsEqual(expected.params, actual.params) {
		return false, fmt.Sprintf("unexpected result: expected \"%s\", got \"%s\"",
			base.ParamsToString(expected.params, '$', '-'), base.ParamsToString(actual.params, '$', '-'))
	} else if actual.err == nil && expected.consumed != actual.consumed {
		return false, fmt.Sprintf("unexpected consumed value: expected %d, got %d", expected.consumed, actual.consumed)
	}

	return true, ""
}

type sipUriInput string

func (data sipUriInput) String() string {
	return string(data)
}
func (data sipUriInput) evaluate() result {
	output, err := ParseSipUri(string(data))
	return &sipUriResult{err, output}
}

type sipUriResult struct {
	err error
	uri base.SipUri
}

func (expected *sipUriResult) equals(other result) (equal bool, reason string) {
	actual := *(other.(*sipUriResult))
	if expected.err == nil && actual.err != nil {
		return false, fmt.Sprintf("unexpected error: %s", actual.err.Error())
	} else if expected.err != nil && actual.err == nil {
		return false, fmt.Sprintf("unexpected success: got \"%s\"", actual.uri.String())
	} else if actual.err != nil {
		// Expected error. Test passes immediately.
		return true, ""
	}

	equal = expected.uri.Equals(&actual.uri)
	if !equal {
		reason = fmt.Sprintf("expected result %s, but got %s", expected.uri.String(), actual.uri.String())
	}
	return
}

type hostPortInput string

func (data hostPortInput) String() string {
	return string(data)
}

func (data hostPortInput) evaluate() result {
	host, port, err := ParseHostPort(string(data))
	return &hostPortResult{err, host, port}
}

type hostPortResult struct {
	err  error
	host string
	port *uint16
}

func (expected *hostPortResult) equals(other result) (equal bool, reason string) {
	actual := *(other.(*hostPortResult))
	if expected.err == nil && actual.err != nil {
		return false, fmt.Sprintf("unexpected error: %s", actual.err.Error())
	} else if expected.err != nil && actual.err != nil {
		// Expected failure. Return true unconditionally.
		return true, ""
	}

	var actualStr string
	if actual.port == nil {
		actualStr = actual.host
	} else {
		actualStr = fmt.Sprintf("%s:%d", actual.host, actual.port)
	}

	if expected.err != nil && actual.err == nil {
		return false, fmt.Sprintf("unexpected success: got %s", actualStr)
	} else if expected.host != actual.host {
		return false, fmt.Sprintf("unexpected host part: expected \"%s\", got \"%s\"", expected.host, actual.host)
	} else if uint16PtrStr(expected.port) != uint16PtrStr(actual.port) {
		return false, fmt.Sprintf("unexpected port: expected %s, got %s",
			uint16PtrStr(expected.port),
			uint16PtrStr(actual.port))
	}

	return true, ""
}

func TestZZZCountTests(t *testing.T) {
	fmt.Printf("\n *** %d tests run ***", testsRun)
	fmt.Printf("\n *** %d tests passed (%.2f%%) ***\n\n", testsPassed, (float32(testsPassed) * 100.0 / float32(testsRun)))
}

func strPtrStr(strPtr *string) string {
	if strPtr == nil {
		return "nil"
	} else {
		return *strPtr
	}
}

func uint16PtrStr(uint16Ptr *uint16) string {
	if uint16Ptr == nil {
		return "nil"
	} else {
		return strconv.Itoa(int(*uint16Ptr))
	}
}

func errToStr(err error) string {
	if err == nil {
		return "nil"
	} else {
		return err.Error()
	}
}
