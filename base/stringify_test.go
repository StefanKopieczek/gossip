package base

// These tests confirm that our various structures stringify correctly.

import (
	"fmt"
	"testing"
)

// Generic test for testing anything with a String() method.
type stringTest struct {
	description string
	input       fmt.Stringer
	expected    string
}

func doTests(tests []stringTest, t *testing.T) {
	passed := 0
	for _, test := range tests {
		if test.input.String() != test.expected {
			t.Errorf("[FAIL] %v: Expected: \"%v\", Got: \"%v\"",
				test.description,
				test.expected,
				test.input.String(),
			)
		} else {
			passed++
		}
	}
	t.Logf("Passed %v/%v tests", passed, len(tests))
}

// Some global ports to use since port is still a pointer.
var port5060 uint16 = 5060
var port6060 uint16 = 6060

func TestSipUri(t *testing.T) {
	doTests([]stringTest{
		{"Basic SIP URI",
			&SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com"},
			"sip:alice@wonderland.com"},
		{"SIP URI with no user",
			&SipUri{User: NoString{}, Password: NoString{}, Host: "wonderland.com"},
			"sip:wonderland.com"},
		{"SIP URI with password",
			&SipUri{User: String{"alice"}, Password: String{"hunter2"}, Host: "wonderland.com"},
			"sip:alice:hunter2@wonderland.com"},
		{"SIP URI with explicit port 5060",
			&SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com", Port: &port5060},
			"sip:alice@wonderland.com:5060"},
		{"SIP URI with other port",
			&SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com", Port: &port6060},
			"sip:alice@wonderland.com:6060"},
		{"Basic SIPS URI",
			&SipUri{IsEncrypted: true, User: String{"alice"}, Password: NoString{}, Host: "wonderland.com"},
			"sips:alice@wonderland.com"},
		{"SIP URI with one parameter",
			&SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com",
				UriParams: NewParams().Add("food", String{"cake"})},
			"sip:alice@wonderland.com;food=cake"},
		{"SIP URI with one no-value parameter",
			&SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com",
				UriParams: NewParams().Add("something", NoString{})},
			"sip:alice@wonderland.com;something"},
		{"SIP URI with three parameters",
			&SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com",
				UriParams: NewParams().Add("food", String{"cake"}).Add("something", NoString{}).Add("drink", String{"tea"})},
			"sip:alice@wonderland.com;food=cake;something;drink=tea"},
		{"SIP URI with one header",
			&SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com",
				Headers: NewParams().Add("CakeLocation", String{"Tea Party"})},
			"sip:alice@wonderland.com?CakeLocation=\"Tea Party\""},
		{"SIP URI with three headers",
			&SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com",
				Headers: NewParams().Add("CakeLocation", String{"Tea Party"}).Add("Identity", String{"Mad Hatter"}).Add("OtherHeader", String{"Some value"})},
			"sip:alice@wonderland.com?CakeLocation=\"Tea Party\"&Identity=\"Mad Hatter\"&OtherHeader=\"Some value\""},
		{"SIP URI with parameter and header",
			&SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com",
				UriParams: NewParams().Add("food", String{"cake"}),
				Headers:   NewParams().Add("CakeLocation", String{"Tea Party"})},
			"sip:alice@wonderland.com;food=cake?CakeLocation=\"Tea Party\""},
		{"Wildcard URI", &WildcardUri{}, "*"},
	}, t)
}

func TestHeaders(t *testing.T) {
	doTests([]stringTest{
		{"Basic To Header",
			&ToHeader{DisplayName: NoString{},
				Address: &SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com"}},
			"To: <sip:alice@wonderland.com>"},
		{"To Header with display name",
			&ToHeader{DisplayName: String{"Alice Liddell"},
				Address: &SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com"}},
			"To: \"Alice Liddell\" <sip:alice@wonderland.com>"},
		{"To Header with parameters",
			&ToHeader{DisplayName: NoString{},
				Address: &SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com"},
				Params:  NewParams().Add("food", String{"cake"})},
			"To: <sip:alice@wonderland.com>;food=cake"},
		{"Basic From Header",
			&FromHeader{DisplayName: NoString{},
				Address: &SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com"}},
			"From: <sip:alice@wonderland.com>"},
		{"From Header with display name",
			&FromHeader{DisplayName: String{"Alice Liddell"},
				Address: &SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com"}},
			"From: \"Alice Liddell\" <sip:alice@wonderland.com>"},
		{"From Header with parameters",
			&FromHeader{DisplayName: NoString{},
				Address: &SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com"},
				Params:  NewParams().Add("food", String{"cake"})},
			"From: <sip:alice@wonderland.com>;food=cake"},
		{"Basic Contact Header",
			&ContactHeader{DisplayName: NoString{},
				Address: &SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com"}},
			"Contact: <sip:alice@wonderland.com>"},
		{"Contact Header with display name",
			&ContactHeader{DisplayName: String{"Alice Liddell"},
				Address: &SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com"}},
			"Contact: \"Alice Liddell\" <sip:alice@wonderland.com>"},
		{"Contact Header with parameters",
			&ContactHeader{DisplayName: NoString{},
				Address: &SipUri{User: String{"alice"}, Password: NoString{}, Host: "wonderland.com"},
				Params:  NewParams().Add("food", String{"cake"})},
			"Contact: <sip:alice@wonderland.com>;food=cake"},
		{"Contact Header with Wildcard URI",
			&ContactHeader{DisplayName: NoString{}, Address: &WildcardUri{}},
			"Contact: *"},
		{"Contact Header with display name and Wildcard URI",
			&ContactHeader{DisplayName: String{"Mad Hatter"}, Address: &WildcardUri{}},
			"Contact: \"Mad Hatter\" *"},
		{"Contact Header with Wildcard URI and parameters",
			&ContactHeader{DisplayName: NoString{}, Address: &WildcardUri{}, Params: NewParams().Add("food", String{"cake"})},
			"Contact: *;food=cake"},
	}, t)
}
