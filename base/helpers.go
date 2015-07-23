package base

func (hs *headers) From() (*FromHeader, bool) {
	if headers, ok := hs.headers["from"]; ok {
		return headers[0].(*FromHeader), true

	} else {
		return &FromHeader{}, false
	}
}

func (hs *headers) Contact() (*ContactHeader, bool) {
	if headers, ok := hs.headers["contact"]; ok {
		return headers[0].(*ContactHeader), true
	} else {
		return &ContactHeader{}, false
	}
}
