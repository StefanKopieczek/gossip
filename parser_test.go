package gossip

import "fmt"
//import "strings"
//import "strconv"
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


func TestParams(t *testing.T) {
    // Need named variables for pointer field values.
    bar := "bar"
    barQuote := "\"bar\""
    barQuote2 := "\"bar"
    barQuote3 := "bar\""
    barBaz := "bar;baz"
    //baz := "baz"
    bob := "bob"
    boop := "boop"
    b := "b"
    empty := ""
    //hunter2 := "Hunter2"
    //port5060 := uint16(5060)
    //port9 := uint16(9)
    //ui16_5 := uint16(5)
    //ui16_5060 := uint16(5060)
    doTests([]test {
        // TEST: parseParams
        test{&paramInput{";foo=bar",              ';', ';',  0,  false, true},   &paramResult{pass, map[string]*string{"foo":&bar},                          8}},
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
/*
func getSipUriTests() map[sipUriTest]sipUriResult {
    // Need named variables for pointer fields in SipUri struct.
    b := "b"
    bar := "bar"
    baz := "baz"
    bob := "bob"
    emptyString := ""
    hunter2 := "Hunter2"
    ui16_5 := uint16(5)
    ui16_5060 := uint16(5060)

    sipUriTests := map[sipUriTest]sipUriResult {
        "sip:bob@example.com"                          : sipUriResult{true, SipUri{User:&bob, Host:"example.com"}},
        "sip:bob@192.168.0.1"                          : sipUriResult{true, SipUri{User:&bob, Host:"192.168.0.1"}},
        "sip:bob:Hunter2@example.com"                  : sipUriResult{true, SipUri{User:&bob, Password:&hunter2, Host:"example.com"}},
        "sips:bob:Hunter2@example.com"                 : sipUriResult{true, SipUri{IsEncrypted:true, User:&bob, Password:&hunter2, Host:"example.com"}},
        "sips:bob@example.com"                         : sipUriResult{true, SipUri{IsEncrypted:true, User:&bob, Host:"example.com"}},
        "sip:example.com"                              : sipUriResult{true, SipUri{Host:"example.com"}},
        "example.com"                                  : sipUriResult{false, SipUri{}},
        "bob@example.com"                              : sipUriResult{false, SipUri{}},
        "sip:bob@example.com:5060"                     : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5060}},
        "sip:bob@88.88.88.88:5060"                     : sipUriResult{true, SipUri{User:&bob, Host:"88.88.88.88", Port:&ui16_5060}},
        "sip:bob:Hunter2@example.com:5060"             : sipUriResult{true, SipUri{User:&bob, Password:&hunter2, Host:"example.com", Port:&ui16_5060}},
        "sip:bob@example.com:5"                        : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5}},
        "sip:bob@example.com;foo=bar"                  : sipUriResult{true, SipUri{User:&bob, Host:"example.com",
                                                            UriParams:map[string]*string{"foo":&bar}}},
        "sip:bob@example.com:5060;foo=bar"             : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5060,
                                                            UriParams:map[string]*string{"foo":&bar}}},
        "sip:bob@example.com:5;foo"                    : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                            UriParams:map[string]*string{"foo":nil}}},
        "sip:bob@example.com:5;foo;baz=bar"            : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                            UriParams:map[string]*string{"foo":nil, "baz":&bar}}},
        "sip:bob@example.com:5;baz=bar;foo"            : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                            UriParams:map[string]*string{"foo":nil, "baz":&bar}}},
        "sip:bob@example.com:5;foo;baz=bar;a=b"        : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                            UriParams:map[string]*string{"foo":nil, "baz":&bar, "a":&b}}},
        "sip:bob@example.com:5;baz=bar;foo;a=b"        : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                            UriParams:map[string]*string{"foo":nil, "baz":&bar, "a":&b}}},
        "sip:bob@example.com?foo=bar"                  : sipUriResult{true, SipUri{User:&bob, Host:"example.com",
                                                            Headers:map[string]*string{"foo":&bar}}},
        "sip:bob@example.com?foo="                     : sipUriResult{true, SipUri{User:&bob, Host:"example.com",
                                                            Headers:map[string]*string{"foo":&emptyString}}},
        "sip:bob@example.com:5060?foo=bar"             : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5060,
                                                            Headers:map[string]*string{"foo":&bar}}},
        "sip:bob@example.com:5?foo=bar"                : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                            Headers:map[string]*string{"foo":&bar}}},
        "sips:bob@example.com:5?baz=bar&foo=&a=b"      : sipUriResult{true, SipUri{IsEncrypted:true, User:&bob, Host:"example.com", Port:&ui16_5,
                                                            Headers:map[string]*string{"baz":&bar, "a":&b, "foo":&emptyString}}},
        "sip:bob@example.com:5?baz=bar&foo&a=b"        : sipUriResult{false, SipUri{}},
        "sip:bob@example.com:5?foo"                    : sipUriResult{false, SipUri{}},
        "sip:bob@example.com:50?foo"                   : sipUriResult{false, SipUri{}},
        "sip:bob@example.com:50?foo=bar&baz"           : sipUriResult{false, SipUri{}},
        "sip:bob@example.com;foo?foo=bar"              : sipUriResult{true, SipUri{User:&bob, Host:"example.com",
                                                            UriParams:map[string]*string{"foo":nil},
                                                            Headers:map[string]*string{"foo":&bar}}},
        "sip:bob@example.com:5060;foo?foo=bar"         : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5060,
                                                            UriParams:map[string]*string{"foo":nil},
                                                            Headers:map[string]*string{"foo":&bar}}},
        "sip:bob@example.com:5;foo?foo=bar"            : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                            UriParams:map[string]*string{"foo":nil},
                                                            Headers:map[string]*string{"foo":&bar}}},
        "sips:bob@example.com:5;foo?baz=bar&a=b&foo="  : sipUriResult{true, SipUri{IsEncrypted:true, User:&bob, Host:"example.com", Port:&ui16_5,
                                                            UriParams:map[string]*string{"foo":nil},
                                                            Headers:map[string]*string{"baz":&bar, "a":&b, "foo":&emptyString}}},
        "sip:bob@example.com:5;foo?baz=bar&foo&a=b"    : sipUriResult{false, SipUri{}},
        "sip:bob@example.com:5;foo?foo"                : sipUriResult{false, SipUri{}},
        "sip:bob@example.com:50;foo?foo"               : sipUriResult{false, SipUri{}},
        "sip:bob@example.com:50;foo?foo=bar&baz"       : sipUriResult{false, SipUri{}},
        "sip:bob@example.com;foo=baz?foo=bar"          : sipUriResult{true, SipUri{User:&bob, Host:"example.com",
                                                            UriParams:map[string]*string{"foo":&baz},
                                                            Headers:map[string]*string{"foo":&bar}}},
        "sip:bob@example.com:5060;foo=baz?foo=bar"     : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5060,
                                                            UriParams:map[string]*string{"foo":&baz},
                                                            Headers:map[string]*string{"foo":&bar}}},
        "sip:bob@example.com:5;foo=baz?foo=bar"        : sipUriResult{true, SipUri{User:&bob, Host:"example.com", Port:&ui16_5,
                                                            UriParams:map[string]*string{"foo":&baz},
                                                            Headers:map[string]*string{"foo":&bar}}},
        "sips:bob@example.com:5;foo=baz?baz=bar&a=b"   : sipUriResult{true, SipUri{IsEncrypted:true, User:&bob, Host:"example.com", Port:&ui16_5,
                                                            UriParams:map[string]*string{"foo":&baz},
                                                            Headers:map[string]*string{"baz":&bar, "a":&b}}},
        "sip:bob@example.com:5;foo=baz?baz=bar&foo&a=b": sipUriResult{false, SipUri{}},
        "sip:bob@example.com:5;foo=baz?foo"            : sipUriResult{false, SipUri{}},
        "sip:bob@example.com:50;foo=baz?foo"           : sipUriResult{false, SipUri{}},
        "sip:bob@example.com:50;foo=baz?foo=bar&baz"   : sipUriResult{false, SipUri{}},
    }

    return sipUriTests
}

func getHostPortTests() (map[hostPortTest]hostPortResult) {
    port5060 := uint16(5060)
    port9 := uint16(9)

    hostPortTests := map[hostPortTest]hostPortResult {
        "example.com"        : hostPortResult{true, "example.com", nil},
        "192.168.0.1"        : hostPortResult{true, "192.168.0.1", nil},
        "abc123"             : hostPortResult{true, "abc123",      nil},
        "example.com:5060"   : hostPortResult{true, "example.com", &port5060},
        "example.com:9"      : hostPortResult{true, "example.com", &port9},
        "192.168.0.1:5060"   : hostPortResult{true, "192.168.0.1", &port5060},
        "192.168.0.1:9"      : hostPortResult{true, "192.168.0.1", &port9},
        "abc123:5060"        : hostPortResult{true, "abc123",      &port5060},
        "abc123:9"           : hostPortResult{true, "abc123",      &port9},
        // TODO IPV6, c.f. IPv6reference in RFC 3261 s25
    }

    return hostPortTests
}

func getHeaderBlockTests() (map[headerBlockTest]headerBlockResult) {
    return map[headerBlockTest]headerBlockResult {
        []string{"All on one line."}                             : headerBlockResult{"All on one line.", 1},
        []string{"Line one", "Line two."}                        : headerBlockResult{"Line one", 1},
        []string{"Line one", " then an indent"}                  : headerBlockResult{"Line one then an indent", 2},
        []string{"Line one", " then an indent", "then line two"} : headerBlockResult{"Line one then an indent", 2},
        []string{"Line one", "Line two", " then an indent"}      : headerBlockResult{"Line one", 1},
        []string{"Line one", "\twith tab indent"}                : headerBlockResult{"Line one with tab indent", 2},
        []string{"Line one", "      with a big indent"}          : headerBlockResult{"Line one with a big indent", 2},
        []string{"Line one", " \twith space then tab"}           : headerBlockResult{"Line one with space then tab", 2},
        []string{"Line one", "\t    with tab then spaces"}       : headerBlockResult{"Line one with tab then spaces", 2},
        []string{""}                                             : headerBlockResult{"", 0},
        []string{" "}                                            : headerBlockResult{" ", 1},
        []string{" foo"}                                         : headerBlockResult{" foo", 1},
    }
}

func TestParseHostPort(t *testing.T) {
    for test, expected := range(getHostPortTests()) {
        host, port, err := parseHostPort(string(test))
        totalTests++

        if err != nil && expected.success {
            t.Error(fmt.Sprintf("Unexpected failure on parsing \"%s\": %s", test, err.Error()))
            continue
        }

        parsedStr := host
        if port != nil {
        }
    }
    // TODO
}*/

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

func TestZZZCountTests (t *testing.T) {
    fmt.Printf("\n *** %d tests run *** \n\n", testsRun)
}
