package opanic

import (
	_ "embed"
	"testing"
)

//go:embed testdata/sigabrt_sample.txt
var sigabrtSample []byte

func TestScan_DetectsSIGABRT(t *testing.T) {
	snap := Scan(sigabrtSample, nil)
	if snap == nil {
		t.Fatal("panicparse did not detect SIGABRT format")
	}
	if len(snap.Goroutines) == 0 {
		t.Fatal("expected at least one goroutine in SIGABRT snapshot")
	}
}
