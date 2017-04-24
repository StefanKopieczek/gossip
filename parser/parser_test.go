package parser

import (
	"github.com/remodoy/gossip/base"
	"github.com/remodoy/gossip/log"
	"github.com/remodoy/gossip/utils"
)

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
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
		t.Logf("Running test with input: %v", test.args.String())
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
var port5060 uint16 = uint16(5060)
var ui16_5 uint16 = uint16(5)
var ui16_5060 = uint16(5060)
var ui16_9 uint16 = uint16(9)
var noParams = base.NewParams()

func TestAAAASetup(t *testing.T) {
	log.SetDefaultLogLevel(c_LOG_LEVEL)
}

func TestParams(t *testing.T) {
	doTests([]test{
		// TEST: parseParams
		test{&paramInput{";foo=bar", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}), 8}},
		test{&paramInput{";foo=", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{""}), 5}},
		test{&paramInput{";foo", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.NoString{}), 4}},
		test{&paramInput{";foo=bar!hello", ';', ';', '!', false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}), 8}},
		test{&paramInput{";foo!hello", ';', ';', '!', false, true}, &paramResult{pass, base.NewParams().Add("foo", base.NoString{}), 4}},
		test{&paramInput{";foo=!hello", ';', ';', '!', false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{""}), 5}},
		test{&paramInput{";foo=bar!h;l!o", ';', ';', '!', false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}), 8}},
		test{&paramInput{";foo!h;l!o", ';', ';', '!', false, true}, &paramResult{pass, base.NewParams().Add("foo", base.NoString{}), 4}},
		test{&paramInput{"foo!h;l!o", ';', ';', '!', false, true}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{"foo;h;l!o", ';', ';', '!', false, true}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{";foo=bar;baz=boop", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.String{"boop"}), 17}},
		test{&paramInput{";foo=bar;baz=boop!lol", ';', ';', '!', false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.String{"boop"}), 17}},
		test{&paramInput{";foo=bar;baz", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.NoString{}), 12}},
		test{&paramInput{";foo;baz=boop", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.NoString{}).Add("baz", base.String{"boop"}), 13}},
		test{&paramInput{";foo=bar;baz=boop;a=b", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.String{"boop"}).Add("a", base.String{"b"}), 21}},
		test{&paramInput{";foo;baz=boop;a=b", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.NoString{}).Add("baz", base.String{"boop"}).Add("a", base.String{"b"}), 17}},
		test{&paramInput{";foo=bar;baz;a=b", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.NoString{}).Add("a", base.String{"b"}), 16}},
		test{&paramInput{";foo=bar;baz=boop;a", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.String{"boop"}).Add("a", base.NoString{}), 19}},
		test{&paramInput{";foo=bar;baz=;a", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.String{""}).Add("a", base.NoString{}), 15}},
		test{&paramInput{";foo=;baz=bob;a", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{""}).Add("baz", base.String{"bob"}).Add("a", base.NoString{}), 15}},
		test{&paramInput{"foo=bar", ';', ';', 0, false, true}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{"$foo=bar", '$', ',', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}), 8}},
		test{&paramInput{"$foo", '$', ',', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.NoString{}), 4}},
		test{&paramInput{"$foo=bar!hello", '$', ',', '!', false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}), 8}},
		test{&paramInput{"$foo#hello", '$', ',', '#', false, true}, &paramResult{pass, base.NewParams().Add("foo", base.NoString{}), 4}},
		test{&paramInput{"$foo=bar!h;,!o", '$', ',', '!', false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}), 8}},
		test{&paramInput{"$foo!h;l!,", '$', ',', '!', false, true}, &paramResult{pass, base.NewParams().Add("foo", base.NoString{}), 4}},
		test{&paramInput{"foo!h;l!o", '$', ',', '!', false, true}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{"foo,h,l!o", '$', ',', '!', false, true}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{"$foo=bar,baz=boop", '$', ',', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.String{"boop"}), 17}},
		test{&paramInput{"$foo=bar;baz", '$', ',', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar;baz"}), 12}},
		test{&paramInput{"$foo=bar,baz=boop!lol", '$', ',', '!', false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.String{"boop"}), 17}},
		test{&paramInput{"$foo=bar,baz", '$', ',', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.NoString{}), 12}},
		test{&paramInput{"$foo=,baz", '$', ',', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{""}).Add("baz", base.NoString{}), 9}},
		test{&paramInput{"$foo,baz=boop", '$', ',', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.NoString{}).Add("baz", base.String{"boop"}), 13}},
		test{&paramInput{"$foo=bar,baz=boop,a=b", '$', ',', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.String{"boop"}).Add("a", base.String{"b"}), 21}},
		test{&paramInput{"$foo,baz=boop,a=b", '$', ',', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.NoString{}).Add("baz", base.String{"boop"}).Add("a", base.String{"b"}), 17}},
		test{&paramInput{"$foo=bar,baz,a=b", '$', ',', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.NoString{}).Add("a", base.String{"b"}), 16}},
		test{&paramInput{"$foo=bar,baz=boop,a", '$', ',', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.String{"boop"}).Add("a", base.NoString{}), 19}},
		test{&paramInput{";foo", ';', ';', 0, false, false}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{";foo=", ';', ';', 0, false, false}, &paramResult{pass, base.NewParams().Add("foo", base.String{""}), 5}},
		test{&paramInput{";foo=bar;baz=boop", ';', ';', 0, false, false}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.String{"boop"}), 17}},
		test{&paramInput{";foo=bar;baz", ';', ';', 0, false, false}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{";foo;bar=baz", ';', ';', 0, false, false}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{";foo=;baz=boop", ';', ';', 0, false, false}, &paramResult{pass, base.NewParams().Add("foo", base.String{""}).Add("baz", base.String{"boop"}), 14}},
		test{&paramInput{";foo=bar;baz=", ';', ';', 0, false, false}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.String{""}), 13}},
		test{&paramInput{"$foo=bar,baz=,a=b", '$', ',', 0, false, true}, &paramResult{pass,
			base.NewParams().Add("foo", base.String{"bar"}).Add("baz", base.String{""}).Add("a", base.String{"b"}), 17}},
		test{&paramInput{"$foo=bar,baz,a=b", '$', ',', 0, false, false}, &paramResult{fail, base.NewParams(), 17}},
		test{&paramInput{";foo=\"bar\"", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"\"bar\""}), 10}},
		test{&paramInput{";foo=\"bar", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"\"bar"}), 9}},
		test{&paramInput{";foo=bar\"", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar\""}), 9}},
		test{&paramInput{";\"foo\"=bar", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("\"foo\"", base.String{"bar"}), 10}},
		test{&paramInput{";foo\"=bar", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("foo\"", base.String{"bar"}), 9}},
		test{&paramInput{";\"foo=bar", ';', ';', 0, false, true}, &paramResult{pass, base.NewParams().Add("\"foo", base.String{"bar"}), 9}},
		test{&paramInput{";foo=\"bar\"", ';', ';', 0, true, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}), 10}},
		test{&paramInput{";foo=\"ba\"r", ';', ';', 0, true, true}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{";foo=ba\"r", ';', ';', 0, true, true}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{";foo=bar\"", ';', ';', 0, true, true}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{";foo=\"bar", ';', ';', 0, true, true}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{";\"foo\"=bar", ';', ';', 0, true, true}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{";\"foo=bar", ';', ';', 0, true, true}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{";foo\"=bar", ';', ';', 0, true, true}, &paramResult{fail, base.NewParams(), 0}},
		test{&paramInput{";foo=\"bar;baz\"", ';', ';', 0, true, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar;baz"}), 14}},
		test{&paramInput{";foo=\"bar;baz\";a=b", ';', ';', 0, true, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar;baz"}).Add("a", base.String{"b"}), 18}},
		test{&paramInput{";foo=\"bar;baz\";a", ';', ';', 0, true, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar;baz"}).Add("a", base.NoString{}), 16}},
		test{&paramInput{";foo=bar", ';', ';', 0, true, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{"bar"}), 8}},
		test{&paramInput{";foo=", ';', ';', 0, true, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{""}), 5}},
		test{&paramInput{";foo=\"\"", ';', ';', 0, true, true}, &paramResult{pass, base.NewParams().Add("foo", base.String{""}), 7}},
	}, t)
}

func TestSipUris(t *testing.T) {
	doTests([]test{
		test{sipUriInput("sip:bob@example.com"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", UriParams: noParams, Headers: noParams}}},
		test{sipUriInput("sip:bob@192.168.0.1"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "192.168.0.1", UriParams: noParams, Headers: noParams}}},
		test{sipUriInput("sip:bob:Hunter2@example.com"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.String{"Hunter2"}, Host: "example.com", UriParams: noParams, Headers: noParams}}},
		test{sipUriInput("sips:bob:Hunter2@example.com"), &sipUriResult{pass, base.SipUri{IsEncrypted: true, User: base.String{"bob"}, Password: base.String{"Hunter2"},
			Host: "example.com", UriParams: noParams, Headers: noParams}}},
		test{sipUriInput("sips:bob@example.com"), &sipUriResult{pass, base.SipUri{IsEncrypted: true, User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", UriParams: noParams, Headers: noParams}}},
		test{sipUriInput("sip:example.com"), &sipUriResult{pass, base.SipUri{User: base.NoString{}, Password: base.NoString{}, Host: "example.com", UriParams: noParams, Headers: noParams}}},
		test{sipUriInput("example.com"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("bob@example.com"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:5060"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5060, UriParams: noParams, Headers: noParams}}},
		test{sipUriInput("sip:bob@88.88.88.88:5060"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "88.88.88.88", Port: &ui16_5060, UriParams: noParams, Headers: noParams}}},
		test{sipUriInput("sip:bob:Hunter2@example.com:5060"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.String{"Hunter2"},
			Host: "example.com", Port: &ui16_5060, UriParams: noParams, Headers: noParams}}},
		test{sipUriInput("sip:bob@example.com:5"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5, UriParams: noParams, Headers: noParams}}},
		test{sipUriInput("sip:bob@example.com;foo=bar"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com",
			UriParams: base.NewParams().Add("foo", base.String{"bar"}), Headers: noParams}}},
		test{sipUriInput("sip:bob@example.com:5060;foo=bar"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5060,
			UriParams: base.NewParams().Add("foo", base.String{"bar"}), Headers: noParams}}},
		test{sipUriInput("sip:bob@example.com:5;foo"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5,
			UriParams: base.NewParams().Add("foo", base.NoString{}), Headers: noParams}}},
		test{sipUriInput("sip:bob@example.com:5;foo;baz=bar"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5,
			UriParams: base.NewParams().Add("foo", base.NoString{}).Add("baz", base.String{"bar"}), Headers: noParams}}},
		test{sipUriInput("sip:bob@example.com:5;baz=bar;foo"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5,
			UriParams: base.NewParams().Add("foo", base.NoString{}).Add("baz", base.String{"bar"}), Headers: noParams}}},
		test{sipUriInput("sip:bob@example.com:5;foo;baz=bar;a=b"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5,
			UriParams: base.NewParams().Add("foo", base.NoString{}).Add("baz", base.String{"bar"}).Add("a", base.String{"b"}), Headers: noParams}}},
		test{sipUriInput("sip:bob@example.com:5;baz=bar;foo;a=b"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5,
			UriParams: base.NewParams().Add("foo", base.NoString{}).Add("baz", base.String{"bar"}).Add("a", base.String{"b"}), Headers: noParams}}},
		test{sipUriInput("sip:bob@example.com?foo=bar"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com",
			UriParams: noParams, Headers: base.NewParams().Add("foo", base.String{"bar"})}}},
		test{sipUriInput("sip:bob@example.com?foo="), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com",
			UriParams: noParams, Headers: base.NewParams().Add("foo", base.String{""})}}},
		test{sipUriInput("sip:bob@example.com:5060?foo=bar"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5060,
			UriParams: noParams, Headers: base.NewParams().Add("foo", base.String{"bar"})}}},
		test{sipUriInput("sip:bob@example.com:5?foo=bar"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5,
			UriParams: noParams, Headers: base.NewParams().Add("foo", base.String{"bar"})}}},
		test{sipUriInput("sips:bob@example.com:5?baz=bar&foo=&a=b"), &sipUriResult{pass, base.SipUri{IsEncrypted: true, User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5,
			UriParams: noParams, Headers: base.NewParams().Add("baz", base.String{"bar"}).Add("a", base.String{"b"}).Add("foo", base.String{""})}}},
		test{sipUriInput("sip:bob@example.com:5?baz=bar&foo&a=b"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:5?foo"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:50?foo"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:50?foo=bar&baz"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com;foo?foo=bar"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com",
			UriParams: base.NewParams().Add("foo", base.NoString{}),
			Headers:   base.NewParams().Add("foo", base.String{"bar"})}}},
		test{sipUriInput("sip:bob@example.com:5060;foo?foo=bar"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5060,
			UriParams: base.NewParams().Add("foo", base.NoString{}),
			Headers:   base.NewParams().Add("foo", base.String{"bar"})}}},
		test{sipUriInput("sip:bob@example.com:5;foo?foo=bar"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5,
			UriParams: base.NewParams().Add("foo", base.NoString{}),
			Headers:   base.NewParams().Add("foo", base.String{"bar"})}}},
		test{sipUriInput("sips:bob@example.com:5;foo?baz=bar&a=b&foo="), &sipUriResult{pass, base.SipUri{IsEncrypted: true, User: base.String{"bob"},
			Password: base.NoString{}, Host: "example.com", Port: &ui16_5,
			UriParams: base.NewParams().Add("foo", base.NoString{}),
			Headers:   base.NewParams().Add("baz", base.String{"bar"}).Add("a", base.String{"b"}).Add("foo", base.String{""})}}},
		test{sipUriInput("sip:bob@example.com:5;foo?baz=bar&foo&a=b"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:5;foo?foo"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:50;foo?foo"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com:50;foo?foo=bar&baz"), &sipUriResult{fail, base.SipUri{}}},
		test{sipUriInput("sip:bob@example.com;foo=baz?foo=bar"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com",
			UriParams: base.NewParams().Add("foo", base.String{"baz"}),
			Headers:   base.NewParams().Add("foo", base.String{"bar"})}}},
		test{sipUriInput("sip:bob@example.com:5060;foo=baz?foo=bar"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5060,
			UriParams: base.NewParams().Add("foo", base.String{"baz"}),
			Headers:   base.NewParams().Add("foo", base.String{"bar"})}}},
		test{sipUriInput("sip:bob@example.com:5;foo=baz?foo=bar"), &sipUriResult{pass, base.SipUri{User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5,
			UriParams: base.NewParams().Add("foo", base.String{"baz"}),
			Headers:   base.NewParams().Add("foo", base.String{"bar"})}}},
		test{sipUriInput("sips:bob@example.com:5;foo=baz?baz=bar&a=b"), &sipUriResult{pass, base.SipUri{IsEncrypted: true, User: base.String{"bob"}, Password: base.NoString{}, Host: "example.com", Port: &ui16_5,
			UriParams: base.NewParams().Add("foo", base.String{"baz"}),
			Headers:   base.NewParams().Add("baz", base.String{"bar"}).Add("a", base.String{"b"})}}},
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

/*
func TestHeaderBlocks(t *testing.T) {
	doTests([]test{
		test{headerBlockInput([]string{"All on one line."}), &headerBlockResult{"All on one line.", 1}},
		test{headerBlockInput([]string{"Line one", "Line two."}), &headerBlockResult{"Line one", 1}},
		test{headerBlockInput([]string{"Line one", " then an indent"}), &headerBlockResult{"Line one then an indent", 2}},
		test{headerBlockInput([]string{"Line one", " then an indent", "then line two"}), &headerBlockResult{"Line one then an indent", 2}},
		test{headerBlockInput([]string{"Line one", "Line two", " then an indent"}), &headerBlockResult{"Line one", 1}},
		test{headerBlockInput([]string{"Line one", "\twith tab indent"}), &headerBlockResult{"Line one with tab indent", 2}},
		test{headerBlockInput([]string{"Line one", "      with a big indent"}), &headerBlockResult{"Line one with a big indent", 2}},
		test{headerBlockInput([]string{"Line one", " \twith space then tab"}), &headerBlockResult{"Line one with space then tab", 2}},
		test{headerBlockInput([]string{"Line one", "\t    with tab then spaces"}), &headerBlockResult{"Line one with tab then spaces", 2}},
		test{headerBlockInput([]string{""}), &headerBlockResult{"", 0}},
		test{headerBlockInput([]string{" "}), &headerBlockResult{" ", 1}},
		test{headerBlockInput([]string{}), &headerBlockResult{"", 0}},
		test{headerBlockInput([]string{" foo"}), &headerBlockResult{" foo", 1}},
	}, t)
}
*/
func TestToHeaders(t *testing.T) {
	fooEqBar := base.NewParams().Add("foo", base.String{"bar"})
	fooSingleton := base.NewParams().Add("foo", base.NoString{})
	doTests([]test{
		test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{toHeaderInput("To : \"Alice Liddell\" <sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{toHeaderInput("To  : \"Alice Liddell\" <sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{toHeaderInput("To\t: \"Alice Liddell\" <sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{toHeaderInput("To:\n  \"Alice Liddell\" \n\t<sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{toHeaderInput("t: Alice <sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{toHeaderInput("To: Alice sip:alice@wonderland.com"), &toHeaderResult{fail,
			&base.ToHeader{}}},

		test{toHeaderInput("To:"), &toHeaderResult{fail,
			&base.ToHeader{}}},

		test{toHeaderInput("To: "), &toHeaderResult{fail,
			&base.ToHeader{}}},

		test{toHeaderInput("To:\t"), &toHeaderResult{fail,
			&base.ToHeader{}}},

		test{toHeaderInput("To: foo"), &toHeaderResult{fail,
			&base.ToHeader{}}},

		test{toHeaderInput("To: foo bar"), &toHeaderResult{fail,
			&base.ToHeader{}}},

		test{toHeaderInput("To: \"Alice\" sip:alice@wonderland.com"), &toHeaderResult{fail,
			&base.ToHeader{}}},

		test{toHeaderInput("To: \"<Alice>\" sip:alice@wonderland.com"), &toHeaderResult{fail,
			&base.ToHeader{}}},

		test{toHeaderInput("To: \"sip:alice@wonderland.com\""), &toHeaderResult{fail,
			&base.ToHeader{}}},

		test{toHeaderInput("To: \"sip:alice@wonderland.com\"  <sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"sip:alice@wonderland.com"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{toHeaderInput("T: \"<sip:alice@wonderland.com>\"  <sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"<sip:alice@wonderland.com>"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{toHeaderInput("To: \"<sip: alice@wonderland.com>\"  <sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"<sip: alice@wonderland.com>"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com>;foo=bar"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  fooEqBar}}},

		test{toHeaderInput("To: sip:alice@wonderland.com;foo=bar"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.NoString{},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  fooEqBar}}},

		test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com;foo=bar>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, fooEqBar, noParams},
				Params:  noParams}}},

		test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com?foo=bar>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, fooEqBar},
				Params:  noParams}}},

		test{toHeaderInput("to: \"Alice Liddell\" <sip:alice@wonderland.com>;foo"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  fooSingleton}}},

		test{toHeaderInput("TO: \"Alice Liddell\" <sip:alice@wonderland.com;foo>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, fooSingleton, noParams},
				Params:  noParams}}},

		test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com?foo>"), &toHeaderResult{fail,
			&base.ToHeader{}}},

		test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com;foo?foo=bar>;foo=bar"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, fooSingleton, fooEqBar},
				Params:  fooEqBar}}},

		test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com;foo?foo=bar>;foo"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, fooSingleton, fooEqBar},
				Params:  fooSingleton}}},

		test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{toHeaderInput("To: sip:alice@wonderland.com, sip:hatter@wonderland.com"), &toHeaderResult{fail,
			&base.ToHeader{}}},

		test{toHeaderInput("To: *"), &toHeaderResult{fail, &base.ToHeader{}}},

		test{toHeaderInput("To: <*>"), &toHeaderResult{fail, &base.ToHeader{}}},

		test{toHeaderInput("To: \"Alice Liddell\"<sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{toHeaderInput("To: Alice Liddell <sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{toHeaderInput("To: Alice Liddell<sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{toHeaderInput("To: Alice<sip:alice@wonderland.com>"), &toHeaderResult{pass,
			&base.ToHeader{DisplayName: base.String{"Alice"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},
	}, t)
}

func TestFromHeaders(t *testing.T) {
	// These are identical to the To: header tests, but there's no clean way to share them :(
	fooEqBar := base.NewParams().Add("foo", base.String{"bar"})
	fooSingleton := base.NewParams().Add("foo", base.NoString{})
	doTests([]test{
		test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{fromHeaderInput("From : \"Alice Liddell\" <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{fromHeaderInput("From   : \"Alice Liddell\" <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{fromHeaderInput("From\t: \"Alice Liddell\" <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{fromHeaderInput("From:\n  \"Alice Liddell\" \n\t<sip:alice@wonderland.com>"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{fromHeaderInput("f: Alice <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{fromHeaderInput("From: Alice sip:alice@wonderland.com"), &fromHeaderResult{fail,
			&base.FromHeader{}}},

		test{fromHeaderInput("From:"), &fromHeaderResult{fail,
			&base.FromHeader{}}},

		test{fromHeaderInput("From: "), &fromHeaderResult{fail,
			&base.FromHeader{}}},

		test{fromHeaderInput("From:\t"), &fromHeaderResult{fail,
			&base.FromHeader{}}},

		test{fromHeaderInput("From: foo"), &fromHeaderResult{fail,
			&base.FromHeader{}}},

		test{fromHeaderInput("From: foo bar"), &fromHeaderResult{fail,
			&base.FromHeader{}}},

		test{fromHeaderInput("From: \"Alice\" sip:alice@wonderland.com"), &fromHeaderResult{fail,
			&base.FromHeader{}}},

		test{fromHeaderInput("From: \"<Alice>\" sip:alice@wonderland.com"), &fromHeaderResult{fail,
			&base.FromHeader{}}},

		test{fromHeaderInput("From: \"sip:alice@wonderland.com\""), &fromHeaderResult{fail,
			&base.FromHeader{}}},

		test{fromHeaderInput("From: \"sip:alice@wonderland.com\"  <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"sip:alice@wonderland.com"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{fromHeaderInput("From: \"<sip:alice@wonderland.com>\"  <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"<sip:alice@wonderland.com>"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{fromHeaderInput("From: \"<sip: alice@wonderland.com>\"  <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"<sip: alice@wonderland.com>"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{fromHeaderInput("FrOm: \"Alice Liddell\" <sip:alice@wonderland.com>;foo=bar"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  fooEqBar}}},

		test{fromHeaderInput("FrOm: sip:alice@wonderland.com;foo=bar"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.NoString{},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  fooEqBar}}},

		test{fromHeaderInput("from: \"Alice Liddell\" <sip:alice@wonderland.com;foo=bar>"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, fooEqBar, noParams},
				Params:  noParams}}},

		test{fromHeaderInput("F: \"Alice Liddell\" <sip:alice@wonderland.com?foo=bar>"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, fooEqBar},
				Params:  noParams}}},

		test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com>;foo"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  fooSingleton}}},

		test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com;foo>"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, fooSingleton, noParams},
				Params:  noParams}}},

		test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com?foo>"), &fromHeaderResult{fail,
			&base.FromHeader{}}},

		test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com;foo?foo=bar>;foo=bar"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, fooSingleton, fooEqBar},
				Params:  fooEqBar}}},

		test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com;foo?foo=bar>;foo"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, fooSingleton, fooEqBar},
				Params:  fooSingleton}}},

		test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
			&base.FromHeader{DisplayName: base.String{"Alice Liddell"},
				Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
				Params:  noParams}}},

		test{fromHeaderInput("From: sip:alice@wonderland.com, sip:hatter@wonderland.com"), &fromHeaderResult{fail,
			&base.FromHeader{}}},

		test{fromHeaderInput("From: *"), &fromHeaderResult{fail, &base.FromHeader{}}},

		test{fromHeaderInput("From: <*>"), &fromHeaderResult{fail, &base.FromHeader{}}},
	}, t)
}

func TestContactHeaders(t *testing.T) {
	fooEqBar := base.NewParams().Add("foo", base.String{"bar"})
	fooSingleton := base.NewParams().Add("foo", base.NoString{})
	doTests([]test{
		test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("Contact : \"Alice Liddell\" <sip:alice@wonderland.com>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams}}}},
		test{contactHeaderInput("Contact  : \"Alice Liddell\" <sip:alice@wonderland.com>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams}}}},
		test{contactHeaderInput("Contact\t: \"Alice Liddell\" <sip:alice@wonderland.com>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams}}}},
		test{contactHeaderInput("Contact:\n  \"Alice Liddell\" \n\t<sip:alice@wonderland.com>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("m: Alice <sip:alice@wonderland.com>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("Contact: *"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.NoString{}, Address: &base.WildcardUri{}, Params: noParams}}}},

		test{contactHeaderInput("Contact: \t  *"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.NoString{}, Address: &base.WildcardUri{}, Params: noParams}}}},

		test{contactHeaderInput("M: *"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.NoString{}, Address: &base.WildcardUri{}, Params: noParams}}}},

		test{contactHeaderInput("Contact: *"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.NoString{}, Address: &base.WildcardUri{}, Params: noParams}}}},

		test{contactHeaderInput("Contact: \"John\" *"), &contactHeaderResult{
			fail,
			[]*base.ContactHeader{}}},

		test{contactHeaderInput("Contact: \"John\" <*>"), &contactHeaderResult{
			fail,
			[]*base.ContactHeader{}}},

		test{contactHeaderInput("Contact: *;foo=bar"), &contactHeaderResult{
			fail,
			[]*base.ContactHeader{}}},

		test{contactHeaderInput("Contact: Alice sip:alice@wonderland.com"), &contactHeaderResult{
			fail,
			[]*base.ContactHeader{
				&base.ContactHeader{}}}},

		test{contactHeaderInput("Contact:"), &contactHeaderResult{
			fail,
			[]*base.ContactHeader{
				&base.ContactHeader{}}}},

		test{contactHeaderInput("Contact: "), &contactHeaderResult{
			fail,
			[]*base.ContactHeader{
				&base.ContactHeader{}}}},

		test{contactHeaderInput("Contact:\t"), &contactHeaderResult{
			fail,
			[]*base.ContactHeader{
				&base.ContactHeader{}}}},

		test{contactHeaderInput("Contact: foo"), &contactHeaderResult{
			fail,
			[]*base.ContactHeader{
				&base.ContactHeader{}}}},

		test{contactHeaderInput("Contact: foo bar"), &contactHeaderResult{
			fail,
			[]*base.ContactHeader{
				&base.ContactHeader{}}}},

		test{contactHeaderInput("Contact: \"Alice\" sip:alice@wonderland.com"), &contactHeaderResult{
			fail,
			[]*base.ContactHeader{
				&base.ContactHeader{}}}},

		test{contactHeaderInput("Contact: \"<Alice>\" sip:alice@wonderland.com"), &contactHeaderResult{
			fail,
			[]*base.ContactHeader{
				&base.ContactHeader{}}}},

		test{contactHeaderInput("Contact: \"sip:alice@wonderland.com\""), &contactHeaderResult{
			fail,
			[]*base.ContactHeader{
				&base.ContactHeader{}}}},

		test{contactHeaderInput("Contact: \"sip:alice@wonderland.com\"  <sip:alice@wonderland.com>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"sip:alice@wonderland.com"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("Contact: \"<sip:alice@wonderland.com>\"  <sip:alice@wonderland.com>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"<sip:alice@wonderland.com>"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("Contact: \"<sip: alice@wonderland.com>\"  <sip:alice@wonderland.com>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"<sip: alice@wonderland.com>"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("cOntACt: \"Alice Liddell\" <sip:alice@wonderland.com>;foo=bar"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  fooEqBar}}}},

		test{contactHeaderInput("contact: \"Alice Liddell\" <sip:alice@wonderland.com;foo=bar>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, fooEqBar, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("M: \"Alice Liddell\" <sip:alice@wonderland.com?foo=bar>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, fooEqBar},
					Params:  noParams}}}},

		test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com>;foo"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  fooSingleton}}}},

		test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com;foo>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, fooSingleton, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com?foo>"), &contactHeaderResult{
			fail,
			[]*base.ContactHeader{
				&base.ContactHeader{}}}},

		test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com;foo?foo=bar>;foo=bar"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, fooSingleton, fooEqBar},
					Params:  fooEqBar}}}},

		test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com;foo?foo=bar>;foo"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, fooSingleton, fooEqBar},
					Params:  fooSingleton}}}},

		test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("Contact: sip:alice@wonderland.com, sip:hatter@wonderland.com"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.NoString{}, Address: &base.SipUri{false, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams}, Params: noParams},
				&base.ContactHeader{DisplayName: base.NoString{}, Address: &base.SipUri{false, base.String{"hatter"}, base.NoString{}, "wonderland.com", nil, noParams, noParams}, Params: noParams}}}},

		test{contactHeaderInput("Contact: \"Alice Liddell\" <sips:alice@wonderland.com>, \"Madison Hatter\" <sip:hatter@wonderland.com>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{true, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams},
				&base.ContactHeader{DisplayName: base.String{"Madison Hatter"},
					Address: &base.SipUri{false, base.String{"hatter"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("Contact: <sips:alice@wonderland.com>, \"Madison Hatter\" <sip:hatter@wonderland.com>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.NoString{},
					Address: &base.SipUri{true, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams},
				&base.ContactHeader{DisplayName: base.String{"Madison Hatter"},
					Address: &base.SipUri{false, base.String{"hatter"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("Contact: \"Alice Liddell\" <sips:alice@wonderland.com>, <sip:hatter@wonderland.com>"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{true, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams},
				&base.ContactHeader{DisplayName: base.NoString{},
					Address: &base.SipUri{false, base.String{"hatter"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("Contact: \"Alice Liddell\" <sips:alice@wonderland.com>, \"Madison Hatter\" <sip:hatter@wonderland.com>" +
			",    sip:kat@cheshire.gov.uk"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{true, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams},
				&base.ContactHeader{DisplayName: base.String{"Madison Hatter"},
					Address: &base.SipUri{false, base.String{"hatter"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams},
				&base.ContactHeader{DisplayName: base.NoString{},
					Address: &base.SipUri{false, base.String{"kat"}, base.NoString{}, "cheshire.gov.uk", nil, noParams, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("Contact: \"Alice Liddell\" <sips:alice@wonderland.com>;foo=bar, \"Madison Hatter\" <sip:hatter@wonderland.com>" +
			",    sip:kat@cheshire.gov.uk"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{true, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  fooEqBar},
				&base.ContactHeader{DisplayName: base.String{"Madison Hatter"},
					Address: &base.SipUri{false, base.String{"hatter"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams},
				&base.ContactHeader{DisplayName: base.NoString{},
					Address: &base.SipUri{false, base.String{"kat"}, base.NoString{}, "cheshire.gov.uk", nil, noParams, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("Contact: \"Alice Liddell\" <sips:alice@wonderland.com>, \"Madison Hatter\" <sip:hatter@wonderland.com>;foo=bar" +
			",    sip:kat@cheshire.gov.uk"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{true, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams},
				&base.ContactHeader{DisplayName: base.String{"Madison Hatter"},
					Address: &base.SipUri{false, base.String{"hatter"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  fooEqBar},
				&base.ContactHeader{DisplayName: base.NoString{},
					Address: &base.SipUri{false, base.String{"kat"}, base.NoString{}, "cheshire.gov.uk", nil, noParams, noParams},
					Params:  noParams}}}},

		test{contactHeaderInput("Contact: \"Alice Liddell\" <sips:alice@wonderland.com>, \"Madison Hatter\" <sip:hatter@wonderland.com>" +
			",    sip:kat@cheshire.gov.uk;foo=bar"), &contactHeaderResult{
			pass,
			[]*base.ContactHeader{
				&base.ContactHeader{DisplayName: base.String{"Alice Liddell"},
					Address: &base.SipUri{true, base.String{"alice"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams},
				&base.ContactHeader{DisplayName: base.String{"Madison Hatter"},
					Address: &base.SipUri{false, base.String{"hatter"}, base.NoString{}, "wonderland.com", nil, noParams, noParams},
					Params:  noParams},
				&base.ContactHeader{DisplayName: base.NoString{},
					Address: &base.SipUri{false, base.String{"kat"}, base.NoString{}, "cheshire.gov.uk", nil, noParams, noParams},
					Params:  fooEqBar}}}},
	}, t)
}

func TestSplitByWS(t *testing.T) {
	doTests([]test{
		test{splitByWSInput("Hello world"), splitByWSResult([]string{"Hello", "world"})},
		test{splitByWSInput("Hello\tworld"), splitByWSResult([]string{"Hello", "world"})},
		test{splitByWSInput("Hello    world"), splitByWSResult([]string{"Hello", "world"})},
		test{splitByWSInput("Hello  world"), splitByWSResult([]string{"Hello", "world"})},
		test{splitByWSInput("Hello\t world"), splitByWSResult([]string{"Hello", "world"})},
		test{splitByWSInput("Hello\t world"), splitByWSResult([]string{"Hello", "world"})},
		test{splitByWSInput("Hello\t \tworld"), splitByWSResult([]string{"Hello", "world"})},
		test{splitByWSInput("Hello\t\tworld"), splitByWSResult([]string{"Hello", "world"})},
		test{splitByWSInput("Hello\twonderful\tworld"), splitByWSResult([]string{"Hello", "wonderful", "world"})},
		test{splitByWSInput("Hello   wonderful\tworld"), splitByWSResult([]string{"Hello", "wonderful", "world"})},
		test{splitByWSInput("Hello   wonderful  world"), splitByWSResult([]string{"Hello", "wonderful", "world"})},
	}, t)
}

func TestCSeqs(t *testing.T) {
	doTests([]test{
		test{cSeqInput("CSeq: 1 INVITE"), &cSeqResult{pass, &base.CSeq{1, "INVITE"}}},
		test{cSeqInput("CSeq : 2 INVITE"), &cSeqResult{pass, &base.CSeq{2, "INVITE"}}},
		test{cSeqInput("CSeq  : 3 INVITE"), &cSeqResult{pass, &base.CSeq{3, "INVITE"}}},
		test{cSeqInput("CSeq\t: 4 INVITE"), &cSeqResult{pass, &base.CSeq{4, "INVITE"}}},
		test{cSeqInput("CSeq:\t5\t\tINVITE"), &cSeqResult{pass, &base.CSeq{5, "INVITE"}}},
		test{cSeqInput("CSeq:\t6 \tINVITE"), &cSeqResult{pass, &base.CSeq{6, "INVITE"}}},
		test{cSeqInput("CSeq:    7      INVITE"), &cSeqResult{pass, &base.CSeq{7, "INVITE"}}},
		test{cSeqInput("CSeq: 8  INVITE"), &cSeqResult{pass, &base.CSeq{8, "INVITE"}}},
		test{cSeqInput("CSeq: 0 register"), &cSeqResult{pass, &base.CSeq{0, "register"}}},
		test{cSeqInput("CSeq: 10 reGister"), &cSeqResult{pass, &base.CSeq{10, "reGister"}}},
		test{cSeqInput("CSeq: 17 FOOBAR"), &cSeqResult{pass, &base.CSeq{17, "FOOBAR"}}},
		test{cSeqInput("CSeq: 2147483647 NOTIFY"), &cSeqResult{pass, &base.CSeq{2147483647, "NOTIFY"}}},
		test{cSeqInput("CSeq: 2147483648 NOTIFY"), &cSeqResult{fail, &base.CSeq{}}},
		test{cSeqInput("CSeq: -124 ACK"), &cSeqResult{fail, &base.CSeq{}}},
		test{cSeqInput("CSeq: 1"), &cSeqResult{fail, &base.CSeq{}}},
		test{cSeqInput("CSeq: ACK"), &cSeqResult{fail, &base.CSeq{}}},
		test{cSeqInput("CSeq:"), &cSeqResult{fail, &base.CSeq{}}},
		test{cSeqInput("CSeq: FOO ACK"), &cSeqResult{fail, &base.CSeq{}}},
		test{cSeqInput("CSeq: 9999999999999999999999999999999 SUBSCRIBE"), &cSeqResult{fail, &base.CSeq{}}},
		test{cSeqInput("CSeq: 1 INVITE;foo=bar"), &cSeqResult{fail, &base.CSeq{}}},
		test{cSeqInput("CSeq: 1 INVITE;foo"), &cSeqResult{fail, &base.CSeq{}}},
		test{cSeqInput("CSeq: 1 INVITE;foo=bar;baz"), &cSeqResult{fail, &base.CSeq{}}},
	}, t)
}

func TestCallIds(t *testing.T) {
	doTests([]test{
		test{callIdInput("Call-ID: fdlknfa32bse3yrbew23bf"), &callIdResult{pass, base.CallId("fdlknfa32bse3yrbew23bf")}},
		test{callIdInput("Call-ID : fdlknfa32bse3yrbew23bf"), &callIdResult{pass, base.CallId("fdlknfa32bse3yrbew23bf")}},
		test{callIdInput("Call-ID  : fdlknfa32bse3yrbew23bf"), &callIdResult{pass, base.CallId("fdlknfa32bse3yrbew23bf")}},
		test{callIdInput("Call-ID\t: fdlknfa32bse3yrbew23bf"), &callIdResult{pass, base.CallId("fdlknfa32bse3yrbew23bf")}},
		test{callIdInput("Call-ID: banana"), &callIdResult{pass, base.CallId("banana")}},
		test{callIdInput("calL-id: banana"), &callIdResult{pass, base.CallId("banana")}},
		test{callIdInput("calL-id: 1banana"), &callIdResult{pass, base.CallId("1banana")}},
		test{callIdInput("Call-ID:"), &callIdResult{fail, base.CallId("")}},
		test{callIdInput("Call-ID: banana spaghetti"), &callIdResult{fail, base.CallId("")}},
		test{callIdInput("Call-ID: banana\tspaghetti"), &callIdResult{fail, base.CallId("")}},
		test{callIdInput("Call-ID: banana;spaghetti"), &callIdResult{fail, base.CallId("")}},
		test{callIdInput("Call-ID: banana;spaghetti=tasty"), &callIdResult{fail, base.CallId("")}},
	}, t)
}

func TestMaxForwards(t *testing.T) {
	doTests([]test{
		test{maxForwardsInput("Max-Forwards: 9"), &maxForwardsResult{pass, base.MaxForwards(9)}},
		test{maxForwardsInput("Max-Forwards: 70"), &maxForwardsResult{pass, base.MaxForwards(70)}},
		test{maxForwardsInput("Max-Forwards: 71"), &maxForwardsResult{pass, base.MaxForwards(71)}},
		test{maxForwardsInput("Max-Forwards: 0"), &maxForwardsResult{pass, base.MaxForwards(0)}},
		test{maxForwardsInput("Max-Forwards:      0"), &maxForwardsResult{pass, base.MaxForwards(0)}},
		test{maxForwardsInput("Max-Forwards:\t0"), &maxForwardsResult{pass, base.MaxForwards(0)}},
		test{maxForwardsInput("Max-Forwards: \t 0"), &maxForwardsResult{pass, base.MaxForwards(0)}},
		test{maxForwardsInput("Max-Forwards:\n  0"), &maxForwardsResult{pass, base.MaxForwards(0)}},
		test{maxForwardsInput("Max-Forwards: -1"), &maxForwardsResult{fail, base.MaxForwards(0)}},
		test{maxForwardsInput("Max-Forwards:"), &maxForwardsResult{fail, base.MaxForwards(0)}},
		test{maxForwardsInput("Max-Forwards: "), &maxForwardsResult{fail, base.MaxForwards(0)}},
		test{maxForwardsInput("Max-Forwards:\t"), &maxForwardsResult{fail, base.MaxForwards(0)}},
		test{maxForwardsInput("Max-Forwards:\n"), &maxForwardsResult{fail, base.MaxForwards(0)}},
		test{maxForwardsInput("Max-Forwards: \n"), &maxForwardsResult{fail, base.MaxForwards(0)}},
	}, t)
}

func TestContentLength(t *testing.T) {
	doTests([]test{
		test{contentLengthInput("Content-Length: 9"), &contentLengthResult{pass, base.ContentLength(9)}},
		test{contentLengthInput("Content-Length: 20"), &contentLengthResult{pass, base.ContentLength(20)}},
		test{contentLengthInput("Content-Length: 113"), &contentLengthResult{pass, base.ContentLength(113)}},
		test{contentLengthInput("l: 113"), &contentLengthResult{pass, base.ContentLength(113)}},
		test{contentLengthInput("Content-Length: 0"), &contentLengthResult{pass, base.ContentLength(0)}},
		test{contentLengthInput("Content-Length:      0"), &contentLengthResult{pass, base.ContentLength(0)}},
		test{contentLengthInput("Content-Length:\t0"), &contentLengthResult{pass, base.ContentLength(0)}},
		test{contentLengthInput("Content-Length: \t 0"), &contentLengthResult{pass, base.ContentLength(0)}},
		test{contentLengthInput("Content-Length:\n  0"), &contentLengthResult{pass, base.ContentLength(0)}},
		test{contentLengthInput("Content-Length: -1"), &contentLengthResult{fail, base.ContentLength(0)}},
		test{contentLengthInput("Content-Length:"), &contentLengthResult{fail, base.ContentLength(0)}},
		test{contentLengthInput("Content-Length: "), &contentLengthResult{fail, base.ContentLength(0)}},
		test{contentLengthInput("Content-Length:\t"), &contentLengthResult{fail, base.ContentLength(0)}},
		test{contentLengthInput("Content-Length:\n"), &contentLengthResult{fail, base.ContentLength(0)}},
		test{contentLengthInput("Content-Length: \n"), &contentLengthResult{fail, base.ContentLength(0)}},
	}, t)
}

func TestViaHeaders(t *testing.T) {
	// branch=z9hG4bKnashds8
	fooEqBar := base.NewParams().Add("foo", base.String{"bar"})
	fooEqSlashBar := base.NewParams().Add("foo", base.String{"//bar"})
	singleFoo := base.NewParams().Add("foo", base.NoString{})
	doTests([]test{
		test{viaInput("Via: SIP/2.0/UDP pc33.atlanta.com"), &viaResult{pass, &base.ViaHeader{&base.ViaHop{"SIP", "2.0", "UDP", "pc33.atlanta.com", nil, noParams}}}},
		test{viaInput("Via: bAzz/fooo/BAAR pc33.atlanta.com"), &viaResult{pass, &base.ViaHeader{&base.ViaHop{"bAzz", "fooo", "BAAR", "pc33.atlanta.com", nil, noParams}}}},
		test{viaInput("Via: SIP/2.0/UDP pc33.atlanta.com"), &viaResult{pass, &base.ViaHeader{&base.ViaHop{"SIP", "2.0", "UDP", "pc33.atlanta.com", nil, noParams}}}},
		test{viaInput("Via: SIP /\t2.0 / UDP pc33.atlanta.com"), &viaResult{pass, &base.ViaHeader{&base.ViaHop{"SIP", "2.0", "UDP", "pc33.atlanta.com", nil, noParams}}}},
		test{viaInput("Via: SIP /\n 2.0 / UDP pc33.atlanta.com"), &viaResult{pass, &base.ViaHeader{&base.ViaHop{"SIP", "2.0", "UDP", "pc33.atlanta.com", nil, noParams}}}},
		test{viaInput("Via:\tSIP/2.0/UDP pc33.atlanta.com"), &viaResult{pass, &base.ViaHeader{&base.ViaHop{"SIP", "2.0", "UDP", "pc33.atlanta.com", nil, noParams}}}},
		test{viaInput("Via:\n SIP/2.0/UDP pc33.atlanta.com"), &viaResult{pass, &base.ViaHeader{&base.ViaHop{"SIP", "2.0", "UDP", "pc33.atlanta.com", nil, noParams}}}},
		test{viaInput("Via: SIP/2.0/UDP box:5060"), &viaResult{pass, &base.ViaHeader{&base.ViaHop{"SIP", "2.0", "UDP", "box", &ui16_5060, noParams}}}},
		test{viaInput("Via: SIP/2.0/UDP box;foo=bar"), &viaResult{pass, &base.ViaHeader{&base.ViaHop{"SIP", "2.0", "UDP", "box", nil, fooEqBar}}}},
		test{viaInput("Via: SIP/2.0/UDP box:5060;foo=bar"), &viaResult{pass, &base.ViaHeader{&base.ViaHop{"SIP", "2.0", "UDP", "box", &ui16_5060, fooEqBar}}}},
		test{viaInput("Via: SIP/2.0/UDP box:5060;foo"), &viaResult{pass, &base.ViaHeader{&base.ViaHop{"SIP", "2.0", "UDP", "box", &ui16_5060, singleFoo}}}},
		test{viaInput("Via: SIP/2.0/UDP box:5060;foo=//bar"), &viaResult{pass, &base.ViaHeader{&base.ViaHop{"SIP", "2.0", "UDP", "box", &ui16_5060, fooEqSlashBar}}}},
		test{viaInput("Via: /2.0/UDP box:5060;foo=bar"), &viaResult{fail, &base.ViaHeader{}}},
		test{viaInput("Via: SIP//UDP box:5060;foo=bar"), &viaResult{fail, &base.ViaHeader{}}},
		test{viaInput("Via: SIP/2.0/ box:5060;foo=bar"), &viaResult{fail, &base.ViaHeader{}}},
		test{viaInput("Via:  /2.0/UDP box:5060;foo=bar"), &viaResult{fail, &base.ViaHeader{}}},
		test{viaInput("Via: SIP/ /UDP box:5060;foo=bar"), &viaResult{fail, &base.ViaHeader{}}},
		test{viaInput("Via: SIP/2.0/  box:5060;foo=bar"), &viaResult{fail, &base.ViaHeader{}}},
		test{viaInput("Via: \t/2.0/UDP box:5060;foo=bar"), &viaResult{fail, &base.ViaHeader{}}},
		test{viaInput("Via: SIP/\t/UDP box:5060;foo=bar"), &viaResult{fail, &base.ViaHeader{}}},
		test{viaInput("Via: SIP/2.0/\t  box:5060;foo=bar"), &viaResult{fail, &base.ViaHeader{}}},
		test{viaInput("Via:"), &viaResult{fail, &base.ViaHeader{}}},
		test{viaInput("Via: "), &viaResult{fail, &base.ViaHeader{}}},
		test{viaInput("Via:\t"), &viaResult{fail, &base.ViaHeader{}}},
		test{viaInput("Via: box:5060"), &viaResult{fail, &base.ViaHeader{}}},
		test{viaInput("Via: box:5060;foo=bar"), &viaResult{fail, &base.ViaHeader{}}},
	}, t)
}

// Basic test of unstreamed parsing, using empty INVITE.
func TestUnstreamedParse1(t *testing.T) {
	test := ParserTest{false, []parserTestStep{
		// Steps each have: Input, result, sent error, returned error
		parserTestStep{"INVITE sip:bob@biloxi.com SIP/2.0\r\n\r\n",
			base.NewRequest(base.INVITE,
				&base.SipUri{false, base.String{"bob"}, base.NoString{}, "biloxi.com", nil, noParams, noParams},
				"SIP/2.0",
				make([]base.SipHeader, 0),
				""),
			nil,
			nil},
	}}

	test.Test(t)
}

// Test unstreamed parsing with a header and body.
func TestUnstreamedParse2(t *testing.T) {
	test := ParserTest{false, []parserTestStep{
		// Steps each have: Input, result, sent error, returned error
		parserTestStep{"INVITE sip:bob@biloxi.com SIP/2.0\r\n" +
			"CSeq: 13 INVITE\r\n" +
			"\r\n" +
			"I am a banana",
			base.NewRequest(base.INVITE,
				&base.SipUri{false, base.String{"bob"}, base.NoString{}, "biloxi.com", nil, noParams, noParams},
				"SIP/2.0",
				[]base.SipHeader{&base.CSeq{13, base.INVITE}},
				"I am a banana"),
			nil,
			nil},
	}}

	test.Test(t)
}

// Test unstreamed parsing of a base.Request object (rather than a base.Response).
func TestUnstreamedParse3(t *testing.T) {
	test := ParserTest{false, []parserTestStep{
		// Steps each have: Input, result, sent error, returned error
		parserTestStep{"SIP/2.0 200 OK\r\n" +
			"CSeq: 2 INVITE\r\n" +
			"\r\n" +
			"Everything is awesome.",
			base.NewResponse("SIP/2.0",
				200,
				"OK",
				[]base.SipHeader{&base.CSeq{2, base.INVITE}},
				"Everything is awesome."),
			nil,
			nil},
	}}

	test.Test(t)
}

// Test unstreamed parsing with more than one header.
func TestUnstreamedParse4(t *testing.T) {
	callId := base.CallId("cheesecake1729")
	maxForwards := base.MaxForwards(65)
	test := ParserTest{false, []parserTestStep{
		// Steps each have: Input, result, sent error, returned error
		parserTestStep{"SIP/2.0 200 OK\r\n" +
			"CSeq: 2 INVITE\r\n" +
			"Call-ID: cheesecake1729\r\n" +
			"Max-Forwards: 65\r\n" +
			"\r\n" +
			"Everything is awesome.",
			base.NewResponse("SIP/2.0",
				200,
				"OK",
				[]base.SipHeader{
					&base.CSeq{2, base.INVITE},
					&callId,
					&maxForwards,
				},
				"Everything is awesome."),
			nil,
			nil},
	}}

	test.Test(t)
}

// Test unstreamed parsing with whitespace and line breaks.
func TestUnstreamedParse5(t *testing.T) {
	callId := base.CallId("cheesecake1729")
	maxForwards := base.MaxForwards(63)
	test := ParserTest{false, []parserTestStep{
		// Steps each have: Input, result, sent error, returned error
		parserTestStep{"SIP/2.0 200 OK\r\n" +
			"CSeq:   2     \r\n" +
			"    INVITE\r\n" +
			"Call-ID:\tcheesecake1729\r\n" +
			"Max-Forwards:\t\r\n" +
			"\t63\r\n" +
			"\r\n" +
			"Everything is awesome.",
			base.NewResponse("SIP/2.0",
				200,
				"OK",
				[]base.SipHeader{
					&base.CSeq{2, base.INVITE},
					&callId,
					&maxForwards},
				"Everything is awesome."),
			nil,
			nil},
	}}

	test.Test(t)
}

// Test error responses, and responses of minimal length.
func TestUnstreamedParse6(t *testing.T) {
	test := ParserTest{false, []parserTestStep{
		parserTestStep{"SIP/2.0 403 Forbidden\r\n\r\n",
			base.NewResponse("SIP/2.0",
				403,
				"Forbidden",
				[]base.SipHeader{},
				""),
			nil,
			nil},
	}}

	test.Test(t)
}

// Test requests of minimal length.
func TestUnstreamedParse7(t *testing.T) {
	test := ParserTest{false, []parserTestStep{
		parserTestStep{"ACK sip:foo@bar.com SIP/2.0\r\n\r\n",
			base.NewRequest(base.ACK,
				&base.SipUri{false, base.String{"foo"}, base.NoString{}, "bar.com", nil, noParams, noParams},
				"SIP/2.0",
				[]base.SipHeader{},
				""),
			nil,
			nil},
	}}

	test.Test(t)
}

// TODO: Error cases for unstreamed parse.
// TODO: Multiple writes on unstreamed parse.

// Basic streamed parsing, using empty INVITE.
func TestStreamedParse1(t *testing.T) {
	contentLength := base.ContentLength(0)
	test := ParserTest{true, []parserTestStep{
		// Steps each have: Input, result, sent error, returned error
		parserTestStep{"INVITE sip:bob@biloxi.com SIP/2.0\r\n" +
			"Content-Length: 0\r\n\r\n",
			base.NewRequest(base.INVITE,
				&base.SipUri{false, base.String{"bob"}, base.NoString{}, "biloxi.com", nil, noParams, noParams},
				"SIP/2.0",
				[]base.SipHeader{&contentLength},
				""),
			nil,
			nil},
	}}

	test.Test(t)
}

// Test writing a single message in two stages (breaking after the start line).
func TestStreamedParse2(t *testing.T) {
	contentLength := base.ContentLength(0)
	test := ParserTest{true, []parserTestStep{
		// Steps each have: Input, result, sent error, returned error
		parserTestStep{"INVITE sip:bob@biloxi.com SIP/2.0\r\n", nil, nil, nil},
		parserTestStep{"Content-Length: 0\r\n\r\n",
			base.NewRequest(base.INVITE,
				&base.SipUri{false, base.String{"bob"}, base.NoString{}, "biloxi.com", nil, noParams, noParams},
				"SIP/2.0",
				[]base.SipHeader{&contentLength},
				""),
			nil,
			nil},
	}}

	test.Test(t)
}

// Test writing two successive messages, both with bodies.
func TestStreamedParse3(t *testing.T) {
	contentLength23 := base.ContentLength(23)
	contentLength33 := base.ContentLength(33)
	test := ParserTest{true, []parserTestStep{
		// Steps each have: Input, result, sent error, returned error
		parserTestStep{"INVITE sip:bob@biloxi.com SIP/2.0\r\n", nil, nil, nil},
		parserTestStep{"Content-Length: 23\r\n\r\n" +
			"Hello!\r\nThis is a test.",
			base.NewRequest(base.INVITE,
				&base.SipUri{false, base.String{"bob"}, base.NoString{}, "biloxi.com", nil, noParams, noParams},
				"SIP/2.0",
				[]base.SipHeader{&contentLength23},
				"Hello!\r\nThis is a test."),
			nil,
			nil},
		parserTestStep{"ACK sip:bob@biloxi.com SIP/2.0\r\n" +
			"Content-Length: 33\r\n" +
			"Contact: sip:alice@biloxi.com\r\n\r\n" +
			"This is an ack! : \n ! \r\n contact:",
			base.NewRequest(base.ACK,
				&base.SipUri{false, base.String{"bob"}, base.NoString{}, "biloxi.com", nil, noParams, noParams},
				"SIP/2.0",
				[]base.SipHeader{
					&contentLength33,
					&base.ContactHeader{nil, &base.SipUri{false, base.String{"alice"}, base.NoString{}, "biloxi.com", nil, noParams, noParams}, noParams},
				},
				"This is an ack! : \n ! \r\n contact:"),
			nil,
			nil},
	}}

	test.Test(t)
}

type paramInput struct {
	paramString      string
	start            uint8
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
	output, consumed, err := parseParams(data.paramString, data.start, data.sep, data.end, data.quoteValues, data.permitSingletons)
	return &paramResult{err, output, consumed}
}

type paramResult struct {
	err      error
	params   base.Params
	consumed int
}

func (expected *paramResult) equals(other result) (equal bool, reason string) {
	actual := *(other.(*paramResult))
	if expected.err == nil && actual.err != nil {
		return false, fmt.Sprintf("unexpected error: %s", actual.err.Error())
	} else if expected.err != nil && actual.err == nil {
		return false, fmt.Sprintf("unexpected success: got \"%s\"", actual.params.ToString('-'))
	} else if actual.err == nil && !expected.params.Equals(actual.params) {
		return false, fmt.Sprintf("unexpected result: expected \"%s\", got \"%s\"",
			expected.params.ToString('-'), actual.params.ToString('-'))
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
	host, port, err := parseHostPort(string(data))
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

type headerBlockInput []string

func (data headerBlockInput) String() string {
	return "['" + strings.Join([]string(data), "', '") + "']"
}

func (data headerBlockInput) evaluate() result {
	contents, linesConsumed := getNextHeaderLine([]string(data))
	return &headerBlockResult{contents, linesConsumed}
}

type headerBlockResult struct {
	contents      string
	linesConsumed int
}

func (expected *headerBlockResult) equals(other result) (equal bool, reason string) {
	actual := *(other.(*headerBlockResult))
	if expected.contents != actual.contents {
		return false, fmt.Sprintf("unexpected block contents: got \"%s\"; expected \"%s\"",
			actual.contents, expected.contents)
	} else if expected.linesConsumed != actual.linesConsumed {
		return false, fmt.Sprintf("unexpected number of lines used: %d (expected %d)",
			actual.linesConsumed, expected.linesConsumed)
	}

	return true, ""
}

func parseHeader(rawHeader string) (headers []base.SipHeader, err error) {
	messages := make(chan base.SipMessage, 0)
	errors := make(chan error, 0)
	p := NewParser(messages, errors, false)
	defer func() {
		log.Debug("Stopping %p", p)
		p.Stop()
	}()

	headers, err = (p.(*parser)).parseHeader(rawHeader)

	return
}

type toHeaderInput string

func (data toHeaderInput) String() string {
	return string(data)
}

func (data toHeaderInput) evaluate() result {
	headers, err := parseHeader(string(data))
	if len(headers) == 1 {
		return &toHeaderResult{err, headers[0].(*base.ToHeader)}
	} else if len(headers) == 0 {
		return &toHeaderResult{err, &base.ToHeader{}}
	} else {
		panic(fmt.Sprintf("Multiple headers returned by To test: %s", string(data)))
	}
}

type toHeaderResult struct {
	err    error
	header *base.ToHeader
}

func (expected *toHeaderResult) equals(other result) (equal bool, reason string) {
	actual := *(other.(*toHeaderResult))

	if expected.err == nil && actual.err != nil {
		return false, fmt.Sprintf("unexpected error: %s", actual.err.Error())
	} else if expected.err != nil && actual.err == nil {
		return false, fmt.Sprintf("unexpected success: got:\n%s\n\n", actual.header.String())
	} else if expected.err != nil {
		// Expected error. Return true immediately with no further checks.
		return true, ""
	}

	if expected.header.DisplayName != actual.header.DisplayName {
		return false, fmt.Sprintf("unexpected display name: expected \"%s\"; got \"%s\"",
			strMaybeStr(expected.header.DisplayName),
			strMaybeStr(actual.header.DisplayName))
	}

	switch expected.header.Address.(type) {
	case *base.SipUri:
		uri := *(expected.header.Address.(*base.SipUri))
		urisEqual := uri.Equals(actual.header.Address)
		msg := ""
		if !urisEqual {
			msg = fmt.Sprintf("unexpected result: expected %s, got %s",
				expected.header.Address.String(), actual.header.Address.String())
		}
		if !urisEqual {
			return false, msg
		}
	default:
		// If you're hitting this block, then you need to do the following:
		// - implement a package-private 'equals' method for the URI schema being tested.
		// - add a case block above for that schema, using the 'equals' method in the same was as the existing base.SipUri block above.
		return false, fmt.Sprintf("no support for testing Uri schema in Uri \"%s\" - fix me!", expected.header.Address)
	}

	if !expected.header.Params.Equals(actual.header.Params) {
		return false, fmt.Sprintf("unexpected parameters \"%s\" (expected \"%s\")",
			actual.header.Params.ToString('-'),
			expected.header.Params.ToString('-'))
	}

	return true, ""
}

type fromHeaderInput string

func (data fromHeaderInput) String() string {
	return string(data)
}

func (data fromHeaderInput) evaluate() result {
	headers, err := parseHeader(string(data))
	if len(headers) == 1 {
		return &fromHeaderResult{err, headers[0].(*base.FromHeader)}
	} else if len(headers) == 0 {
		return &fromHeaderResult{err, &base.FromHeader{}}
	} else {
		panic(fmt.Sprintf("Multiple headers returned by From test: %s", string(data)))
	}
}

type fromHeaderResult struct {
	err    error
	header *base.FromHeader
}

func (expected *fromHeaderResult) equals(other result) (equal bool, reason string) {
	actual := *(other.(*fromHeaderResult))

	if expected.err == nil && actual.err != nil {
		return false, fmt.Sprintf("unexpected error: %s", actual.err.Error())
	} else if expected.err != nil && actual.err == nil {
		return false, fmt.Sprintf("unexpected success: got:\n%s\n\n", actual.header.String())
	} else if expected.err != nil {
		// Expected error. Return true immediately with no further checks.
		return true, ""
	}

	if expected.header.DisplayName != actual.header.DisplayName {
		return false, fmt.Sprintf("unexpected display name: expected \"%s\"; got \"%s\"",
			strMaybeStr(expected.header.DisplayName),
			strMaybeStr(actual.header.DisplayName))
	}

	switch expected.header.Address.(type) {
	case *base.SipUri:
		uri := *(expected.header.Address.(*base.SipUri))
		urisEqual := uri.Equals(actual.header.Address)
		msg := ""
		if !urisEqual {
			msg = fmt.Sprintf("unexpected result: expected %s, got %s",
				expected.header.Address.String(), actual.header.Address.String())
		}
		if !urisEqual {
			return false, msg
		}
	default:
		// If you're hitting this block, then you need to do the following:
		// - implement a package-private 'equals' method for the URI schema being tested.
		// - add a case block above for that schema, using the 'equals' method in the same was as the existing base.SipUri block above.
		return false, fmt.Sprintf("no support for testing Uri schema in Uri \"%s\" - fix me!", expected.header.Address)
	}

	if !expected.header.Params.Equals(actual.header.Params) {
		return false, fmt.Sprintf("unexpected parameters \"%s\" (expected \"%s\")",
			actual.header.Params.ToString('-'),
			expected.header.Params.ToString('-'))
	}

	return true, ""
}

type contactHeaderInput string

func (data contactHeaderInput) String() string {
	return string(data)
}

func (data contactHeaderInput) evaluate() result {
	headers, err := parseHeader(string(data))
	contactHeaders := make([]*base.ContactHeader, len(headers))
	if len(headers) > 0 {
		for idx, header := range headers {
			contactHeaders[idx] = header.(*base.ContactHeader)
		}
		return &contactHeaderResult{err, contactHeaders}
	} else {
		return &contactHeaderResult{err, contactHeaders}
	}
}

type contactHeaderResult struct {
	err     error
	headers []*base.ContactHeader
}

func (expected *contactHeaderResult) equals(other result) (equal bool, reason string) {
	actual := *(other.(*contactHeaderResult))

	if expected.err == nil && actual.err != nil {
		return false, fmt.Sprintf("unexpected error: %s", actual.err.Error())
	} else if expected.err != nil && actual.err != nil {
		// Expected error. Return true immediately with no further checks.
		return true, ""
	}

	var buffer bytes.Buffer
	for _, header := range actual.headers {
		buffer.WriteString(fmt.Sprintf("\n\t%s", header))
	}
	buffer.WriteString("\n\n")
	actualStr := buffer.String()

	if expected.err != nil && actual.err == nil {
		return false, fmt.Sprintf("unexpected success: got: %s", actualStr)
	}

	if len(expected.headers) != len(actual.headers) {
		return false, fmt.Sprintf("expected %d headers; got %d. Last expected header: %s. Last actual header: %s",
			len(expected.headers), len(actual.headers),
			expected.headers[len(expected.headers)-1].String(), actual.headers[len(actual.headers)-1].String())
	}

	for idx := range expected.headers {
		if expected.headers[idx].DisplayName != actual.headers[idx].DisplayName {
			return false, fmt.Sprintf("unexpected display name: expected \"%s\"; got \"%s\"",
				strMaybeStr(expected.headers[idx].DisplayName),
				strMaybeStr(actual.headers[idx].DisplayName))
		}

		UrisEqual := expected.headers[idx].Address.Equals(actual.headers[idx].Address)
		if !UrisEqual {
			return false, fmt.Sprintf("expected Uri %#v; got Uri %#v", expected.headers[idx].Address, actual.headers[idx].Address)
		}

		if !expected.headers[idx].Params.Equals(actual.headers[idx].Params) {
			return false, fmt.Sprintf("unexpected parameters \"%s\" (expected \"%s\")",
				actual.headers[idx].Params.ToString('-'),
				expected.headers[idx].Params.ToString('-'))
		}
	}

	return true, ""
}

type splitByWSInput string

func (data splitByWSInput) String() string {
	return string(data)
}

func (data splitByWSInput) evaluate() result {
	return splitByWSResult(splitByWhitespace(string(data)))
}

type splitByWSResult []string

func (expected splitByWSResult) equals(other result) (equal bool, reason string) {
	actual := other.(splitByWSResult)
	if len(expected) != len(actual) {
		return false, fmt.Sprintf("unexpected result length in splitByWS test: expected %d %v, got %d %v.", len(expected), expected, len(actual), actual)
	}

	for idx, e := range expected {
		if e != actual[idx] {
			return false, fmt.Sprintf("unexpected result at index %d in splitByWS test: expected '%s'; got '%s'", idx, e, actual[idx])
		}
	}

	return true, ""
}

type cSeqInput string

func (data cSeqInput) String() string {
	return string(data)
}

func (data cSeqInput) evaluate() result {
	headers, err := parseHeader(string(data))
	if len(headers) == 1 {
		return &cSeqResult{err, headers[0].(*base.CSeq)}
	} else if len(headers) == 0 {
		return &cSeqResult{err, &base.CSeq{}}
	} else {
		panic(fmt.Sprintf("Multiple headers returned by base.CSeq test: %s", string(data)))
	}
}

type cSeqResult struct {
	err    error
	header *base.CSeq
}

func (expected *cSeqResult) equals(other result) (equal bool, reason string) {
	actual := *(other.(*cSeqResult))
	if expected.err == nil && actual.err != nil {
		return false, fmt.Sprintf("unexpected error: %s", actual.err.Error())
	} else if expected.err != nil && actual.err == nil {
		return false, fmt.Sprintf("unexpected success: got \"%s\"", actual.header.String())
	} else if actual.err == nil && expected.header.SeqNo != actual.header.SeqNo {
		return false, fmt.Sprintf("unexpected sequence number: expected \"%d\", got \"%d\"",
			expected.header.SeqNo, actual.header.SeqNo)
	} else if actual.err == nil && expected.header.MethodName != actual.header.MethodName {
		return false, fmt.Sprintf("unexpected method name: expected %s, got %s", expected.header.MethodName, actual.header.MethodName)
	}

	return true, ""
}

type callIdInput string

func (data callIdInput) String() string {
	return string(data)
}

func (data callIdInput) evaluate() result {
	headers, err := parseHeader(string(data))
	if len(headers) == 1 {
		return &callIdResult{err, *(headers[0].(*base.CallId))}
	} else if len(headers) == 0 {
		return &callIdResult{err, base.CallId("")}
	} else {
		panic(fmt.Sprintf("Multiple headers returned by base.CallId test: %s", string(data)))
	}
}

type callIdResult struct {
	err    error
	header base.CallId
}

func (expected callIdResult) equals(other result) (equal bool, reason string) {
	actual := *(other.(*callIdResult))
	if expected.err == nil && actual.err != nil {
		return false, fmt.Sprintf("unexpected error: %s", actual.err.Error())
	} else if expected.err != nil && actual.err == nil {
		return false, fmt.Sprintf("unexpected success: got \"%s\"", actual.header.String())
	} else if actual.err == nil && expected.header.String() != actual.header.String() {
		return false, fmt.Sprintf("unexpected call ID string: expected \"%s\", got \"%s\"",
			expected.header, actual.header)
	}
	return true, ""
}

type maxForwardsInput string

func (data maxForwardsInput) String() string {
	return string(data)
}

func (data maxForwardsInput) evaluate() result {
	headers, err := parseHeader(string(data))
	if len(headers) == 1 {
		return &maxForwardsResult{err, *(headers[0].(*base.MaxForwards))}
	} else if len(headers) == 0 {
		return &maxForwardsResult{err, base.MaxForwards(0)}
	} else {
		panic(fmt.Sprintf("Multiple headers returned by Max-Forwards test: %s", string(data)))
	}
}

type maxForwardsResult struct {
	err    error
	header base.MaxForwards
}

func (expected *maxForwardsResult) equals(other result) (equal bool, reason string) {
	actual := *(other.(*maxForwardsResult))
	if expected.err == nil && actual.err != nil {
		return false, fmt.Sprintf("unexpected error: %s", actual.err.Error())
	} else if expected.err != nil && actual.err == nil {
		return false, fmt.Sprintf("unexpected success: got \"%s\"", actual.header.String())
	} else if actual.err == nil && expected.header != actual.header {
		return false, fmt.Sprintf("unexpected max forwards value: expected \"%d\", got \"%d\"",
			expected.header, actual.header)
	}
	return true, ""
}

type contentLengthInput string

func (data contentLengthInput) String() string {
	return string(data)
}

func (data contentLengthInput) evaluate() result {
	headers, err := parseHeader(string(data))
	if len(headers) == 1 {
		return &contentLengthResult{err, *(headers[0].(*base.ContentLength))}
	} else if len(headers) == 0 {
		return &contentLengthResult{err, base.ContentLength(0)}
	} else {
		panic(fmt.Sprintf("Multiple headers returned by Content-Length test: %s", string(data)))
	}
}

type contentLengthResult struct {
	err    error
	header base.ContentLength
}

func (expected *contentLengthResult) equals(other result) (equal bool, reason string) {
	actual := *(other.(*contentLengthResult))
	if expected.err == nil && actual.err != nil {
		return false, fmt.Sprintf("unexpected error: %s", actual.err.Error())
	} else if expected.err != nil && actual.err == nil {
		return false, fmt.Sprintf("unexpected success: got \"%s\"", actual.header.String())
	} else if actual.err == nil && expected.header != actual.header {
		return false, fmt.Sprintf("unexpected max forwards value: expected \"%d\", got \"%d\"",
			expected.header, actual.header)
	}
	return true, ""
}

type viaInput string

func (data viaInput) String() string {
	return string(data)
}

func (data viaInput) evaluate() result {
	headers, err := parseHeader(string(data))
	if len(headers) == 0 {
		return &viaResult{err, &base.ViaHeader{}}
	} else if len(headers) == 1 {
		return &viaResult{err, headers[0].(*base.ViaHeader)}
	} else {
		panic("got more than one via header on test " + data)
	}
}

type viaResult struct {
	err    error
	header *base.ViaHeader
}

func (expected *viaResult) equals(other result) (equal bool, reason string) {
	actual := *(other.(*viaResult))
	if expected.err == nil && actual.err != nil {
		return false, fmt.Sprintf("unexpected error: %s", actual.err.Error())
	} else if expected.err != nil && actual.err == nil {
		return false, "unexpected success - got: " + actual.header.String()
	} else if expected.err != nil {
		// Got an error, and were expecting one - return with no further checks.
	} else if len(*expected.header) != len(*actual.header) {
		return false,
			fmt.Sprintf("unexpected number of entries: expected %d; got %d.\n"+
				"expected the following entries: %s\n"+
				"got the following entries: %s",
				len(*expected.header), len(*actual.header),
				expected.header.String(), actual.header.String())
	}

	for idx, expectedHop := range *expected.header {
		actualHop := (*actual.header)[idx]
		if expectedHop.ProtocolName != actualHop.ProtocolName {
			return false, fmt.Sprintf("unexpected protocol name '%s' in via entry %d - expected '%s'",
				actualHop.ProtocolName, idx, expectedHop.ProtocolName)
		} else if expectedHop.ProtocolVersion != actualHop.ProtocolVersion {
			return false, fmt.Sprintf("unexpected protocol version '%s' in via entry %d - expected '%s'",
				actualHop.ProtocolVersion, idx, expectedHop.ProtocolVersion)
		} else if expectedHop.Transport != actualHop.Transport {
			return false, fmt.Sprintf("unexpected transport '%s' in via entry %d - expected '%s'",
				actualHop.Transport, idx, expectedHop.Transport)
		} else if expectedHop.Host != actualHop.Host {
			return false, fmt.Sprintf("unexpected host '%s' in via entry %d - expected '%s'",
				actualHop.Host, idx, expectedHop.Host)
		} else if !utils.Uint16PtrEq(expectedHop.Port, actualHop.Port) {
			return false, fmt.Sprintf("unexpected port '%d' in via entry %d - expected '%d'",
				uint16PtrStr(actualHop.Port), idx, uint16PtrStr(expectedHop.Port))
		} else if !expectedHop.Params.Equals(actualHop.Params) {
			return false, fmt.Sprintf("unexpected params '%s' in via entry %d - expected '%s'",
				actualHop.Params.ToString('-'),
				idx,
				expectedHop.Params.ToString('-'))
		}
	}

	return true, ""
}

type ParserTest struct {
	streamed bool
	steps    []parserTestStep
}

func (test *ParserTest) Test(t *testing.T) {
	testsRun++
	output := make(chan base.SipMessage)
	errs := make(chan error)

	p := NewParser(output, errs, test.streamed)
	defer p.Stop()

	for stepIdx, step := range test.steps {
		success, reason := step.Test(p, output, errs)
		if !success {
			t.Errorf("failure in test step %d of input:\n%s\n\nfailure was: %s", stepIdx, test.String(), reason)
			return
		}
	}

	testsPassed++
	return
}

func (t *ParserTest) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("[")
	for _, step := range t.steps {
		buffer.WriteString(step.input)
		buffer.WriteString(",")
	}
	buffer.WriteString("]")
	return buffer.String()
}

type parserTestStep struct {
	input string

	// Slightly kludgy - two of these must be nil at any time.
	result        base.SipMessage
	sentError     error
	returnedError error
}

func (step *parserTestStep) Test(parser Parser, msgChan chan base.SipMessage, errChan chan error) (success bool, reason string) {
	_, err := parser.Write([]byte(step.input))
	if err != step.returnedError {
		success = false
		reason = fmt.Sprintf("expected returned error %s on write; got %s", errToStr(step.returnedError), errToStr(err))
		return
	} else if step.returnedError != nil {
		success = true
		return
	}

	// TODO - check returns here as they look a bit fishy.
	if err == nil {
		select {
		case msg := <-msgChan:
			if msg == nil && step.result != nil {
				success = false
				reason = fmt.Sprintf("nil message returned from parser; expected:\n%s", step.result.String())
			} else if msg != nil && step.result == nil {
				success = false
				reason = fmt.Sprintf("expected no message to be returned; got\n%s", msg.String())
			} else if msg.String() != step.result.String() {
				success = false
				reason = fmt.Sprintf("unexpected message returned by parser; expected:\n\n%s\n\nbut got:\n\n%s", step.result.String(), msg.String())
			} else {
				success = true
			}
		case err = <-errChan:
			if err == nil && step.sentError != nil {
				success = false
				reason = fmt.Sprintf("nil error output from parser; expected: %s", step.sentError.Error())
			} else if err != nil && step.sentError == nil {
				success = false
				reason = fmt.Sprintf("expected no error; parser output: %s", err.Error())
			} else {
				success = true
			}
		case <-time.After(time.Second * 1):
			if step.result != nil || step.sentError != nil {
				success = false
				reason = "timeout when processing input"
			} else {
				success = true
			}
		}
	}

	return
}

func TestZZZCountTests(t *testing.T) {
	fmt.Printf("\n *** %d tests run ***", testsRun)
	fmt.Printf("\n *** %d tests passed (%.2f%%) ***\n\n", testsPassed, (float32(testsPassed) * 100.0 / float32(testsRun)))
}

func strMaybeStr(s base.MaybeString) string {
	switch s := s.(type) {
	case base.NoString:
		return "<none>"
	case base.String:
		return s.String()
	default:
		return "nil"
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
