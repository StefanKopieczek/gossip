package parser

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
)

func TestParseAddressValue(t *testing.T) {
	// Make sure there are no out of index errors
	var err error

	// check for panics
	_, _, _, err = ParseAddressValue("*")
	_, _, _, err = ParseAddressValue("<*")
	_, _, _, err = ParseAddressValue("<")
	_, _, _, err = ParseAddressValue("*")

	_, _, _, err = ParseAddressValue("<sip:15@10.10.1.99>")
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, err = ParseAddressValue(`<sip:01214248526@10.105.80.114>;tag=25526~ffa80926-5fac-4dd6-b405-2dbbc56ae9a2-551664735`)
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, err = ParseAddressValue(`"A. G. Bell" <sip:agb@bell-telephone.com>;tag=a48s`)
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, err = ParseAddressValue(`sip:+12125551212@server.phone2net.com;tag=887s`)
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, err = ParseAddressValues(`<sip:bigbox3.site3.atlanta.com;lr>,<sip:server10.biloxi.com;lr>`)
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, err = ParseAddressValues(`<sip:alice@atlanta.com>, <sip:bob@biloxi.com>`)
	if err != nil {
		t.Fatal(err)
	}

}

func TestParseURI(t *testing.T) {
	return
	uri := "sip:bob@example.com:5;foo;baz=bar;a=b?foo=bar"
	tagIndex := strings.Index(uri, ";")

	var tags string
	if tagIndex != -1 {
		// we will have an opaque part -- remove the tags
		paramIndex := strings.Index(uri, "?")
		if paramIndex == -1 {
			paramIndex = len(uri)
		}

		tags = uri[tagIndex:paramIndex]
		uri = uri[:tagIndex] + uri[paramIndex:]

	}

	u, err := url.Parse(uri)
	if err != nil {
		fmt.Println(err)
		fmt.Println(tags)
		fmt.Println(u)
		return
	}

}

func TestSIPSURI(t *testing.T) {
	to := `"Alice Liddell" <sips:alice@wonderland.com>`

	_, b, _, err := ParseAddressValue(to)
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != "sips:alice@wonderland.com" {
		t.Fatalf("Expected: sips:alice@wonderland.com Got: %s", b)
	}

}
