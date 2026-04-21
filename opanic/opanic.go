// Package opanic detects Go panics and runtime crashes from a worker
// process's stderr and parses them into a structured goroutine snapshot
// via github.com/maruel/panicparse.
package opanic

import (
	"bytes"
	"errors"
	"io"
	"sync"

	"github.com/maruel/panicparse/v2/stack"
)

// Re-exports of the panicparse snapshot types so consumers only need to
// import this package.
type (
	Snapshot  = stack.Snapshot
	Goroutine = stack.Goroutine
	Stack     = stack.Stack
	Signature = stack.Signature
	Call      = stack.Call
	Func      = stack.Func
	Arg       = stack.Arg
	Args      = stack.Args
	Location  = stack.Location
)

// DefaultTailSize is the default number of bytes of stderr retained for
// panic detection when no override is provided.
const DefaultTailSize = 256 * 1024

// TailWriter is a fixed-size ring buffer that retains the most recent N
// bytes written. Write is amortized O(len(p)) with no reallocation after
// the first; Bytes is O(N). Safe for concurrent use.
type TailWriter struct {
	mu    sync.Mutex
	buf   []byte
	size  int
	start int // index of the oldest byte
	count int // bytes currently stored, 0..size
}

// NewTailWriter returns a TailWriter that retains the last `size` bytes.
// A non-positive size yields a no-op writer.
func NewTailWriter(size int) *TailWriter {
	return &TailWriter{size: size}
}

func (t *TailWriter) Write(p []byte) (int, error) {
	n := len(p)
	if n == 0 || t.size <= 0 {
		return n, nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	//only the trailing `size` bytes of p can survive
	if n > t.size {
		p = p[n-t.size:]
	}
	if t.buf == nil {
		t.buf = make([]byte, t.size)
	}
	for _, b := range p {
		head := (t.start + t.count) % t.size
		t.buf[head] = b
		if t.count < t.size {
			t.count++
		} else {
			t.start = (t.start + 1) % t.size
		}
	}
	return n, nil
}

// Bytes returns a copy of the bytes currently buffered, in write order.
func (t *TailWriter) Bytes() []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.count == 0 {
		return nil
	}
	out := make([]byte, t.count)
	if t.start+t.count <= t.size {
		copy(out, t.buf[t.start:t.start+t.count])
		return out
	}
	first := t.size - t.start
	copy(out, t.buf[t.start:])
	copy(out[first:], t.buf[:t.count-first])
	return out
}

// Scan returns a goroutine snapshot if b contains a Go panic or runtime
// crash (e.g. SIGSEGV/SIGABRT with a goroutine dump). If logf is non-nil,
// parse errors are reported through it.
func Scan(b []byte, logf func(string, ...interface{})) *Snapshot {
	if len(b) == 0 {
		return nil
	}
	s, _, err := stack.ScanSnapshot(bytes.NewReader(b), io.Discard, stack.DefaultOpts())
	if err != nil && !errors.Is(err, io.EOF) {
		if logf != nil {
			logf("opanic.Scan: parse error: %s", err)
		}
		return nil
	}
	return s
}
