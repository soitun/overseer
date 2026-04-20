package opanic

import (
	"strings"
	"testing"
)

const samplePanic = `panic: something went wrong

goroutine 1 [running]:
main.crash(...)
	/home/user/main.go:10
main.main()
	/home/user/main.go:6 +0x25
exit status 2
`

func TestTailWriter_KeepsTail(t *testing.T) {
	w := NewTailWriter(10)
	w.Write([]byte("hello world, this is a long string"))
	got := string(w.Bytes())
	if got != "ong string" {
		t.Fatalf("unexpected tail %q", got)
	}
}

func TestTailWriter_BelowCapacity(t *testing.T) {
	w := NewTailWriter(64)
	w.Write([]byte("short"))
	if got := string(w.Bytes()); got != "short" {
		t.Fatalf("unexpected bytes %q", got)
	}
}

func TestTailWriter_MultipleWrites(t *testing.T) {
	w := NewTailWriter(5)
	w.Write([]byte("abc"))
	w.Write([]byte("defgh"))
	if got := string(w.Bytes()); got != "defgh" {
		t.Fatalf("unexpected bytes %q", got)
	}
	w.Write([]byte("ij"))
	if got := string(w.Bytes()); got != "fghij" {
		t.Fatalf("unexpected bytes %q", got)
	}
}

func TestScan_Detects(t *testing.T) {
	snap := Scan([]byte(samplePanic), nil)
	if snap == nil {
		t.Fatal("expected snapshot, got nil")
	}
	if len(snap.Goroutines) == 0 {
		t.Fatal("expected at least one goroutine")
	}
}

func TestScan_NoPanic(t *testing.T) {
	snap := Scan([]byte("just some log output\nno panic here\n"), nil)
	if snap != nil {
		t.Fatalf("expected nil, got %d goroutines", len(snap.Goroutines))
	}
}

func TestScan_EmptyInput(t *testing.T) {
	if Scan(nil, nil) != nil {
		t.Fatal("expected nil for nil input")
	}
	if Scan([]byte{}, nil) != nil {
		t.Fatal("expected nil for empty input")
	}
}

func TestScan_TruncatedTail(t *testing.T) {
	prefix := strings.Repeat("log line\n", 100)
	snap := Scan([]byte(prefix+samplePanic), nil)
	if snap == nil {
		t.Fatal("expected snapshot even with log prefix")
	}
}
