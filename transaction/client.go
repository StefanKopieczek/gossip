package transaction

import (
	"github.com/discoviking/fsm"
	"github.com/remodoy/gossip/base"
	"github.com/remodoy/gossip/log"
	"github.com/remodoy/gossip/timing"
)

// SIP Client Transaction FSM
// Implements the behaviour described in RFC 3261 section 17.1

// FSM States
const (
	client_state_calling = iota
	client_state_proceeding
	client_state_completed
	client_state_terminated
	client_state_on_call
    client_state_canceling
    client_state_bye_sent
)

// FSM Inputs
const (
	client_input_1xx fsm.Input = iota
	client_input_2xx
	client_input_300_plus
	client_input_bye
	client_input_timer_a
	client_input_timer_b
	client_input_timer_d
	client_input_transport_err
	client_input_delete
    client_input_terminate // Selectively CANCEL or BYE to terminate
)

// Initialises the correct kind of FSM based on request method.
func (tx *ClientTransaction) initFSM() {
	if tx.origin.Method == base.INVITE {
		tx.initInviteFSM()
	} else {
		tx.initNonInviteFSM()
	}
}

func (tx *ClientTransaction) initInviteFSM() {
	log.Debug("Initialising client INVITE transaction FSM")

	// Define Actions

	// Resend the request.
	act_resend := func() fsm.Input {
		log.Debug("Client transaction %p, act_resend", tx)
		tx.timer_a_time *= 2
		tx.timer_a.Reset(tx.timer_a_time)
		tx.resend()
		return fsm.NO_INPUT
	}

	// Just pass up the latest response.
	act_passup := func() fsm.Input {
		log.Debug("Client transaction %p, act_passup", tx)
		tx.passUp()
		return fsm.NO_INPUT
	}

	// Handle 300+ responses.
	// Pass up response and send ACK, start timer D.
	act_300 := func() fsm.Input {
		log.Debug("Client transaction %p, act_300", tx)
		tx.passUp()
		tx.Ack()
		if tx.timer_d != nil {
			tx.timer_d.Stop()
		}
		tx.timer_d = timing.AfterFunc(tx.timer_d_time, func() {
			tx.fsm.Spin(client_input_timer_d)
		})
		return fsm.NO_INPUT
	}

	// Send an ACK.
	act_ack := func() fsm.Input {
		log.Debug("Client transaction %p, act_ack", tx)
		tx.Ack()
		return fsm.NO_INPUT
	}

	// Send up transport failure error.
	act_trans_err := func() fsm.Input {
		log.Debug("Client transaction %p, act_trans_err", tx)
		tx.transportError()
		return client_input_delete
	}

	// Send up timeout error.
	act_timeout := func() fsm.Input {
		log.Debug("Client transaction %p, act_timeout", tx)
		tx.timeoutError()
		return client_input_delete
	}

	// Pass up the response and delete the transaction.
	act_passup_delete := func() fsm.Input {
		log.Debug("Client transaction %p, act_passup_delete", tx)
		tx.passUpRequest()
		tx.Delete()
		return fsm.NO_INPUT
	}

	// Just delete the transaction.
	act_delete := func() fsm.Input {
		log.Debug("Client transaction %p, act_delete", tx)
		tx.Delete()
		return fsm.NO_INPUT
	}

    // Cancel ongoing invite
    act_cancel := func() fsm.Input {
        log.Debug("Client transaction %p, act_cancel", tx)
        tx.sendCancel();
        return fsm.NO_INPUT
    }

    act_bye := func() fsm.Input {
        log.Debug("Client transaction %p, act_cancel", tx)
        tx.sendBye();
        return fsm.NO_INPUT
    }

	// Define States

	// Calling
	client_state_def_calling := fsm.State{
		Index: client_state_calling,
		Outcomes: map[fsm.Input]fsm.Outcome{
			client_input_1xx:           {client_state_proceeding, act_passup},
			client_input_2xx:           {client_state_on_call, act_passup},
			client_input_300_plus:      {client_state_completed, act_300},
            client_input_bye:           {client_state_terminated, fsm.NO_ACTION},
			client_input_timer_a:       {client_state_calling, act_resend},
			client_input_timer_b:       {client_state_terminated, act_timeout},
			client_input_transport_err: {client_state_terminated, act_trans_err},
            client_input_terminate:     {client_state_canceling, fsm.NO_ACTION},
		},
	}

	// Proceeding
	client_state_def_proceeding := fsm.State{
		Index: client_state_proceeding,
		Outcomes: map[fsm.Input]fsm.Outcome{
			client_input_1xx:      {client_state_proceeding, act_passup},
			client_input_2xx:      {client_state_on_call, act_passup},
			client_input_300_plus: {client_state_completed, act_300},
            client_input_bye:      {client_state_proceeding, fsm.NO_ACTION},
			client_input_timer_a:  {client_state_proceeding, fsm.NO_ACTION},
			client_input_timer_b:  {client_state_proceeding, fsm.NO_ACTION},
            client_input_terminate:{client_state_canceling, act_cancel},
		},
	}

    // On call
    client_state_def_on_call := fsm.State{
        Index: client_state_on_call,
        Outcomes: map[fsm.Input]fsm.Outcome{
            client_input_1xx:           {client_state_on_call, fsm.NO_ACTION},
            client_input_2xx:           {client_state_on_call, fsm.NO_ACTION},
            client_input_300_plus:      {client_state_on_call, fsm.NO_ACTION},
            client_input_bye:           {client_state_terminated, act_passup_delete},
            client_input_timer_a:       {client_state_on_call, fsm.NO_ACTION},
            client_input_timer_b:       {client_state_on_call, fsm.NO_ACTION},
            client_input_terminate:     {client_state_bye_sent, act_bye},
        },
    }

	// Completed
	client_state_def_completed := fsm.State{
		Index: client_state_completed,
		Outcomes: map[fsm.Input]fsm.Outcome{
			client_input_1xx:           {client_state_completed, fsm.NO_ACTION},
			client_input_2xx:           {client_state_completed, fsm.NO_ACTION},
			client_input_300_plus:      {client_state_completed, act_ack},
            client_input_bye:           {client_state_completed, fsm.NO_ACTION},
			client_input_timer_d:       {client_state_terminated, act_delete},
			client_input_transport_err: {client_state_terminated, act_trans_err},
			client_input_timer_a:       {client_state_completed, fsm.NO_ACTION},
			client_input_timer_b:       {client_state_completed, fsm.NO_ACTION},
		},
	}

	// Terminated
	client_state_def_terminated := fsm.State{
		Index: client_state_terminated,
		Outcomes: map[fsm.Input]fsm.Outcome{
			client_input_1xx:      {client_state_terminated, fsm.NO_ACTION},
			client_input_2xx:      {client_state_terminated, fsm.NO_ACTION},
			client_input_300_plus: {client_state_terminated, fsm.NO_ACTION},
            client_input_bye:      {client_state_terminated, fsm.NO_ACTION},
			client_input_timer_a:  {client_state_terminated, fsm.NO_ACTION},
			client_input_timer_b:  {client_state_terminated, fsm.NO_ACTION},
			client_input_delete:   {client_state_terminated, act_delete},
		},
	}

    // Canceling
    client_state_def_canceling := fsm.State{
        Index: client_state_canceling,
        Outcomes: map[fsm.Input]fsm.Outcome{
            client_input_1xx:      {client_state_canceling, act_cancel},
            client_input_2xx:      {client_state_bye_sent, act_bye}, // CANCEL was too late, need BYE
            client_input_300_plus: {client_state_terminated, fsm.NO_ACTION},
            client_input_bye:      {client_state_terminated, fsm.NO_ACTION},
            client_input_timer_a:  {client_state_canceling, act_cancel}, // XXX
            client_input_timer_b:  {client_state_terminated, fsm.NO_ACTION}, // XXX
            client_input_delete:   {client_state_terminated, act_delete},
        },
    }

    // BYE sent
    client_state_def_bye_sent := fsm.State{
        Index: client_state_bye_sent,
        Outcomes: map[fsm.Input]fsm.Outcome{
            client_input_1xx:      {client_state_canceling, act_cancel}, // ???
            client_input_2xx:      {client_state_on_call, fsm.NO_ACTION}, // YAY, session dead
            client_input_300_plus: {client_state_terminated, fsm.NO_ACTION},
            client_input_bye:      {client_state_terminated, fsm.NO_ACTION},
            client_input_timer_a:  {client_state_canceling, act_cancel}, // XXX
            client_input_timer_b:  {client_state_terminated, fsm.NO_ACTION}, // XXX
            client_input_delete:   {client_state_terminated, act_delete},
        },
    }

	fsm, err := fsm.Define(
		client_state_def_calling,
		client_state_def_proceeding,
        client_state_def_on_call,
		client_state_def_completed,
		client_state_def_terminated,
        client_state_def_canceling,
        client_state_def_bye_sent,
	)

	if err != nil {
		log.Severe("Failure to define INVITE client transaction fsm: %s", err.Error())
	}

	tx.fsm = fsm
}

func (tx *ClientTransaction) initNonInviteFSM() {
	log.Debug("Initialising client non-INVITE transaction FSM")

	// Define Actions

	// Resend the request.
	act_resend := func() fsm.Input {
		tx.timer_a_time *= 2
		// For non-INVITE, cap timer A at T2 seconds.
		if tx.timer_a_time > T2 {
			tx.timer_a_time = T2
		}

		tx.timer_a.Reset(tx.timer_a_time)
		tx.resend()
		return fsm.NO_INPUT
	}

	// Just pass up the latest response.
	act_passup := func() fsm.Input {
		tx.passUp()
		return fsm.NO_INPUT
	}

	// Handle a final response.
	act_final := func() fsm.Input {
		tx.passUp()
		if tx.timer_d != nil {
			tx.timer_d.Stop()
		}
		tx.timer_d = timing.AfterFunc(tx.timer_d_time, func() {
			tx.fsm.Spin(client_input_timer_d)
		})
		return fsm.NO_INPUT
	}

	// Send up transport failure error.
	act_trans_err := func() fsm.Input {
		tx.transportError()
		return client_input_delete
	}

	// Send up timeout error.
	act_timeout := func() fsm.Input {
		tx.timeoutError()
		return client_input_delete
	}

	// Just delete the transaction.
	act_delete := func() fsm.Input {
		tx.Delete()
		return fsm.NO_INPUT
	}

	// Define States

	// "Trying"
	client_state_def_calling := fsm.State{
		Index: client_state_calling,
		Outcomes: map[fsm.Input]fsm.Outcome{
			client_input_1xx:           {client_state_proceeding, act_passup},
			client_input_2xx:           {client_state_completed, act_final},
			client_input_300_plus:      {client_state_completed, act_final},
			client_input_timer_a:       {client_state_calling, act_resend},
			client_input_timer_b:       {client_state_terminated, act_timeout},
			client_input_transport_err: {client_state_terminated, act_trans_err},
		},
	}

	// Proceeding
	client_state_def_proceeding := fsm.State{
		Index: client_state_proceeding,
		Outcomes: map[fsm.Input]fsm.Outcome{
			client_input_1xx:           {client_state_proceeding, act_passup},
			client_input_2xx:           {client_state_completed, act_final},
			client_input_300_plus:      {client_state_completed, act_final},
			client_input_timer_a:       {client_state_proceeding, act_resend},
			client_input_timer_b:       {client_state_terminated, act_timeout},
			client_input_transport_err: {client_state_terminated, act_trans_err},
		},
	}

	// Completed
	client_state_def_completed := fsm.State{
		Index: client_state_completed,
		Outcomes: map[fsm.Input]fsm.Outcome{
			client_input_1xx:      {client_state_completed, fsm.NO_ACTION},
			client_input_2xx:      {client_state_completed, fsm.NO_ACTION},
			client_input_300_plus: {client_state_completed, fsm.NO_ACTION},
			client_input_timer_d:  {client_state_terminated, act_delete},
			client_input_timer_a:  {client_state_completed, fsm.NO_ACTION},
			client_input_timer_b:  {client_state_completed, fsm.NO_ACTION},
		},
	}

	// Terminated
	client_state_def_terminated := fsm.State{
		Index: client_state_terminated,
		Outcomes: map[fsm.Input]fsm.Outcome{
			client_input_1xx:      {client_state_terminated, fsm.NO_ACTION},
			client_input_2xx:      {client_state_terminated, fsm.NO_ACTION},
			client_input_300_plus: {client_state_terminated, fsm.NO_ACTION},
			client_input_timer_a:  {client_state_terminated, fsm.NO_ACTION},
			client_input_timer_b:  {client_state_terminated, fsm.NO_ACTION},
			client_input_timer_d:  {client_state_terminated, fsm.NO_ACTION},
			client_input_delete:   {client_state_terminated, act_delete},
		},
	}

	fsm, err := fsm.Define(
		client_state_def_calling,
		client_state_def_proceeding,
		client_state_def_completed,
		client_state_def_terminated,
	)

	if err != nil {
		log.Severe("Failure to define INVITE client transaction fsm: %s", err.Error())
	}

	tx.fsm = fsm
}
