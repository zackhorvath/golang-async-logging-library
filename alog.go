// Package alog provides a simple asynchronous logger that will write to provided io.Writers without blocking calling
// goroutines.
package alog

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Alog is a type that defines a logger. It can be used to write log messages synchronously (via the Write method)
// or asynchronously via the channel returned by the MessageChannel accessor.
type Alog struct {
	dest               io.Writer
	m                  *sync.Mutex
	msgCh              chan string
	errorCh            chan error
	shutdownCh         chan struct{}
	shutdownCompleteCh chan struct{}
}

// New creates a new Alog object that writes to the provided io.Writer.
// If nil is provided the output will be directed to os.Stdout.
func New(w io.Writer) *Alog {
	if w == nil {
		w = os.Stdout
	}
	return &Alog{ // it reads better to initialize these structs in a return
		dest:    w,
		msgCh:   make(chan string),
		errorCh: make(chan error),
		m:       &sync.Mutex{}, // this is functionally equiv to a variable mapped to new(sync.Mutex)
	}
}

// Start begins the message loop for the asynchronous logger. It should be initiated as a goroutine to prevent
// the caller from being blocked.
func (al Alog) Start() {
	for { // this is an infinite for loop
		msg := <-al.msgCh     // this reads bytes from the msgCh channel
		go al.write(msg, nil) // this spawns a new goroutine every time it's called
	}
}

func (al Alog) formatMessage(msg string) string {
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	return fmt.Sprintf("[%v] - %v", time.Now().Format("2006-01-02 15:04:05"), msg)
}

func (al Alog) write(msg string, wg *sync.WaitGroup) {
	al.m.Lock()         // this locks the mutex
	defer al.m.Unlock() // a defer statement defers the execution of a fucntion until the surrounding function returns
	_, err := al.dest.Write([]byte(al.formatMessage(msg)))
	if err != nil { // if there's an error, create a goroutine to pipe that error into the errorCh, this prevents deadlocking
		go func(err error) {
			al.errorCh <- err
		}(err)
	}
}

func (al Alog) shutdown() {
}

// MessageChannel returns a channel that accepts messages that should be written to the log.
func (al Alog) MessageChannel() chan<- string { // addded 'chan<-', since msgCh will never send messages to consumers
	return al.msgCh
}

// ErrorChannel returns a channel that will be populated when an error is raised during a write operation.
// This channel should always be monitored in some way to prevent deadlock goroutines from being generated
// when errors occur.
func (al Alog) ErrorChannel() <-chan error { // added '<-chan', since errorCh will only receive messages on this channel
	return al.errorCh
}

// Stop shuts down the logger. It will wait for all pending messages to be written and then return.
// The logger will no longer function after this method has been called.
func (al Alog) Stop() {
}

// Write synchronously sends the message to the log output
func (al Alog) Write(msg string) (int, error) {
	return al.dest.Write([]byte(al.formatMessage(msg)))
}
