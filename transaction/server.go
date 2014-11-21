package transaction

import (
	"errors"
	"time"

	"github.com/discoviking/fsm"
	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
)

// SIP Server Transaction FSM
// Implements the behaviour described in RFC 3261 section 17.2

// FSM States
const (
	server_state_trying = iota
	server_state_proceeding
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

// Choose the right FSM init function depending on request method.
func (tx *ServerTransaction) initFSM() {
	if tx.origin.Method == base.INVITE {
		tx.initInviteFSM()
	} else {
		tx.initNonInviteFSM()
	}
}

func (tx *ServerTransaction) initInviteFSM() {
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

func (tx *ServerTransaction) initNonInviteFSM() {
	// Define Actions

	// Send response
	act_respond := func() fsm.Input {
		err := tx.transport.Send(tx.dest, tx.lastResp)
		if err != nil {
			return server_input_transport_err
		}

		return fsm.NO_INPUT
	}

	// Send final response
	act_final := func() fsm.Input {
		err := tx.transport.Send(tx.dest, tx.lastResp)
		if err != nil {
			return server_input_transport_err
		}

		// Start timer J (we just reuse timer h)
		tx.timer_h = time.AfterFunc(64*T1, func() {
			tx.fsm.Spin(server_input_timer_h)
		})

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

	// Trying
	server_state_def_trying := fsm.State{
		Index: server_state_trying,
		Outcomes: map[fsm.Input]fsm.Outcome{
			server_input_request:       {server_state_trying, fsm.NO_ACTION},
			server_input_user_1xx:      {server_state_proceeding, act_respond},
			server_input_user_2xx:      {server_state_completed, act_respond},
			server_input_user_300_plus: {server_state_completed, act_respond},
		},
	}

	// Proceeding
	server_state_def_proceeding := fsm.State{
		Index: server_state_proceeding,
		Outcomes: map[fsm.Input]fsm.Outcome{
			server_input_request:       {server_state_proceeding, act_respond},
			server_input_user_1xx:      {server_state_proceeding, act_respond},
			server_input_user_2xx:      {server_state_completed, act_final},
			server_input_user_300_plus: {server_state_completed, act_final},
			server_input_transport_err: {server_state_terminated, act_trans_err},
		},
	}

	// Completed
	server_state_def_completed := fsm.State{
		Index: server_state_completed,
		Outcomes: map[fsm.Input]fsm.Outcome{
			server_input_request:       {server_state_completed, act_respond},
			server_input_user_1xx:      {server_state_completed, fsm.NO_ACTION},
			server_input_user_2xx:      {server_state_completed, fsm.NO_ACTION},
			server_input_user_300_plus: {server_state_completed, fsm.NO_ACTION},
			server_input_timer_h:       {server_state_terminated, act_timeout},
			server_input_transport_err: {server_state_terminated, act_trans_err},
		},
	}

	// Terminated
	server_state_def_terminated := fsm.State{
		Index: server_state_terminated,
		Outcomes: map[fsm.Input]fsm.Outcome{
			server_input_request:       {server_state_terminated, fsm.NO_ACTION},
			server_input_user_1xx:      {server_state_terminated, fsm.NO_ACTION},
			server_input_user_2xx:      {server_state_terminated, fsm.NO_ACTION},
			server_input_user_300_plus: {server_state_terminated, fsm.NO_ACTION},
			server_input_timer_h:       {server_state_terminated, fsm.NO_ACTION},
		},
	}

	// Define FSM
	fsm, err := fsm.Define(
		server_state_def_trying,
		server_state_def_proceeding,
		server_state_def_completed,
		server_state_def_terminated,
	)
	if err != nil {
		log.Severe("Failed to define transaction FSM. Transaction will be dropped.")
		return
	}

	tx.fsm = fsm
}
