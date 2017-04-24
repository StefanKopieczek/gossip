package utils

import "github.com/remodoy/gossip/log"

// The buffer size of the primitive input and output chans.
const c_ELASTIC_CHANSIZE = 3

// A dynamic channel that does not block on send, but has an unlimited buffer capacity.
// ElasticChan uses a dynamic slice to buffer signals received on the input channel until
// the output channel is ready to process them.
type ElasticChan struct {
	In      chan interface{}
	Out     chan interface{}
	buffer  []interface{}
	stopped bool
}

// Initialise the Elastic channel, and start the management goroutine.
func (c *ElasticChan) Init() {
	c.In = make(chan interface{}, c_ELASTIC_CHANSIZE)
	c.Out = make(chan interface{}, c_ELASTIC_CHANSIZE)
	c.buffer = make([]interface{}, 0)

	go c.manage()
}

// Poll for input from one end of the channel and add it to the buffer.
// Also poll sending buffered signals out over the output chan.
func (c *ElasticChan) manage() {
	for {
		if len(c.buffer) > 0 {
			// The buffer has something in it, so try to send as well as
			// receive.
			// (Receive first in order to minimize blocked Send() calls).
			select {
			case in, ok := <-c.In:
				if !ok {
					log.Debug("Chan %p will dispose", c)
					break
				}
				log.Debug("Chan %p gets '%v'", c, in)
				c.buffer = append(c.buffer, in)
			case c.Out <- c.buffer[0]:
				log.Debug("Chan %p sends '%v'", c, c.buffer[0])
				c.buffer = c.buffer[1:]
			}
		} else {
			// The buffer is empty, so there's nothing to send.
			// Just wait to receive.
			in, ok := <-c.In
			if !ok {
				log.Debug("Chan %p will dispose", c)
				break
			}
			log.Debug("Chan %p gets '%v'", c, in)
			c.buffer = append(c.buffer, in)
		}
	}

	c.dispose()
}

func (c *ElasticChan) dispose() {
	log.Debug("Chan %p disposing...", c)
	for len(c.buffer) > 0 {
		c.Out <- c.buffer[0]
		c.buffer = c.buffer[1:]
	}
	log.Debug("Chan %p disposed", c)
}
