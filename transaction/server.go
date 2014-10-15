package transaction

import (
	"errors"

	"github.com/discoviking/fsm"
	"github.com/stefankopieczek/gossip/log"
)

// SIP Server Transaction FSM
// Implements the behaviour described in RFC 3261 section 17.1

// FSM States
const (
	server_state_proceeding = iota
	server_state_completed
	server_state_confirmed
	server_state_terminated
)

// FSM Inputs
const (
	server_input_request fsm.Input = iota
	server_input_ack
	server_input_user_1xx
	server_input_user_2xx
	server_input_user_300_plus
	server_input_timer_g
	server_input_timer_h
	server_input_timer_i
	server_input_transport_err
)

func (tx *ServerTransaction) initFSM() {
	// Define Actions

	// Send response
	act_respond := func() fsm.Input {
		err := tx.transport.Send(tx.dest, tx.lastResp)
		if err != nil {
			return server_input_transport_err
		}

		return fsm.NO_INPUT
	}

	// Inform user of transport error
	act_trans_err := func() fsm.Input {
		tx.tu_err <- errors.New("failed to send response")
		return fsm.NO_INPUT
	}

	// Inform user of timeout error
	act_timeout := func() fsm.Input {
		tx.tu_err <- errors.New("timed out waiting for ACK")
		return fsm.NO_INPUT
	}

	// Define States

	// Proceeding
	server_state_def_proceeding := fsm.State{
		Index: server_state_proceeding,
		Outcomes: map[fsm.Input]fsm.Outcome{
			server_input_request:       {server_state_proceeding, act_respond},
			server_input_user_1xx:      {server_state_proceeding, act_respond},
			server_input_user_2xx:      {server_state_terminated, act_respond},
			server_input_user_300_plus: {server_state_completed, act_respond},
			server_input_transport_err: {server_state_terminated, act_trans_err},
		},
	}

	// Completed
	server_state_def_completed := fsm.State{
		Index: server_state_completed,
		Outcomes: map[fsm.Input]fsm.Outcome{
			server_input_request:       {server_state_completed, act_respond},
			server_input_ack:           {server_state_confirmed, fsm.NO_ACTION},
			server_input_user_1xx:      {server_state_completed, fsm.NO_ACTION},
			server_input_user_2xx:      {server_state_completed, fsm.NO_ACTION},
			server_input_user_300_plus: {server_state_completed, fsm.NO_ACTION},
			server_input_timer_g:       {server_state_completed, act_respond},
			server_input_timer_h:       {server_state_terminated, act_timeout},
			server_input_transport_err: {server_state_terminated, act_trans_err},
		},
	}

	// Confirmed
	server_state_def_confirmed := fsm.State{
		Index: server_state_confirmed,
		Outcomes: map[fsm.Input]fsm.Outcome{
			server_input_request:       {server_state_confirmed, fsm.NO_ACTION},
			server_input_user_1xx:      {server_state_confirmed, fsm.NO_ACTION},
			server_input_user_2xx:      {server_state_confirmed, fsm.NO_ACTION},
			server_input_user_300_plus: {server_state_confirmed, fsm.NO_ACTION},
			server_input_timer_i:       {server_state_terminated, fsm.NO_ACTION},
		},
	}

	// Terminated
	server_state_def_terminated := fsm.State{
		Index: server_state_terminated,
		Outcomes: map[fsm.Input]fsm.Outcome{
			server_input_request:       {server_state_terminated, fsm.NO_ACTION},
			server_input_ack:           {server_state_terminated, fsm.NO_ACTION},
			server_input_user_1xx:      {server_state_terminated, fsm.NO_ACTION},
			server_input_user_2xx:      {server_state_terminated, fsm.NO_ACTION},
			server_input_user_300_plus: {server_state_terminated, fsm.NO_ACTION},
		},
	}

	// Define FSM
	fsm, err := fsm.Define(
		server_state_def_proceeding,
		server_state_def_completed,
		server_state_def_confirmed,
		server_state_def_terminated,
	)
	if err != nil {
		log.Severe("Failed to define transaction FSM. Transaction will be dropped.")
		return
	}

	tx.fsm = fsm
}
