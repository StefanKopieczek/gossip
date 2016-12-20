package base

import (
	"errors"
	"strconv"
	"time"
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

func (hs *headers) Expires() (*time.Duration, error) {
	expiresHeaders, ok := hs.headers["expires"]
	if !ok {
		return nil, errors.New("expire header does not exist")
	}

	expires := expiresHeaders[0].(*GenericHeader)

	expiresInt, err := strconv.Atoi(expires.Contents)
	if err != nil {
		return nil, err
	}

	duration := time.Second * time.Duration(expiresInt)

	return &duration, nil
}
