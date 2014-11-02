package main

import (
	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
	"github.com/stefankopieczek/gossip/transport"
)

import "time"

const SENDER_ADDR string = "169.254.96.61:5060"
const RECEIVER_ADDR string = "169.254.255.255:5061"

// const RECEIVER_ADDR string = "169.254.138.185:5061"
const MESSAGE_INTERVAL time.Duration = time.Second

func send() {
	transportManager, err := transport.NewManager("udp", SENDER_ADDR)
	if err != nil {
		panic(err)
	}

	log.Info("About to send messages!")
	for {
		sendInvite(transportManager)
		time.Sleep(time.Second)
	}
}

func sendInvite(manager *transport.Manager) {
	targetUser := "recipient"
	message := base.NewRequest(
		base.INVITE,
		&base.SipUri{User: &targetUser, Host: "localhost"},
		"SIP/2.0",
		make([]base.SipHeader, 0),
		"Hello!")

	log.Info("Sending INVITE")
	err := manager.Send(RECEIVER_ADDR, message)
	if err != nil {
		panic(err)
	}
	log.Info("INVITE sent!")
}
