package base

import (
	"fmt"
	"strings"
)

func (hs *headers) From() (*FromHeader, bool) {
	if headers, ok := hs.headers["from"]; ok {
		return headers[0].(*FromHeader), true

	} else {
		return &FromHeader{}, false
	}
}

func (hs *headers) To() (*ToHeader, bool) {
	if headers, ok := hs.headers["to"]; ok {
		return headers[0].(*ToHeader), true
	} else {
		return &ToHeader{}, false
	}
}

func (hs *headers) Contact() (*ContactHeader, bool) {
	if headers, ok := hs.headers["contact"]; ok {
		return headers[0].(*ContactHeader), true
	} else {
		return &ContactHeader{}, false
	}
}

func (hs *headers) Via() ([]*ViaHeader, bool) {
	headers, ok := hs.headers["via"]
	if !ok {
		return nil, false
	}

	var viaHeaders []*ViaHeader
	for _, header := range headers {
		viaHeaders = append(viaHeaders, header.(*ViaHeader))
	}

	return viaHeaders, true
}

func (hs *headers) CallID() (*CallId, bool) {
	if headers, ok := hs.headers["call-id"]; ok {
		return headers[0].(*CallId), ok
	} else {
		return nil, false
	}
}

func (hs *headers) CSeq() (*CSeq, bool) {
	if headers, ok := hs.headers["cseq"]; ok {
		return headers[0].(*CSeq), ok
	}

	return &CSeq{}, false
}

func (hs *headers) HeaderContents(name string) ([]string, error) {
	headers := hs.headers[strings.ToLower(name)]
	if len(headers) < 1 {
		return nil, fmt.Errorf("missing header, name: %s", name)
	}

	var contents []string

	for _, header := range headers {
		headerStr := header.String()
		sep := strings.Index(headerStr, ":")
		contents = append(contents, strings.TrimSpace(headerStr[sep+1:]))
	}
	return contents, nil

}
