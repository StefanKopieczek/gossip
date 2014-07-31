package utils

import "github.com/stefankopieczek/gossip/log"

// The buffer size of the primitive input and output chans.
const c_ELASTIC_CHANSIZE = 3

// A dynamic channel that does not block on send, but has an unlimited buffer capacity.
// ElasticChannel uses a dynamic slice to buffer signals received on the input channel until
// the output channel is ready to process them.
type ElasticChannel struct {
    In chan interface{}
    Out chan interface{}
    buffer []interface{}
    stopped bool
}

// Initialise the Elastic channel, and start the management goroutine.
func (c *ElasticChannel) Init() {
    log.Debug("New ElasticChannel %p created", c)
    c.In = make(chan interface{}, c_ELASTIC_CHANSIZE)
    c.Out = make(chan interface{}, c_ELASTIC_CHANSIZE)
    c.buffer = make([]interface{}, 0)

    go c.manage()
}

// Poll for input from one end of the channel and add it to the buffer.
// Also poll sending buffered signals out over the output chan.
func (c *ElasticChannel) manage() {
    for {
        if len(c.buffer) > 0 {
            // The buffer has something in it, so try to send as well as
            // receive.
            // (Receive first in order to minimize blocked Send() calls).
            log.Debug("ElasticChannel %p waiting to send/recv", c)
            select {
            case in, ok := <- c.In:
                if !ok {
                    log.Debug("ElasticChannel %p aborts on input channel close", c)
                    break
                }
                log.Debug("ElasticChannel %p receives %v", c, in)
                c.buffer = append(c.buffer, in)
            case c.Out <- c.buffer[0]:
                log.Debug("ElasticChannel %p sends %v", c, c.buffer[0])
                c.buffer = c.buffer[1:]
            }
        } else {
            // The buffer is empty, so there's nothing to send.
            // Just wait to receive.
            log.Debug("ElasticChannel %p waiting to receive", c)
            in, ok := <- c.In
            if !ok {
                log.Debug("ElasticChannel %p aborts on input channel close", c)
                break
            }
            c.buffer = append(c.buffer, in)
            log.Debug("ElasticChannel %p receives %v", c, c.buffer[len(c.buffer)-1])
        }
    }

    c.dispose()
}

func (c *ElasticChannel) dispose() {
    log.Debug("ElasticChannel %p is disposing - %d items still to send", c, len(c.buffer))
    for len(c.buffer) > 0 {
        c.Out <- c.buffer[0]
        c.buffer = c.buffer[1:]
        log.Debug("ElasticChannel %p dispatches an item - %d left until disposed", c, len(c.buffer))
    }
}
