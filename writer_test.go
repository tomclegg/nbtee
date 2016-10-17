package nbtee

import (
	"bytes"
	"gopkg.in/check.v1"
	"log"
	"testing"
)

func Test(t *testing.T) { check.TestingT(t) }

type Suite struct{}

var _ = check.Suite(&Suite{})

func (s *Suite) TestCloseUnused(c *check.C) {
	w := NewWriter(4)
	c.Check(w.Close(), check.IsNil)
}

func (s *Suite) TestAddRemove(c *check.C) {
	b := &bytes.Buffer{}
	w := NewWriter(4)
	w.Add(b)
	w.Write([]byte{1,2,3})
	w.Flush()
	w.Remove(b)
	w.Write([]byte{4,5,6})
	w.Flush()
	w.Add(b)
	w.Write([]byte{7,8,9})
	w.Flush()
	c.Check(w.Close(), check.IsNil)
	c.Check(b.Bytes(), check.DeepEquals, []byte{1,2,3,7,8,9})
}

func (s *Suite) TestRemoveAndClose(c *check.C) {
	b := &bytes.Buffer{}
	w := NewWriter(4)
	w.Add(b)
	w.Write([]byte{1,2,3})
	w.RemoveAndClose(b)
	w.Write([]byte{4,5,6})
	c.Check(w.Close(), check.IsNil)
	c.Check(b.Bytes(), check.DeepEquals, []byte{1,2,3})
}

func (s *Suite) Test1Kx1K(c *check.C) {
	n := 1000
	w := NewWriter(n)
	bufs := make([]bytes.Buffer, n)
	for i := range bufs {
		w.Add(&bufs[i])
		w.Write([]byte{1})
	}
	w.Flush()
	for i, b := range bufs {
		w.RemoveAndClose(&bufs[i])
		c.Check(len(b.Bytes()), check.Equals, n-i)
	}
	w.Close()
}

func (s *Suite) TestSmallBufLen(c *check.C) {
	for bufLen := range []int{0, 1, 2, 3} {
		n := 1000
		w := NewWriter(bufLen)
		bufs := make([]bytes.Buffer, n)
		for i := range bufs {
			w.Add(&bufs[i])
			w.Write([]byte{1})
		}
		w.Flush()
		for i, b := range bufs {
			w.RemoveAndClose(&bufs[i])
			c.Check(len(b.Bytes()) > 0, check.Equals, true)
		}
		w.Close()
	}
}

// Adding a writer that has already been added is a no-op.
func (s *Suite) TestSameWriterAddedTwice(c *check.C) {
	b := &bytes.Buffer{}
	w := NewWriter(4)
	w.Add(b)
	w.Add(b)
	w.Write([]byte{1,2,3})
	w.Flush()
	w.Remove(b)
	w.Write([]byte{4,5,6})
	w.Flush()
	w.Add(b)
	w.Write([]byte{7,8,9})
	w.Flush()
	c.Check(w.Close(), check.IsNil)
	c.Check(b.Bytes(), check.DeepEquals, []byte{1,2,3,7,8,9})
}

func (s *Suite) TestCloseOnSinkError(c *check.C) {
	// TODO
}

func (s *Suite) TestCloseOnRemove(c *check.C) {
	// TODO
}

func (s *Suite) TestNoCloseIfNotCloser(c *check.C) {
	// TODO
}

func ExampleWriter_Remove() {
	w := NewWriter(5)
	b := &bytes.Buffer{}
	w.Add(b)
	closer, err := w.Remove(b)
	if err != nil {
		panic(err)
	}
	err = closer.Close()
	if err != nil {
		log.Print("w encountered an error writing to b:", err)
	}
	w.Close()
}

func ExampleWriter_RemoveAndClose() {
	w := NewWriter(5)
	b := &bytes.Buffer{}
	w.Add(b)
	err := w.RemoveAndClose(b)
	if err != nil {
		log.Print("w encountered an error writing to b:", err)
	}
	w.Close()
}
