package transaction

import "github.com/discoviking/fsm"

// SIP Client Transaction FSM
// Implements the behaviour described in RFC 3261 section 17.1

// FSM States
const (
	client_state_calling = iota
	client_state_proceeding
	client_state_completed
	client_state_terminated
)

// FSM Inputs
const (
	client_input_1xx fsm.Input = iota
	client_input_2xx
	client_input_300_plus
	client_input_timer_a
	client_input_timer_b
	client_input_timer_d
	client_input_transport_err
)

func (tx *ClientTransaction) initFSM() {
	// Define Actions

	// Resend the request.
	act_resend := func() fsm.Input {
		tx.timer_a_time *= 2
		tx.timer_a.Reset(tx.timer_a_time)
		tx.resend()
		return fsm.NO_INPUT
	}

	// Just pass up the latest response.
	act_passup := func() fsm.Input {
		tx.passUp()
		return fsm.NO_INPUT
	}

	// Handle 300+ responses.
	// Pass up response and send ACK.
	act_300 := func() fsm.Input {
		tx.passUp()
		tx.Ack()
		return fsm.NO_INPUT
	}

	// Send an ACK.
	act_ack := func() fsm.Input {
		tx.Ack()
		return fsm.NO_INPUT
	}

	// Send up transport failure error.
	act_trans_err := func() fsm.Input {
		tx.transportError()
		return fsm.NO_INPUT
	}

	// Send up timeout error.
	act_timeout := func() fsm.Input {
		tx.timeoutError()
		return fsm.NO_INPUT
	}

	// Define States

	// Calling
	client_state_def_calling := fsm.State{
		Index: client_state_calling,
		Outcomes: map[fsm.Input]fsm.Outcome{
			client_input_1xx:           {client_state_proceeding, act_passup},
			client_input_2xx:           {client_state_terminated, act_passup},
			client_input_300_plus:      {client_state_completed, act_300},
			client_input_timer_a:       {client_state_calling, act_resend},
			client_input_timer_b:       {client_state_terminated, act_timeout},
			client_input_transport_err: {client_state_terminated, act_trans_err},
		},
	}

	// Proceeding
	client_state_def_proceeding := fsm.State{
		Index: client_state_proceeding,
		Outcomes: map[fsm.Input]fsm.Outcome{
			client_input_1xx:      {client_state_proceeding, act_passup},
			client_input_2xx:      {client_state_terminated, act_passup},
			client_input_300_plus: {client_state_completed, act_300},
		},
	}

	// Completed
	client_state_def_completed := fsm.State{
		Index: client_state_completed,
		Outcomes: map[fsm.Input]fsm.Outcome{
			client_input_1xx:           {client_state_completed, fsm.NO_ACTION},
			client_input_2xx:           {client_state_completed, fsm.NO_ACTION},
			client_input_300_plus:      {client_state_completed, act_ack},
			client_input_timer_d:       {client_state_terminated, fsm.NO_ACTION},
			client_input_transport_err: {client_state_terminated, act_trans_err},
		},
	}

	// Terminated
	client_state_def_terminated := fsm.State{
		Index: client_state_terminated,
		Outcomes: map[fsm.Input]fsm.Outcome{
			client_input_1xx:      {client_state_terminated, fsm.NO_ACTION},
			client_input_2xx:      {client_state_terminated, fsm.NO_ACTION},
			client_input_300_plus: {client_state_terminated, fsm.NO_ACTION},
		},
	}

	fsm := fsm.Define(
		client_state_def_calling,
		client_state_def_proceeding,
		client_state_def_completed,
		client_state_def_terminated,
	)

	tx.fsm = fsm
}
