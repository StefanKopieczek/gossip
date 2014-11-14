package parser

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/stefankopieczek/gossip/log"
)

// The error returned by the GetNextChunk and GetNextLine methods of Parserbuffer
// when the buffer has ben stopped.
var ERR_BUFFER_STOPPED error = fmt.Errorf("Parser has stopped")

// The number of writes to the buffer that can queue unhandled before
// subsequent writes start to block.
const c_writeBuffSize int = 5

// parserBuffer is a specialized buffer for use in the parser package.
// It is written to via the non-blocking Write.
// It exposes various blocking read methods, which wait until the requested
// data is avaiable, and then return it.
type parserBuffer struct {
	buffer bytes.Buffer

	lineBreaks []int

	dataIn       chan string
	requestsIn   chan dataRequest
	requestQueue []dataRequest

	stop chan bool
}

// Create a new parserBuffer object (see struct comment for object details).
// Note that resources owned by the parserBuffer may not be able to be GCed
// until the Dispose() method is called.
func newParserBuffer() *parserBuffer {
	var pb parserBuffer
	pb.lineBreaks = make([]int, 0)
	pb.requestsIn = make(chan dataRequest, 0)
	pb.requestQueue = make([]dataRequest, 0)
	pb.dataIn = make(chan string, c_writeBuffSize)
	pb.stop = make(chan bool)

	go pb.manage()

	return &pb
}

// Block until the buffer contains at least one CRLF-terminated line.
// Return the line, excluding the terminal CRLF, and delete it from the buffer.
// Returns an error if the parserbuffer has been stopped.
func (pb *parserBuffer) NextLine() (response string, err error) {
	var request lineRequest = make(chan string)

	// Handle the case where pb has been stopped.
	defer func() {
		if r := recover(); r != nil {
			err = ERR_BUFFER_STOPPED
		}
	}()

	pb.requestsIn <- request

	var ok bool
	response, ok = <-request

	if !ok {
		err = ERR_BUFFER_STOPPED
	}

	return
}

// Block until the buffer contains at least n characters.
// Return precisely those n characters, then delete them from the buffer.
func (pb *parserBuffer) NextChunk(n int) (response string, err error) {
	var request chunkRequest = chunkRequest{
		n:        n,
		response: make(chan string),
	}

	// Handle the case where pb has been stopped.
	defer func() {
		if r := recover(); r != nil {
			err = ERR_BUFFER_STOPPED
		}
	}()
	pb.requestsIn <- request

	var ok bool
	response, ok = <-request.response

	if !ok {
		err = ERR_BUFFER_STOPPED
	}

	return
}

// Append the given string to the buffer.
// This method is generally non-blocking, but is not guaranteed to be so depending
// on the relative request and response load.
// Specifically, it may block if a large number of requests are pending, and
// then several writes are made in succession.
func (pb *parserBuffer) Write(s string) {
	pb.dataIn <- s
}

// Stop the parser buffer.
func (pb *parserBuffer) Stop() {
	pb.stop <- true
}

// The main management loop for the buffer.
// Receives incoming requests and new buffer data, and handles the requests as data
// becomes available.
func (pb *parserBuffer) manage() {
	// Inline the function for handling requests, as we need it in a couple of places.
	handleRequests := func() {
	requestLoop:
		for len(pb.requestQueue) > 0 {
			// See if we can respond to any requests.
			switch pb.requestQueue[0].(type) {
			case lineRequest:
				if len(pb.lineBreaks) > 0 {
					// The data in the buffer has at least one CRLF, so we're able to service
					// this request as we do have a complete line.
					breakpoint := pb.lineBreaks[0] + 2
					s := string(pb.buffer.Next(breakpoint))
					s = s[:len(s)-2] // Strip CRLF
					pb.requestQueue[0].(lineRequest) <- s
					pb.lineBreaks = pb.lineBreaks[1:]
					pb.requestQueue = pb.requestQueue[1:]
					for idx := range pb.lineBreaks {
						pb.lineBreaks[idx] -= breakpoint
					}
				} else {
					// Don't service any subsequent requests, as we can't yet process the first
					// one, and they need to be handled in order. Wait for more data and try again.
					break requestLoop
				}
			case chunkRequest:
				chunkReq := pb.requestQueue[0].(chunkRequest)
				if pb.buffer.Len() >= chunkReq.n {
					// We have enough data in the buffer to service the request for chunkReq.n characters.
					chunkReq.response <- string(pb.buffer.Next(chunkReq.n))
					pb.requestQueue = pb.requestQueue[1:]

					// Update the stored line-break indices, discarding any line breaks which were contained
					// within the chunk we just returned.
					discardedLineBreaks := 0
					for idx := range pb.lineBreaks {
						pb.lineBreaks[idx] -= chunkReq.n
						if pb.lineBreaks[idx] < 0 {
							discardedLineBreaks += 1
						}
					}

					if discardedLineBreaks < len(pb.lineBreaks) {
						pb.lineBreaks = pb.lineBreaks[discardedLineBreaks+1:]
					} else {
						pb.lineBreaks = make([]int, 0)
					}
				} else {
					break requestLoop
				}
			}
		}
	}

mainLoop:
	for {
		handleRequests()

		// Now that we've handled all the requests we can, block until we have more data or new requests.
		select {
		case data := <-pb.dataIn:
			bufferEndIdx := pb.buffer.Len()
			pb.buffer.WriteString(data)
			for _, idx := range indexAll(data, "\r\n") {
				pb.lineBreaks = append(pb.lineBreaks, bufferEndIdx+idx)
			}
		case request := <-pb.requestsIn:
			pb.requestQueue = append(pb.requestQueue, request)
		case <-pb.stop:
			// Stop main loop, dispatch all pending requests, and end.
			log.Debug("Parserbuffer %p got the stop signal", pb)
			break mainLoop
		}
	}

	// We've received a stop signal, so stop handling new requests.
	// Close all open request objects with an error.
	close(pb.requestsIn)
	log.Debug("Parserbuffer %p closing outstanding requests", pb)
	for _, request := range pb.requestQueue {
		switch request.(type) {
		case lineRequest:
			close(request.(lineRequest))
		case chunkRequest:
			close(request.(chunkRequest).response)
		}
	}
}

// Generic interface for data requests made to the buffer.
// Requests generally take the form of a chan for the requested data to be sent
// down, as well as additional parameters defining the exact request.
type dataRequest interface{}

// Request for the next CRLF-terminated line in the buffer.
type lineRequest chan string

// Request for the next n bytes in the buffer.
type chunkRequest struct {
	n        int
	response chan string
}

// Utility method.
// Returns a slice containing the indices of all instances of 'target' in 'source'.
// Overlapping instances are considered, e.g. indexAll("banana", "ana") -> [1, 3].
func indexAll(source string, target string) []int {
	indices := make([]int, 0)
	offset := 0
	for offset < len(source) {
		index := strings.Index(source[offset:], target)
		if index == -1 {
			break
		}

		indices = append(indices, offset+index)
		offset += index + 1
	}

	return indices
}
