// Package nbtee ("non-blocking tee") provides an io.WriteCloser that
// writes everything to a set of other io.Writers ("sinks").
//
// Unlike an io.MultiWriter, an nbtee.Writer does not wait for all
// sinks to accept each write.  Instead, it copies writes into an
// internal buffer, and feeds the data to all sinks asynchronously.
//
// A sink that processes Write() calls too slowly misses some writes.
//
// If a sink's Write() method returns an error, it gets closed (if it
// implements io.Closer), and does not receive any more writes.
//
// Sinks can be added and removed while the nbtee.Writer is in use.
//
// The maximum memory used by a Writer is BufsPerSink * number of
// clients * size of []byte slice sent to Write().
//
package nbtee

import (
	"errors"
	"io"
)

type Writer struct {
	// Number of write buffers that can be queued up for each sink
	// before it starts missing writes. Changing this will affect
	// writers added in subsequent Add() calls. Do not change it
	// while an Add() call is in progress.
	//
	// If BufsPerSink > 1, whenever a sink's buffer reaches
	// capacity-1 subsequent writes will be skipped by that sink
	// until its buffer drains to 0.
	BufsPerSink int
	err         error
	cmd         chan interface{}
	done        chan struct{}
}

type cmdAdd *sink
type cmdRemove struct {
	io.Writer
	done chan *sink
}
type cmdWrite []byte
type cmdFlush chan struct{}

var ErrNotFound = errors.New("writer was never added, or was already removed")

// NewWriter returns a new Writer with the given BufsPerSink. The
// returned Writer will not be ready to use until Start() is called.
func NewWriter(bufsPerSink int) *Writer {
	return &Writer{BufsPerSink: bufsPerSink}
}

// Start must be called before Add or Write, and must not be called
// twice. Start returns w, so you can say
//
//   w := NewWriter(8).Start()
func (w *Writer) Start() *Writer {
	w.cmd = make(chan interface{})
	w.done = make(chan struct{})
	go w.run()
	return w
}

// Add a writer. The given writer will get a copy of everything
// written to w. If writer returns an error on any write, it will be
// closed (if it implements io.Closer), and then removed.
//
// If the given writer has already been added (and not removed), Add
// does nothing.
func (w *Writer) Add(writer io.Writer) {
	s := &sink{
		Writer:   writer,
		c:        make(chan []byte, w.BufsPerSink),
		done:     make(chan struct{}),
		flushers: make(chan chan struct{}, 1),
	}
	go s.drain()
	w.cmd <- cmdAdd(s)
}

// RemoveAndClose removes a writer, waits for it to drain any buffered
// data, and (if the writer implements io.Closer) closes it.
func (w *Writer) RemoveAndClose(writer io.Writer) error {
	done := make(chan *sink)
	w.cmd <- cmdRemove{Writer: writer, done: done}
	sink, ok := <-done
	if !ok {
		return ErrNotFound
	}
	<-sink.done
	sink.Flush()
	return sink.Close()
}

// Remove removes a writer. Does not wait for buffered data to drain
// before returning. Returns an io.Closer whose Close() method will
// (if applicable) wait for all buffered data to be written, close the
// original writer, and return any error encountered while writing or
// closing.
//
// Returns ErrNotFound if the writer was not added, or was already
// removed.
func (w *Writer) Remove(writer io.Writer) (io.Closer, error) {
	done := make(chan *sink)
	w.cmd <- cmdRemove{Writer: writer, done: done}
	sink, ok := <-done
	if !ok {
		return nil, ErrNotFound
	}
	return sink, nil
}

// Write implements io.Writer by buffering the given data, and (buffer
// capacity permitting) writing it asynchronously to every sink.
func (w *Writer) Write(buf []byte) (int, error) {
	bufcopy := make([]byte, len(buf))
	copy(bufcopy, buf)
	w.cmd <- cmdWrite(bufcopy)
	return len(buf), nil
}

// Close frees all resources. However, it does not wait for sinks to
// finish draining their buffers, or close them.
//
// Do not call any other method after calling Close.
func (w *Writer) Close() error {
	if w.done == nil {
		// No sinks have been Added yet, so there is nothing
		// to clean up.
		return nil
	}
	close(w.cmd)
	<-w.done
	return w.err
}

// Flush waits for all current sinks to consume (or discard) all data
// from previous calls to Write(). It is safe to call Write from other
// goroutines while waiting for Flush to return.
func (w *Writer) Flush() {
	done := make(chan struct{})
	w.cmd <- cmdFlush(done)
	<-done
}

func (w *Writer) run() {
	sinks := map[interface{}]*sink{}
	for cmd := range w.cmd {
		switch cmd := cmd.(type) {
		case cmdAdd:
			if _, ok := sinks[cmd.Writer]; ok {
				// We already have a sink writing to
				// this Writer. Terminate its
				// goroutine by closing its channel,
				// then forget it.
				close(cmd.c)
			} else {
				sinks[cmd.Writer] = cmd
			}
		case cmdRemove:
			if s, ok := sinks[cmd.Writer]; ok {
				close(s.c)
				delete(sinks, cmd.Writer)
				if cmd.done != nil {
					cmd.done <- s
				}
			}
			if cmd.done != nil {
				close(cmd.done)
			}
		case cmdFlush:
			for _, s := range sinks {
				s.Flush()
			}
			close(cmd)
		case cmdWrite:
			for _, s := range sinks {
				s.Write(cmd)
			}
		}
	}
	for _, s := range sinks {
		close(s.c)
	}
	close(w.done)
}
