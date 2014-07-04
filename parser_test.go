package gossip

import "bytes"
import "fmt"
import "strings"
import "strconv"
import "testing"

var testsRun int


type input interface{
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
    args input
    expected result
}

func doTests(tests []test, t *testing.T) {
    for _, test := range(tests) {
        testsRun++
        output := test.args.evaluate()
        pass, reason := test.expected.equals(output)
        if !pass {
            t.Errorf("Failure on input \"%s\" : %s", test.args.String(), reason)
        }
    }
}

// Pass and fail placeholders
var fail error = fmt.Errorf("A bad thing happened.")
var pass error = nil

// Need to define immutable variables in order to pointer to them.
var bar string = "bar"
var barQuote string = "\"bar\""
var barQuote2 string = "\"bar"
var barQuote3 string = "bar\""
var barBaz string = "bar;baz"
// var baz string = "baz"
var bob string = "bob"
var boop string = "boop"
var b string = "b"
var empty string = ""
//var hunter2 string = "Hunter2"
//var port5060 string = uint16(5060)
//var port9 string = uint16(9)
//var uint16_5 uint16:= uint16(5)
//var uint16_5060 := uint16(5060)

func TestParams(t *testing.T) {
    doTests([]test {
        // TEST: parseParams
        test{&paramInput{";foo=bar",               ';', ';',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&bar},                          8}},
        test{&paramInput{";foo=",                  ';', ';',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&empty},                        5}},
        test{&paramInput{";foo",                   ';', ';',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":nil},                           4}},
        test{&paramInput{";foo=bar!hello",         ';', ';', '!', false, true},  &paramResult{pass, map[string]*string{"foo":&bar},                          8}},
        test{&paramInput{";foo!hello",             ';', ';', '!', false, true},  &paramResult{pass, map[string]*string{"foo":nil},                           4}},
        test{&paramInput{";foo=!hello",            ';', ';', '!', false, true},  &paramResult{pass, map[string]*string{"foo":&empty},                        5}},
        test{&paramInput{";foo=bar!h;l!o",         ';', ';', '!', false, true},  &paramResult{pass, map[string]*string{"foo":&bar},                          8}},
        test{&paramInput{";foo!h;l!o",             ';', ';', '!', false, true},  &paramResult{pass, map[string]*string{"foo":nil},                           4}},
        test{&paramInput{"foo!h;l!o",              ';', ';', '!', false, true},  &paramResult{fail, map[string]*string{},                                    0}},
        test{&paramInput{"foo;h;l!o",              ';', ';', '!', false, true},  &paramResult{fail, map[string]*string{},                                    0}},
        test{&paramInput{";foo=bar;baz=boop",      ';', ';',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&bar, "baz":&boop},             17}},
        test{&paramInput{";foo=bar;baz=boop!lol",  ';', ';', '!', false, true},  &paramResult{pass, map[string]*string{"foo":&bar, "baz":&boop},             17}},
        test{&paramInput{";foo=bar;baz",           ';', ';',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&bar, "baz":nil},               12}},
        test{&paramInput{";foo;baz=boop",          ';', ';',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":nil, "baz":&boop},              13}},
        test{&paramInput{";foo=bar;baz=boop;a=b",  ';', ';',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&bar, "baz":&boop, "a":&b},     21}},
        test{&paramInput{";foo;baz=boop;a=b",      ';', ';',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":nil, "baz":&boop, "a":&b},      17}},
        test{&paramInput{";foo=bar;baz;a=b",       ';', ';',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&bar, "baz":nil, "a":&b},       16}},
        test{&paramInput{";foo=bar;baz=boop;a",    ';', ';',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&bar, "baz":&boop, "a":nil},    19}},
        test{&paramInput{";foo=bar;baz=;a",        ';', ';',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&bar, "baz":&empty, "a":nil},   15}},
        test{&paramInput{";foo=;baz=bob;a",        ';', ';',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&empty, "baz":&bob, "a":nil},   15}},
        test{&paramInput{"foo=bar",                ';', ';',  0,  false, true},  &paramResult{fail, map[string]*string{},                                    0}},
        test{&paramInput{"$foo=bar",               '$', ',',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&bar},                          8}},
        test{&paramInput{"$foo",                   '$', ',',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":nil},                           4}},
        test{&paramInput{"$foo=bar!hello",         '$', ',', '!', false, true},  &paramResult{pass, map[string]*string{"foo":&bar},                          8}},
        test{&paramInput{"$foo#hello",             '$', ',', '#', false, true},  &paramResult{pass, map[string]*string{"foo":nil},                           4}},
        test{&paramInput{"$foo=bar!h;,!o",         '$', ',', '!', false, true},  &paramResult{pass, map[string]*string{"foo":&bar},                          8}},
        test{&paramInput{"$foo!h;l!,",             '$', ',', '!', false, true},  &paramResult{pass, map[string]*string{"foo":nil},                           4}},
        test{&paramInput{"foo!h;l!o",              '$', ',', '!', false, true},  &paramResult{fail, map[string]*string{},                                    0}},
        test{&paramInput{"foo,h,l!o",              '$', ',', '!', false, true},  &paramResult{fail, map[string]*string{},                                    0}},
        test{&paramInput{"$foo=bar,baz=boop",      '$', ',',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&bar, "baz":&boop},             17}},
        test{&paramInput{"$foo=bar;baz",           '$', ',',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&barBaz},                       12}},
        test{&paramInput{"$foo=bar,baz=boop!lol",  '$', ',', '!', false, true},  &paramResult{pass, map[string]*string{"foo":&bar, "baz":&boop},             17}},
        test{&paramInput{"$foo=bar,baz",           '$', ',',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&bar, "baz":nil},               12}},
        test{&paramInput{"$foo=,baz",              '$', ',',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&empty, "baz":nil},              9}},
        test{&paramInput{"$foo,baz=boop",          '$', ',',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":nil, "baz":&boop},              13}},
        test{&paramInput{"$foo=bar,baz=boop,a=b",  '$', ',',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&bar, "baz":&boop, "a":&b},     21}},
        test{&paramInput{"$foo,baz=boop,a=b",      '$', ',',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":nil, "baz":&boop, "a":&b},      17}},
        test{&paramInput{"$foo=bar,baz,a=b",       '$', ',',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&bar, "baz":nil, "a":&b},       16}},
        test{&paramInput{"$foo=bar,baz=boop,a",    '$', ',',  0,  false, true},  &paramResult{pass, map[string]*string{"foo":&bar, "baz":&boop, "a":nil},    19}},
        test{&paramInput{";foo",                   ';', ';',  0,  false, false}, &paramResult{fail, map[string]*string{},                                     0}},
        test{&paramInput{";foo=",                  ';', ';',  0,  false, false}, &paramResult{pass, map[string]*string{"foo":&empty},                         5}},
        test{&paramInput{";foo=bar;baz=boop",      ';', ';',  0,  false, false}, &paramResult{pass, map[string]*string{"foo":&bar, "baz":&boop},             17}},
        test{&paramInput{";foo=bar;baz",           ';', ';',  0,  false, false}, &paramResult{fail, map[string]*string{},                                     0}},
        test{&paramInput{";foo;bar=baz",           ';', ';',  0,  false, false}, &paramResult{fail, map[string]*string{},                                     0}},
        test{&paramInput{";foo=;baz=boop",         ';', ';',  0,  false, false}, &paramResult{pass, map[string]*string{"foo":&empty, "baz":&boop},           14}},
        test{&paramInput{";foo=bar;baz=",          ';', ';',  0,  false, false}, &paramResult{pass, map[string]*string{"foo":&bar, "baz":&empty},            13}},
        test{&paramInput{"$foo=bar,baz=,a=b",      '$', ',',  0,  false, true},  &paramResult{pass,
                                                                                              map[string]*string{"foo":&bar, "baz":&empty, "a":&b},          17}},
        test{&paramInput{"$foo=bar,baz,a=b",       '$', ',',  0,  false, false}, &paramResult{fail, map[string]*string{},                                    17}},
        test{&paramInput{";foo=\"bar\"",           ';', ';',  0,  false, true},  &paramResult{pass,  map[string]*string{"foo":&barQuote},                    10}},
        test{&paramInput{";foo=\"bar",             ';', ';',  0,  false, true},  &paramResult{pass,  map[string]*string{"foo":&barQuote2},                    9}},
        test{&paramInput{";foo=bar\"",             ';', ';',  0,  false, true},  &paramResult{pass,  map[string]*string{"foo":&barQuote3},                    9}},
        test{&paramInput{";\"foo\"=bar",           ';', ';',  0,  false, true},  &paramResult{pass,  map[string]*string{"\"foo\"":&bar},                     10}},
        test{&paramInput{";foo\"=bar",             ';', ';',  0,  false, true},  &paramResult{pass,  map[string]*string{"foo\"":&bar},                        9}},
        test{&paramInput{";\"foo=bar",             ';', ';',  0,  false, true},  &paramResult{pass,  map[string]*string{"\"foo":&bar},                        9}},
        test{&paramInput{";foo=\"bar\"",           ';', ';',  0,  true, true},   &paramResult{pass,  map[string]*string{"foo":&bar},                         10}},
        test{&paramInput{";foo=bar\"",             ';', ';',  0,  true, true},   &paramResult{fail, map[string]*string{},                                     0}},
        test{&paramInput{";foo=\"bar",             ';', ';',  0,  true, true},   &paramResult{fail, map[string]*string{},                                     0}},
        test{&paramInput{";\"foo\"=bar",           ';', ';',  0,  true, true},   &paramResult{fail, map[string]*string{},                                     0}},
        test{&paramInput{";\"foo=bar",             ';', ';',  0,  true, true},   &paramResult{fail, map[string]*string{},                                     0}},
        test{&paramInput{";foo\"=bar",             ';', ';',  0,  true, true},   &paramResult{fail, map[string]*string{},                                     0}},
        test{&paramInput{";foo=\"bar;baz\"",       ';', ';',  0,  true, true},   &paramResult{pass,  map[string]*string{"foo":&barBaz},                      14}},
        test{&paramInput{";foo=\"bar;baz\";a=b",   ';', ';',  0,  true, true},   &paramResult{pass,  map[string]*string{"foo":&barBaz, "a":&b},              18}},
        test{&paramInput{";foo=\"bar;baz\";a",     ';', ';',  0,  true, true},   &paramResult{pass,  map[string]*string{"foo":&barBaz, "a":nil},             16}},
        test{&paramInput{";foo=bar",               ';', ';',  0,  true, true},   &paramResult{pass,  map[string]*string{"foo":&bar},                          8}},
        test{&paramInput{";foo=",                  ';', ';',  0,  true, true},   &paramResult{pass,  map[string]*string{"foo":&empty},                        5}},
        test{&paramInput{";foo=\"\"",              ';', ';',  0,  true, true},   &paramResult{pass,  map[string]*string{"foo":&empty},                        7}},
    }, t)
}

func TestSipUris(t *testing.T) {
    // Need named variables for pointer fields in SipUri struct.
    b := "b"
    bar := "bar"
    baz := "baz"
    bob := "bob"
    emptyString := ""
    hunter2 := "Hunter2"
    ui16_5 := uint16(5)
    ui16_5060 := uint16(5060)

    doTests([]test {
        test{sipUriInput("sip:bob@example.com"),                          &sipUriResult{pass, SipUri{User:&bob, Host:"example.com"}}},
        test{sipUriInput("sip:bob@192.168.0.1"),                          &sipUriResult{pass, SipUri{User:&bob, Host:"192.168.0.1"}}},
        test{sipUriInput("sip:bob:Hunter2@example.com"),                  &sipUriResult{pass, SipUri{User:&bob, Password:&hunter2, Host:"example.com"}}},
        test{sipUriInput("sips:bob:Hunter2@example.com"),                 &sipUriResult{pass, SipUri{IsEncrypted:true, User:&bob, Password:&hunter2,
                                                                                                     Host:"example.com"}}},
        test{sipUriInput("sips:bob@example.com"),                         &sipUriResult{pass, SipUri{IsEncrypted:true, User:&bob, Host:"example.com"}}},
        test{sipUriInput("sip:example.com"),                              &sipUriResult{pass, SipUri{Host:"example.com"}}},
        test{sipUriInput("example.com"),                                  &sipUriResult{fail, SipUri{}}},
        test{sipUriInput("bob@example.com"),                              &sipUriResult{fail, SipUri{}}},
        test{sipUriInput("sip:bob@example.com:5060"),                     &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5060}}},
        test{sipUriInput("sip:bob@88.88.88.88:5060"),                     &sipUriResult{pass, SipUri{User:&bob, Host:"88.88.88.88", Port:&ui16_5060}}},
        test{sipUriInput("sip:bob:Hunter2@example.com:5060"),             &sipUriResult{pass, SipUri{User:&bob, Password:&hunter2,
                                                                                                     Host:"example.com", Port:&ui16_5060}}},
        test{sipUriInput("sip:bob@example.com:5"),                        &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5}}},
        test{sipUriInput("sip:bob@example.com;foo=bar"),                  &sipUriResult{pass, SipUri{User:&bob, Host:"example.com",
                                                                                                     UriParams:map[string]*string{"foo":&bar}}}},
        test{sipUriInput("sip:bob@example.com:5060;foo=bar"),             &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5060,
                                                                                                     UriParams:map[string]*string{"foo":&bar}}}},
        test{sipUriInput("sip:bob@example.com:5;foo"),                    &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                                                                     UriParams:map[string]*string{"foo":nil}}}},
        test{sipUriInput("sip:bob@example.com:5;foo;baz=bar"),            &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                                                                     UriParams:map[string]*string{"foo":nil, "baz":&bar}}}},
        test{sipUriInput("sip:bob@example.com:5;baz=bar;foo"),            &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                                                                     UriParams:map[string]*string{"foo":nil, "baz":&bar}}}},
        test{sipUriInput("sip:bob@example.com:5;foo;baz=bar;a=b"),        &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                                                                     UriParams:map[string]*string{"foo":nil, "baz":&bar, "a":&b}}}},
        test{sipUriInput("sip:bob@example.com:5;baz=bar;foo;a=b"),        &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                                                                     UriParams:map[string]*string{"foo":nil, "baz":&bar, "a":&b}}}},
        test{sipUriInput("sip:bob@example.com?foo=bar"),                  &sipUriResult{pass, SipUri{User:&bob, Host:"example.com",
                                                                                                     Headers:map[string]*string{"foo":&bar}}}},
        test{sipUriInput("sip:bob@example.com?foo="),                     &sipUriResult{pass, SipUri{User:&bob, Host:"example.com",
                                                                                                     Headers:map[string]*string{"foo":&emptyString}}}},
        test{sipUriInput("sip:bob@example.com:5060?foo=bar"),             &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5060,
                                                                                                     Headers:map[string]*string{"foo":&bar}}}},
        test{sipUriInput("sip:bob@example.com:5?foo=bar"),                &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                                                                     Headers:map[string]*string{"foo":&bar}}}},
        test{sipUriInput("sips:bob@example.com:5?baz=bar&foo=&a=b"),      &sipUriResult{pass, SipUri{IsEncrypted:true, User:&bob, Host:"example.com", Port:&ui16_5,
                                                                                                     Headers:map[string]*string{"baz":&bar, "a":&b,
                                                                                                                                "foo":&emptyString}}}},
        test{sipUriInput("sip:bob@example.com:5?baz=bar&foo&a=b"),        &sipUriResult{fail, SipUri{}}},
        test{sipUriInput("sip:bob@example.com:5?foo"),                    &sipUriResult{fail, SipUri{}}},
        test{sipUriInput("sip:bob@example.com:50?foo"),                   &sipUriResult{fail, SipUri{}}},
        test{sipUriInput("sip:bob@example.com:50?foo=bar&baz"),           &sipUriResult{fail, SipUri{}}},
        test{sipUriInput("sip:bob@example.com;foo?foo=bar"),              &sipUriResult{pass, SipUri{User:&bob, Host:"example.com",
                                                                                                     UriParams:map[string]*string{"foo":nil},
                                                                                                     Headers:map[string]*string{"foo":&bar}}}},
        test{sipUriInput("sip:bob@example.com:5060;foo?foo=bar"),         &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5060,
                                                                                                     UriParams:map[string]*string{"foo":nil},
                                                                                                     Headers:map[string]*string{"foo":&bar}}}},
        test{sipUriInput("sip:bob@example.com:5;foo?foo=bar"),            &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                                                                     UriParams:map[string]*string{"foo":nil},
                                                                                                     Headers:map[string]*string{"foo":&bar}}}},
        test{sipUriInput("sips:bob@example.com:5;foo?baz=bar&a=b&foo="),  &sipUriResult{pass, SipUri{IsEncrypted:true, User:&bob,
                                                                                                     Host:"example.com", Port:&ui16_5,
                                                                                                     UriParams:map[string]*string{"foo":nil},
                                                                                                     Headers:map[string]*string{"baz":&bar, "a":&b,
                                                                                                                                "foo":&emptyString}}}},
        test{sipUriInput("sip:bob@example.com:5;foo?baz=bar&foo&a=b"),    &sipUriResult{fail, SipUri{}}},
        test{sipUriInput("sip:bob@example.com:5;foo?foo"),                &sipUriResult{fail, SipUri{}}},
        test{sipUriInput("sip:bob@example.com:50;foo?foo"),               &sipUriResult{fail, SipUri{}}},
        test{sipUriInput("sip:bob@example.com:50;foo?foo=bar&baz"),       &sipUriResult{fail, SipUri{}}},
        test{sipUriInput("sip:bob@example.com;foo=baz?foo=bar"),          &sipUriResult{pass, SipUri{User:&bob, Host:"example.com",
                                                                                                     UriParams:map[string]*string{"foo":&baz},
                                                                                                     Headers:map[string]*string{"foo":&bar}}}},
        test{sipUriInput("sip:bob@example.com:5060;foo=baz?foo=bar"),     &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5060,
                                                                                                     UriParams:map[string]*string{"foo":&baz},
                                                                                                     Headers:map[string]*string{"foo":&bar}}}},
        test{sipUriInput("sip:bob@example.com:5;foo=baz?foo=bar"),        &sipUriResult{pass, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                                                                     UriParams:map[string]*string{"foo":&baz},
                                                                                                     Headers:map[string]*string{"foo":&bar}}}},
        test{sipUriInput("sips:bob@example.com:5;foo=baz?baz=bar&a=b"),   &sipUriResult{pass, SipUri{IsEncrypted:true, User:&bob, Host:"example.com", Port:&ui16_5,
                                                                                                     UriParams:map[string]*string{"foo":&baz},
                                                                                                     Headers:map[string]*string{"baz":&bar, "a":&b}}}},
        test{sipUriInput("sip:bob@example.com:5;foo=baz?baz=bar&foo&a=b"),&sipUriResult{fail, SipUri{}}},
        test{sipUriInput("sip:bob@example.com:5;foo=baz?foo"),            &sipUriResult{fail, SipUri{}}},
        test{sipUriInput("sip:bob@example.com:50;foo=baz?foo"),           &sipUriResult{fail, SipUri{}}},
        test{sipUriInput("sip:bob@example.com:50;foo=baz?foo=bar&baz"),   &sipUriResult{fail, SipUri{}}},
    }, t)
}

func TestHostPort(t *testing.T) () {
    port5060 := uint16(5060)
    port9 := uint16(9)

    doTests([]test {
        test{hostPortInput("example.com"),      &hostPortResult{pass, "example.com", nil}},
        test{hostPortInput("192.168.0.1"),      &hostPortResult{pass, "192.168.0.1", nil}},
        test{hostPortInput("abc123"),           &hostPortResult{pass, "abc123",      nil}},
        test{hostPortInput("example.com:5060"), &hostPortResult{pass, "example.com", &port5060}},
        test{hostPortInput("example.com:9"),    &hostPortResult{pass, "example.com", &port9}},
        test{hostPortInput("192.168.0.1:5060"), &hostPortResult{pass, "192.168.0.1", &port5060}},
        test{hostPortInput("192.168.0.1:9"),    &hostPortResult{pass, "192.168.0.1", &port9}},
        test{hostPortInput("abc123:5060"),      &hostPortResult{pass, "abc123",      &port5060}},
        test{hostPortInput("abc123:9"),         &hostPortResult{pass, "abc123",      &port9}},
        // TODO IPV6, c.f. IPv6reference in RFC 3261 s25
    }, t)
}

func TestHeaderBlocks(t *testing.T) {
    doTests([]test {
        test{headerBlockInput([]string{"All on one line."}),                             &headerBlockResult{"All on one line.", 1}},
        test{headerBlockInput([]string{"Line one", "Line two."}),                        &headerBlockResult{"Line one", 1}},
        test{headerBlockInput([]string{"Line one", " then an indent"}),                  &headerBlockResult{"Line one then an indent", 2}},
        test{headerBlockInput([]string{"Line one", " then an indent", "then line two"}), &headerBlockResult{"Line one then an indent", 2}},
        test{headerBlockInput([]string{"Line one", "Line two", " then an indent"}),      &headerBlockResult{"Line one", 1}},
        test{headerBlockInput([]string{"Line one", "\twith tab indent"}),                &headerBlockResult{"Line one with tab indent", 2}},
        test{headerBlockInput([]string{"Line one", "      with a big indent"}),          &headerBlockResult{"Line one with a big indent", 2}},
        test{headerBlockInput([]string{"Line one", " \twith space then tab"}),           &headerBlockResult{"Line one with space then tab", 2}},
        test{headerBlockInput([]string{"Line one", "\t    with tab then spaces"}),       &headerBlockResult{"Line one with tab then spaces", 2}},
        test{headerBlockInput([]string{""}),                                             &headerBlockResult{"", 1}},
        test{headerBlockInput([]string{" "}),                                            &headerBlockResult{" ", 1}},
        test{headerBlockInput([]string{}),                                               &headerBlockResult{"", 0}},
        test{headerBlockInput([]string{" foo"}),                                         &headerBlockResult{" foo", 1}},
    }, t)
}

func TestToHeaders(t *testing.T) {
    alice := "alice"
    aliceAddr := "sip:alice@wonderland.com"
    aliceAddrQuot := "<sip:alice@wonderland.com>"
    aliceAddrQuotSp := "<sip: alice@wonderland.com>"
    aliceTitle := "Alice"
    aliceLiddell := "Alice Liddell"
    bar := "bar"
    fooEqBar := map[string]*string{"foo" : &bar}
    fooSingleton := map[string]*string{"foo" : nil}
    noParams := map[string]*string{}
    doTests([]test {
        test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com>"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{toHeaderInput("To:\n  \"Alice Liddell\" \n\t<sip:alice@wonderland.com>"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{toHeaderInput("t: Alice <sip:alice@wonderland.com>"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceTitle,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{toHeaderInput("To: Alice sip:alice@wonderland.com"), &toHeaderResult{fail,
            &ToHeader{}}},

        test{toHeaderInput("To:"), &toHeaderResult{fail,
            &ToHeader{}}},

        test{toHeaderInput("To: "), &toHeaderResult{fail,
            &ToHeader{}}},

        test{toHeaderInput("To:\t"), &toHeaderResult{fail,
            &ToHeader{}}},

        test{toHeaderInput("To: foo"), &toHeaderResult{fail,
            &ToHeader{}}},

        test{toHeaderInput("To: foo bar"), &toHeaderResult{fail,
            &ToHeader{}}},

        test{toHeaderInput("To: \"Alice\" sip:alice@wonderland.com"), &toHeaderResult{fail,
            &ToHeader{}}},

        test{toHeaderInput("To: \"<Alice>\" sip:alice@wonderland.com"), &toHeaderResult{fail,
            &ToHeader{}}},

        test{toHeaderInput("To: \"sip:alice@wonderland.com\""), &toHeaderResult{fail,
            &ToHeader{}}},

        test{toHeaderInput("To: \"sip:alice@wonderland.com\"  <sip:alice@wonderland.com>"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceAddr,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{toHeaderInput("T: \"<sip:alice@wonderland.com>\"  <sip:alice@wonderland.com>"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceAddrQuot,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{toHeaderInput("To: \"<sip: alice@wonderland.com>\"  <sip:alice@wonderland.com>"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceAddrQuotSp,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com>;foo=bar"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:fooEqBar}}},

        test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com;foo=bar>"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, fooEqBar, noParams},
                      params:noParams}}},

        test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com?foo=bar>"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, fooEqBar},
                      params:noParams}}},

        test{toHeaderInput("to: \"Alice Liddell\" <sip:alice@wonderland.com>;foo"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:fooSingleton}}},

        test{toHeaderInput("TO: \"Alice Liddell\" <sip:alice@wonderland.com;foo>"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, fooSingleton, noParams},
                      params:noParams}}},

        test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com?foo>"), &toHeaderResult{fail,
            &ToHeader{}}},

        test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com;foo?foo=bar>;foo=bar"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, fooSingleton, fooEqBar},
                      params:fooEqBar}}},

        test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com;foo?foo=bar>;foo"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, fooSingleton, fooEqBar},
                      params:fooSingleton}}},

        test{toHeaderInput("To: \"Alice Liddell\" <sip:alice@wonderland.com>"), &toHeaderResult{pass,
            &ToHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{toHeaderInput("To: sip:alice@wonderland.com, sip:hatter@wonderland.com"), &toHeaderResult{fail,
            &ToHeader{}}},
    }, t)
}

func TestFromHeaders(t *testing.T) {
    // These are identical to the To: header tests, but there's no clean way to share them :(
    alice := "alice"
    aliceAddr := "sip:alice@wonderland.com"
    aliceAddrQuot := "<sip:alice@wonderland.com>"
    aliceAddrQuotSp := "<sip: alice@wonderland.com>"
    aliceTitle := "Alice"
    aliceLiddell := "Alice Liddell"
    bar := "bar"
    fooEqBar := map[string]*string{"foo" : &bar}
    fooSingleton := map[string]*string{"foo" : nil}
    noParams := map[string]*string{}
    doTests([]test {
        test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{fromHeaderInput("From:\n  \"Alice Liddell\" \n\t<sip:alice@wonderland.com>"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{fromHeaderInput("f: Alice <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceTitle,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{fromHeaderInput("From: Alice sip:alice@wonderland.com"), &fromHeaderResult{fail,
            &FromHeader{}}},

        test{fromHeaderInput("From:"), &fromHeaderResult{fail,
            &FromHeader{}}},

        test{fromHeaderInput("From: "), &fromHeaderResult{fail,
            &FromHeader{}}},

        test{fromHeaderInput("From:\t"), &fromHeaderResult{fail,
            &FromHeader{}}},

        test{fromHeaderInput("From: foo"), &fromHeaderResult{fail,
            &FromHeader{}}},

        test{fromHeaderInput("From: foo bar"), &fromHeaderResult{fail,
            &FromHeader{}}},

        test{fromHeaderInput("From: \"Alice\" sip:alice@wonderland.com"), &fromHeaderResult{fail,
            &FromHeader{}}},

        test{fromHeaderInput("From: \"<Alice>\" sip:alice@wonderland.com"), &fromHeaderResult{fail,
            &FromHeader{}}},

        test{fromHeaderInput("From: \"sip:alice@wonderland.com\""), &fromHeaderResult{fail,
            &FromHeader{}}},

        test{fromHeaderInput("From: \"sip:alice@wonderland.com\"  <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceAddr,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{fromHeaderInput("From: \"<sip:alice@wonderland.com>\"  <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceAddrQuot,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{fromHeaderInput("From: \"<sip: alice@wonderland.com>\"  <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceAddrQuotSp,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{fromHeaderInput("FrOm: \"Alice Liddell\" <sip:alice@wonderland.com>;foo=bar"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:fooEqBar}}},

        test{fromHeaderInput("from: \"Alice Liddell\" <sip:alice@wonderland.com;foo=bar>"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, fooEqBar, noParams},
                      params:noParams}}},

        test{fromHeaderInput("F: \"Alice Liddell\" <sip:alice@wonderland.com?foo=bar>"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, fooEqBar},
                      params:noParams}}},

        test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com>;foo"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:fooSingleton}}},

        test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com;foo>"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, fooSingleton, noParams},
                      params:noParams}}},

        test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com?foo>"), &fromHeaderResult{fail,
            &FromHeader{}}},

        test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com;foo?foo=bar>;foo=bar"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, fooSingleton, fooEqBar},
                      params:fooEqBar}}},

        test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com;foo?foo=bar>;foo"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, fooSingleton, fooEqBar},
                      params:fooSingleton}}},

        test{fromHeaderInput("From: \"Alice Liddell\" <sip:alice@wonderland.com>"), &fromHeaderResult{pass,
            &FromHeader{displayName:&aliceLiddell,
                      uri:&SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                      params:noParams}}},

        test{fromHeaderInput("From: sip:alice@wonderland.com, sip:hatter@wonderland.com"), &fromHeaderResult{fail,
            &FromHeader{}}},
    }, t)
}

func TestContactHeaders(t *testing.T) {
    alice := "alice"
    aliceAddr := "sip:alice@wonderland.com"
    aliceAddrQuot := "<sip:alice@wonderland.com>"
    aliceAddrQuotSp := "<sip: alice@wonderland.com>"
    aliceTitle := "Alice"
    aliceLiddell := "Alice Liddell"
    bar := "bar"
    fooEqBar := map[string]*string{"foo" : &bar}
    fooSingleton := map[string]*string{"foo" : nil}
    hatter := "hatter"
    noParams := map[string]*string{}
    doTests([]test {
        test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com>"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceLiddell,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                          params:noParams}}}},

        test{contactHeaderInput("Contact:\n  \"Alice Liddell\" \n\t<sip:alice@wonderland.com>"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceLiddell,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                          params:noParams}}}},

        test{contactHeaderInput("m: Alice <sip:alice@wonderland.com>"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceTitle,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                          params:noParams}}}},

        test{contactHeaderInput("Contact: Alice sip:alice@wonderland.com"), &contactHeaderResult{fail,
            []*ContactHeader {
            &ContactHeader{}}}},

        test{contactHeaderInput("Contact:"), &contactHeaderResult{fail,
            []*ContactHeader {
            &ContactHeader{}}}},

        test{contactHeaderInput("Contact: "), &contactHeaderResult{fail,
            []*ContactHeader {
            &ContactHeader{}}}},

        test{contactHeaderInput("Contact:\t"), &contactHeaderResult{fail,
            []*ContactHeader {
            &ContactHeader{}}}},

        test{contactHeaderInput("Contact: foo"), &contactHeaderResult{fail,
            []*ContactHeader {
            &ContactHeader{}}}},

        test{contactHeaderInput("Contact: foo bar"), &contactHeaderResult{fail,
            []*ContactHeader {
            &ContactHeader{}}}},

        test{contactHeaderInput("Contact: \"Alice\" sip:alice@wonderland.com"), &contactHeaderResult{fail,
            []*ContactHeader {
            &ContactHeader{}}}},

        test{contactHeaderInput("Contact: \"<Alice>\" sip:alice@wonderland.com"), &contactHeaderResult{fail,
            []*ContactHeader {
            &ContactHeader{}}}},

        test{contactHeaderInput("Contact: \"sip:alice@wonderland.com\""), &contactHeaderResult{fail,
            []*ContactHeader {
            &ContactHeader{}}}},

        test{contactHeaderInput("Contact: \"sip:alice@wonderland.com\"  <sip:alice@wonderland.com>"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceAddr,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                          params:noParams}}}},

        test{contactHeaderInput("Contact: \"<sip:alice@wonderland.com>\"  <sip:alice@wonderland.com>"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceAddrQuot,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                          params:noParams}}}},

        test{contactHeaderInput("Contact: \"<sip: alice@wonderland.com>\"  <sip:alice@wonderland.com>"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceAddrQuotSp,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                          params:noParams}}}},

        test{contactHeaderInput("cOntACt: \"Alice Liddell\" <sip:alice@wonderland.com>;foo=bar"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceLiddell,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                          params:fooEqBar}}}},

        test{contactHeaderInput("contact: \"Alice Liddell\" <sip:alice@wonderland.com;foo=bar>"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceLiddell,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, fooEqBar, noParams},
                          params:noParams}}}},

        test{contactHeaderInput("M: \"Alice Liddell\" <sip:alice@wonderland.com?foo=bar>"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceLiddell,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, noParams, fooEqBar},
                          params:noParams}}}},

        test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com>;foo"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceLiddell,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                          params:fooSingleton}}}},

        test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com;foo>"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceLiddell,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, fooSingleton, noParams},
                          params:noParams}}}},

        test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com?foo>"), &contactHeaderResult{fail,
            []*ContactHeader {
            &ContactHeader{}}}},

        test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com;foo?foo=bar>;foo=bar"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceLiddell,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, fooSingleton, fooEqBar},
                          params:fooEqBar}}}},

        test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com;foo?foo=bar>;foo"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceLiddell,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, fooSingleton, fooEqBar},
                          params:fooSingleton}}}},

        test{contactHeaderInput("Contact: \"Alice Liddell\" <sip:alice@wonderland.com>"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName:&aliceLiddell,
                          uri:SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams},
                          params:noParams}}}},

        test{contactHeaderInput("Contact: sip:alice@wonderland.com, sip:hatter@wonderland.com"), &contactHeaderResult{pass,
            []*ContactHeader {
                &ContactHeader{displayName: nil, uri:SipUri{false, &alice, nil, "wonderland.com", nil, noParams, noParams}, params:noParams},
                &ContactHeader{displayName: nil, uri:SipUri{false, &hatter, nil, "wonderland.com", nil, noParams, noParams}, params:noParams}}}},
    }, t)
}

type paramInput struct {
    paramString string
    start uint8
    sep uint8
    end uint8
    quoteValues bool
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
    err error
    params map[string]*string
    consumed int
}
func (expected *paramResult) equals (other result) (equal bool, reason string) {
    actual := *(other.(*paramResult))
    if expected.err == nil && actual.err != nil {
        return false, fmt.Sprintf("unexpected error: %s", actual.err.Error())
    } else if expected.err != nil && actual.err == nil {
        return false, fmt.Sprintf("unexpected success: got \"%s\"", ParamsToString(actual.params, '$', '-'))
    } else if actual.err == nil && !paramsEqual(expected.params, actual.params) {
        return false, fmt.Sprintf("unexpected result: expected \"%s\", got \"%s\"",
            ParamsToString(expected.params, '$', '-'), ParamsToString(actual.params, '$', '-'))
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
    output, err := parseSipUri(string(data))
    return &sipUriResult{err, output}
}

type sipUriResult struct {
    err error
    uri SipUri
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

    return expected.uri.equals(&actual.uri)
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
    err error
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
    contents, linesConsumed := getNextHeaderBlock([]string(data))
    return &headerBlockResult{contents, linesConsumed}
}

type headerBlockResult struct {
    contents string
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

type toHeaderInput string

func (data toHeaderInput) String() string {
    return string(data)
}

func (data toHeaderInput) evaluate() result {
    parser := NewMessageParser().(*parserImpl)
    headers, err := parser.parseHeaderSection(string(data))
    if len(headers) > 0 {
        return &toHeaderResult{err, headers[0].(*ToHeader)}
    } else {
        return &toHeaderResult{err, &ToHeader{}}
    }
}

type toHeaderResult struct {
    err error
    header *ToHeader
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

    if !strPtrEq(expected.header.displayName, actual.header.displayName) {
        return false, fmt.Sprintf("unexpected display name: expected \"%s\"; got \"%s\"",
            strPtrStr(expected.header.displayName),
            strPtrStr(actual.header.displayName))
    }

    switch expected.header.uri.(type) {
    case *SipUri:
        uri := *(expected.header.uri.(*SipUri))
        urisEqual, msg := uri.equals(actual.header.uri)
        if !urisEqual {
            return false, msg
        }
    default:
        // If you're hitting this block, then you need to do the following:
        // - implement a package-private 'equals' method for the URI schema being tested.
        // - add a case block above for that schema, using the 'equals' method in the same was as the existing SipUri block above.
        return false, fmt.Sprintf("no support for testing uri schema in uri \"%s\" - fix me!", expected.header.uri)
    }

    if !paramsEqual(expected.header.params, actual.header.params) {
        return false, fmt.Sprintf("unexpected parameters \"%s\" (expected \"%s\")",
            ParamsToString(actual.header.params, '$', '-'),
            ParamsToString(expected.header.params, '$', '-'))
    }

    return true, ""
}

type fromHeaderInput string

func (data fromHeaderInput) String() string {
    return string(data)
}

func (data fromHeaderInput) evaluate() result {
    parser := NewMessageParser().(*parserImpl)
    headers, err := parser.parseHeaderSection(string(data))
    if len(headers) > 0 {
        return &fromHeaderResult{err, headers[0].(*FromHeader)}
    } else {
        return &fromHeaderResult{err, &FromHeader{}}
    }
}

type fromHeaderResult struct {
    err error
    header *FromHeader
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

    if !strPtrEq(expected.header.displayName, actual.header.displayName) {
        return false, fmt.Sprintf("unexpected display name: expected \"%s\"; got \"%s\"",
            strPtrStr(expected.header.displayName),
            strPtrStr(actual.header.displayName))
    }

    switch expected.header.uri.(type) {
    case *SipUri:
        uri := *(expected.header.uri.(*SipUri))
        urisEqual, msg := uri.equals(actual.header.uri)
        if !urisEqual {
            return false, msg
        }
    default:
        // If you're hitting this block, then you need to do the following:
        // - implement a package-private 'equals' method for the URI schema being tested.
        // - add a case block above for that schema, using the 'equals' method in the same was as the existing SipUri block above.
        return false, fmt.Sprintf("no support for testing uri schema in uri \"%s\" - fix me!", expected.header.uri)
    }

    if !paramsEqual(expected.header.params, actual.header.params) {
        return false, fmt.Sprintf("unexpected parameters \"%s\" (expected \"%s\")",
            ParamsToString(actual.header.params, '$', '-'),
            ParamsToString(expected.header.params, '$', '-'))
    }

    return true, ""
}

type contactHeaderInput string

func (data contactHeaderInput) String() string {
    return string(data)
}

func (data contactHeaderInput) evaluate() result {
    parser := NewMessageParser().(*parserImpl)
    headers, err := parser.parseHeaderSection(string(data))
    contactHeaders := make([]*ContactHeader, len(headers))
    if len(headers) > 0 {
        for idx, header := range(headers) {
            contactHeaders[idx] = header.(*ContactHeader)
        }
        return &contactHeaderResult{err, contactHeaders}
    } else {
        return &contactHeaderResult{err, contactHeaders}
    }
}

type contactHeaderResult struct {
    err error
    headers []*ContactHeader
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
    for _, header := range(actual.headers) {
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

    for idx := range(expected.headers) {
        if !strPtrEq(expected.headers[idx].displayName, actual.headers[idx].displayName) {
            return false, fmt.Sprintf("unexpected display name: expected \"%s\"; got \"%s\"",
                strPtrStr(expected.headers[idx].displayName),
                strPtrStr(actual.headers[idx].displayName))
        }

        urisEqual, msg := expected.headers[idx].uri.equals(&actual.headers[idx].uri)
        if !urisEqual {
            return false, msg
        }

        if !paramsEqual(expected.headers[idx].params, actual.headers[idx].params) {
            return false, fmt.Sprintf("unexpected parameters \"%s\" (expected \"%s\")",
                ParamsToString(actual.headers[idx].params, '$', '-'),
                ParamsToString(expected.headers[idx].params, '$', '-'))
        }
    }

    return true, ""
}

func TestZZZCountTests (t *testing.T) {
    fmt.Printf("\n *** %d tests run *** \n\n", testsRun)
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

func (a *SipUri) equals(other Uri) (equal bool, reason string) {
    switch other.(type) {
    case *SipUri:
        b := *(other.(*SipUri))
        if a.IsEncrypted != b.IsEncrypted {
            return false, fmt.Sprintf("unexpected IsEncrypted value: expected %b; got %b",
                b.IsEncrypted, a.IsEncrypted)
        } else if !strPtrEq(b.User, a.User) {
            return false, fmt.Sprintf("unexpected User value: expected %s; got %s",
                strPtrStr(b.User), strPtrStr(a.User))
        } else if !strPtrEq(b.Password, a.Password) {
            return false, fmt.Sprintf("unexpected Password value: expected %s; got %s",
                strPtrStr(b.Password), strPtrStr(a.Password))
        } else if b.Host != a.Host {
            return false, fmt.Sprintf("unexpected Host value: expected %s; got %s",
                b.Host, a.Host)
        } else if !uint16PtrEq(b.Port, a.Port) {
            return false, fmt.Sprintf("unexpected Port value: expected %s; got %s",
                uint16PtrStr(b.Port), uint16PtrStr(a.Port))
        } else if !paramsEqual(b.UriParams, a.UriParams) {
            return false, fmt.Sprintf("unequal uri parameters: expected \"%s\"; got \"%s\"",
                ParamsToString(b.UriParams, ';', ';'),
                ParamsToString(a.UriParams, ';', ';'))
        } else if !paramsEqual(b.Headers, a.Headers) {
            return false, fmt.Sprintf("unequal uri headers; expected \"%s\"; got \"%s\"",
                ParamsToString(b.Headers, '?', '&'),
                ParamsToString(a.Headers, '?', '&'))
        }
        return true, ""
    default:
        return false, fmt.Sprintf("unexpected URI schema: expected URI was \"%s\"; got \"%s\"", a.String(), other.String())
    }

}
