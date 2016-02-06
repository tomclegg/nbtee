package nbtee

import (
	"io"
)

type sink struct {
	io.Writer
	c        chan []byte
	done     chan struct{}
	flushers chan chan struct{}
	err      error
}

// Close the sink's writer (if it implements io.Closer) and return the
// first error encountered by this sink's Write or Close.
func (s *sink) Close() error {
	<-s.done
	if w, ok := s.Writer.(io.Closer); ok {
		err := w.Close()
		if s.err == nil {
			s.err = err
		}
		// Ensure we don't close twice
		s.Writer = nil
	}
	return s.err
}

// Flush waits for the sink to drain. It must not be called while
// other goroutines are calling any other methods on this sink. It
// must only be called either before s.c is closed, or after s.done is
// closed.
func (s *sink) Flush() {
	select {
	case <-s.done:
		// closed implies already flushed
	default:
		done := make(chan struct{})
		s.flushers <- done
		s.c <- nil
		<-done
	}
}

// Write some bytes to the sink. If the sink is full, discard the
// bytes. Return as soon as possible regardless of how slowly the sink
// is draining.
func (s *sink) Write(buf []byte) (int, error) {
	select {
	case <-s.done:
		// sink encountered an error, no more data will be
		// written
		return 0, s.err
	default:
	}
	if cap(s.c) > 1 && len(s.c)+1 == cap(s.c) {
		// Sending to the channel now might mean we can't send
		// a nil buffer next time without blocking. So we send
		// a nil buffer now. This notifies the drain()
		// goroutine that it is missing incoming bufs by
		// writing too slowly.
		s.c <- nil
	} else {
		select {
		case s.c <- buf:
		default:
			// channel not ready: this means either cap ==
			// 0 or the last thing we sent was nil.
		}
	}
	return len(buf), nil
}

// Drain the sink by receiving buffers from c and writing them to the
// wrapped io.Writer. If the writer returns an error or c closes,
// close s.done and terminate.
func (s *sink) drain() {
	for buf := range s.c {
		if buf == nil {
			for len(s.c) > 0 {
				<-s.c
			}
		F:
			for {
				select {
				case f := <-s.flushers:
					f <- struct{}{}
				default:
					break F
				}
			}
		} else {
			_, s.err = s.Writer.Write(buf)
			if s.err != nil {
				close(s.done)
				s.Close()
				for _ = range s.c {
				}
				return
			}
		}
	}
	close(s.done)
	close(s.flushers)
}
