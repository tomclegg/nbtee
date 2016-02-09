# nbtee
--
    import "github.com/tomclegg/nbtee"

Package nbtee ("non-blocking tee") provides an io.WriteCloser that writes
everything to a set of other io.Writers ("sinks").

Unlike an io.MultiWriter, an nbtee.Writer does not wait for all sinks to accept
each write. Instead, it copies writes into an internal buffer, and feeds the
data to all sinks asynchronously.

A sink that processes Write() calls too slowly misses some writes.

If a sink's Write() method returns an error, it gets closed (if it implements
io.Closer), and does not receive any more writes.

Sinks can be added and removed while the nbtee.Writer is in use.

The maximum memory used by a Writer is BufsPerSink * number of clients * size of
[]byte slice sent to Write().

## Usage

```go
var ErrNotFound = errors.New("writer was never added, or was already removed")
```

#### type Writer

```go
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
}
```


#### func  NewWriter

```go
func NewWriter(bufsPerSink int) *Writer
```
NewWriter returns a new Writer with the given BufsPerSink. The returned Writer
will not be ready to use until Start() is called.

#### func (*Writer) Add

```go
func (w *Writer) Add(writer io.Writer)
```
Add a writer. The given writer will get a copy of everything written to w. If
writer returns an error on any write, it will be closed (if it implements
io.Closer), and then removed.

If the given writer has already been added (and not removed), Add does nothing.

#### func (*Writer) Close

```go
func (w *Writer) Close() error
```
Close frees all resources. However, it does not wait for sinks to finish
draining their buffers, or close them.

Do not call any other method after calling Close.

#### func (*Writer) Flush

```go
func (w *Writer) Flush()
```
Flush waits for all current sinks to consume (or discard) all data from previous
calls to Write(). It is safe to call Write from other goroutines while waiting
for Flush to return.

#### func (*Writer) Remove

```go
func (w *Writer) Remove(writer io.Writer) (io.Closer, error)
```
Remove removes a writer. Does not wait for buffered data to drain before
returning. Returns an io.Closer whose Close() method will (if applicable) wait
for all buffered data to be written, close the original writer, and return any
error encountered while writing or closing.

Returns ErrNotFound if the writer was not added, or was already removed.

#### func (*Writer) RemoveAndClose

```go
func (w *Writer) RemoveAndClose(writer io.Writer) error
```
RemoveAndClose removes a writer, waits for it to drain any buffered data, and
(if the writer implements io.Closer) closes it.

#### func (*Writer) Start

```go
func (w *Writer) Start() *Writer
```
Start must be called before Add or Write, and must not be called twice. Start
returns w, so you can say

    w := NewWriter(8).Start()

#### func (*Writer) Write

```go
func (w *Writer) Write(buf []byte) (int, error)
```
Write implements io.Writer by buffering the given data, and (buffer capacity
permitting) writing it asynchronously to every sink.
