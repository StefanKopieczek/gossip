package transaction

import (
	"testing"
)

var c_SERVER string = "localhost:5060"
var c_CLIENT string = "localhost:5061"

func TestSendInvite(t *testing.T) {
	invite, err := request([]string{
		"INVITE sip:joe@bloggs.com SIP/2.0",
		"Via: SIP/2.0/UDP " + c_CLIENT + ";branch=z9hG4bK776asdhds",
		"",
		"",
	})
	assertNoError(t, err)

	test := transactionTest{t: t,
		actions: []action{
			&userSend{invite},
			&transportRecv{invite},
		}}
	test.Execute()
}

func TestReceiveOK(t *testing.T) {
	invite, err := request([]string{
		"INVITE sip:joe@bloggs.com SIP/2.0",
		"CSeq: 1 INVITE",
		"Via: SIP/2.0/UDP " + c_CLIENT + ";branch=z9hG4bK776asdhds",
		"",
		"",
	})
	assertNoError(t, err)

	ok, err := response([]string{
		"SIP/2.0 200 OK",
		"CSeq: 1 INVITE",
		"Via: SIP/2.0/UDP " + c_SERVER + ";branch=z9hG4bK776asdhds",
		"",
		"",
	})
	assertNoError(t, err)

	test := transactionTest{t: t,
		actions: []action{
			&userSend{invite},
			&transportRecv{invite},
			&transportSend{ok},
			&userRecv{ok},
		}}
	test.Execute()
}
