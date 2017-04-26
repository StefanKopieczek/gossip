package transaction

import (
	"errors"

	"github.com/discoviking/fsm"
	"github.com/remodoy/gossip/base"
	"github.com/remodoy/gossip/log"
	"github.com/remodoy/gossip/timing"
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
	server_input_delete
)

// Define actions.

// Send response
func (tx *ServerTransaction) act_respond() fsm.Input {
	err := tx.transport.Send(tx.dest, tx.lastResp)
	if err != nil {
		return server_input_transport_err
	}

	return fsm.NO_INPUT
}

// Send final response
func (tx *ServerTransaction) act_final() fsm.Input {
	err := tx.transport.Send(tx.dest, tx.lastResp)
	if err != nil {
		return server_input_transport_err
	}

	// Start timer J (we just reuse timer h)
	tx.timer_h = timing.AfterFunc(64*T1, func() {
		tx.fsm.Spin(server_input_timer_h)
	})

	return fsm.NO_INPUT
}

// Inform user of transport error
func (tx *ServerTransaction) act_trans_err() fsm.Input {
	tx.tu_err <- errors.New("failed to send response")
	return server_input_delete
}

// Inform user of timeout error
func (tx *ServerTransaction) act_timeout() fsm.Input {
	tx.tu_err <- errors.New("transaction timed out")
	return server_input_delete
}

// Just delete the transaction.
func (tx *ServerTransaction) act_delete() fsm.Input {
	tx.Delete()
	return fsm.NO_INPUT
}

// Send response and delete the transaction.
func (tx *ServerTransaction) act_respond_delete() fsm.Input {
	tx.Delete()

	err := tx.transport.Send(tx.dest, tx.lastResp)
	if err != nil {
		return server_input_transport_err
	}
	return fsm.NO_INPUT
}

// Choose the right FSM init function depending on request method.
func (tx *ServerTransaction) initFSM() {
	if tx.origin.Method == base.INVITE {
		tx.initInviteFSM()
	} else {
		tx.initNonInviteFSM()
	}
}

func (tx *ServerTransaction) initInviteFSM() {
	// Define States

	// Proceeding
	server_state_def_proceeding := fsm.State{
		Index: server_state_proceeding,
		Outcomes: map[fsm.Input]fsm.Outcome{
			server_input_request:       {server_state_proceeding, tx.act_respond},
			server_input_user_1xx:      {server_state_proceeding, tx.act_respond},
			server_input_user_2xx:      {server_state_terminated, tx.act_respond_delete},
			server_input_user_300_plus: {server_state_completed, tx.act_respond},
			server_input_transport_err: {server_state_terminated, tx.act_trans_err},
		},
	}

	// Completed
	server_state_def_completed := fsm.State{
		Index: server_state_completed,
		Outcomes: map[fsm.Input]fsm.Outcome{
			server_input_request:       {server_state_completed, tx.act_respond},
			server_input_ack:           {server_state_confirmed, fsm.NO_ACTION},
			server_input_user_1xx:      {server_state_completed, fsm.NO_ACTION},
			server_input_user_2xx:      {server_state_completed, fsm.NO_ACTION},
			server_input_user_300_plus: {server_state_completed, fsm.NO_ACTION},
			server_input_timer_g:       {server_state_completed, tx.act_respond},
			server_input_timer_h:       {server_state_terminated, tx.act_timeout},
			server_input_transport_err: {server_state_terminated, tx.act_trans_err},
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
			server_input_timer_i:       {server_state_terminated, tx.act_delete},
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
			server_input_delete:        {server_state_terminated, tx.act_delete},
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
	// Define States

	// Trying
	server_state_def_trying := fsm.State{
		Index: server_state_trying,
		Outcomes: map[fsm.Input]fsm.Outcome{
			server_input_request:       {server_state_trying, fsm.NO_ACTION},
			server_input_user_1xx:      {server_state_proceeding, tx.act_respond},
			server_input_user_2xx:      {server_state_completed, tx.act_respond},
			server_input_user_300_plus: {server_state_completed, tx.act_respond},
		},
	}

	// Proceeding
	server_state_def_proceeding := fsm.State{
		Index: server_state_proceeding,
		Outcomes: map[fsm.Input]fsm.Outcome{
			server_input_request:       {server_state_proceeding, tx.act_respond},
			server_input_user_1xx:      {server_state_proceeding, tx.act_respond},
			server_input_user_2xx:      {server_state_completed, tx.act_final},
			server_input_user_300_plus: {server_state_completed, tx.act_final},
			server_input_transport_err: {server_state_terminated, tx.act_trans_err},
		},
	}

	// Completed
	server_state_def_completed := fsm.State{
		Index: server_state_completed,
		Outcomes: map[fsm.Input]fsm.Outcome{
			server_input_request:       {server_state_completed, tx.act_respond},
			server_input_user_1xx:      {server_state_completed, fsm.NO_ACTION},
			server_input_user_2xx:      {server_state_completed, fsm.NO_ACTION},
			server_input_user_300_plus: {server_state_completed, fsm.NO_ACTION},
			server_input_timer_h:       {server_state_terminated, tx.act_timeout},
			server_input_transport_err: {server_state_terminated, tx.act_trans_err},
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
			server_input_delete:        {server_state_terminated, tx.act_delete},
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
