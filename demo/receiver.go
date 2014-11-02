package main

import "fmt"

import (
	"github.com/stefankopieczek/gossip/log"
	"github.com/stefankopieczek/gossip/transport"
)

const LOCAL_ADDR string = "oryx.home:20780"

func receive() {
	transportManager, err := transport.NewManager("tcp", LOCAL_ADDR)
	if err != nil {
		panic(err)
	}

	if transportManager == nil {
		panic("Transport manager is nil!")
	}

	transportManager.Listen()
	log.Info("Ready to receive messages!")
	messages := transportManager.GetChannel()
	for message := range messages {
		fmt.Printf(message.String())
	}

}
