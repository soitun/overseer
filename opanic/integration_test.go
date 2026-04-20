package opanic

import (
	"io"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// buildPanicker compiles testdata/panicker into the test's temp dir and
// returns the resulting binary path.
func buildPanicker(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "panicker")
	cmd := exec.Command("go", "build", "-o", out, "./testdata/panicker")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build panicker failed: %s\n%s", err, output)
	}
	return out
}

// Exercises the end-to-end pipeline: a real child process panics, its
// stderr flows through TailWriter, and Scan extracts a snapshot.
func TestScan_FromPanickingChild(t *testing.T) {
	var fired atomic.Bool
	var gotSnap atomic.Pointer[string]
	onPanic := func(s *Snapshot) {
		fired.Store(true)
		if len(s.Goroutines) > 0 && len(s.Goroutines[0].Stack.Calls) > 0 {
			top := s.Goroutines[0].Stack.Calls[0].Func.Complete
			gotSnap.Store(&top)
		}
	}
	cmd := exec.Command(buildPanicker(t))
	tail := NewTailWriter(DefaultTailSize)
	cmd.Stdout = io.Discard
	cmd.Stderr = tail
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("panicker did not exit in time")
	}
	if snap := Scan(tail.Bytes(), nil); snap != nil {
		onPanic(snap)
	}
	if !fired.Load() {
		t.Fatalf("Scan did not detect a panic; stderr tail:\n%s", tail.Bytes())
	}
	if top := gotSnap.Load(); top != nil {
		t.Logf("top frame: %s", *top)
	}
}
