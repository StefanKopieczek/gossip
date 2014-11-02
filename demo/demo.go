package main

import "github.com/stefankopieczek/gossip/log"

func main() {
	log.SetDefaultLogLevel(log.DEBUG)
	// go receive()
	go send()

	// Block forever.
	c := make(chan bool, 0)
	<-c
}
