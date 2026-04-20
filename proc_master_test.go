package overseer

import (
	"sync"
	"testing"
)

func TestTryRestart_ShouldRestartNil_TriggersRestart(t *testing.T) {
	mp := &master{Config: &Config{}}
	// slaveCmd is nil so triggerRestart returns immediately; we just verify
	// the tryRestart path doesn't set pendingRestart.
	mp.tryRestart()
	if mp.pendingRestart {
		t.Fatal("expected pendingRestart=false when ShouldRestart is nil")
	}
}

func TestTryRestart_ShouldRestartFalse_Defers(t *testing.T) {
	mp := &master{Config: &Config{
		ShouldRestart: func() bool { return false },
	}}
	mp.tryRestart()
	if !mp.pendingRestart {
		t.Fatal("expected pendingRestart=true when ShouldRestart returns false")
	}
}

func TestTryRestart_ShouldRestartTrue_ClearsPending(t *testing.T) {
	mp := &master{Config: &Config{
		ShouldRestart: func() bool { return true },
	}}
	mp.pendingRestart = true
	mp.tryRestart()
	if mp.pendingRestart {
		t.Fatal("expected pendingRestart=false after ShouldRestart returns true")
	}
}

func TestTryRestart_TransitionFromFalseToTrue(t *testing.T) {
	var allow bool
	mp := &master{Config: &Config{
		ShouldRestart: func() bool { return allow },
	}}
	mp.tryRestart()
	if !mp.pendingRestart {
		t.Fatal("expected pendingRestart=true on first call (allow=false)")
	}
	allow = true
	mp.tryRestart()
	if mp.pendingRestart {
		t.Fatal("expected pendingRestart=false on second call (allow=true)")
	}
}

// Exercises the mutex-guarded state transitions from many goroutines so
// `go test -race` catches any regression in pendingRestart/restarting.
func TestRestart_ConcurrentRaceFree(t *testing.T) {
	var allow bool
	var mu sync.Mutex
	mp := &master{Config: &Config{
		ShouldRestart: func() bool { mu.Lock(); v := allow; mu.Unlock(); return v },
	}}
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); mp.tryRestart() }()
		go func() { defer wg.Done(); mp.triggerRestart() }()
		if i == 100 {
			mu.Lock()
			allow = true
			mu.Unlock()
		}
	}
	wg.Wait()
}

// triggerRestart is the manual path; it must bypass ShouldRestart and
// must never leave pendingRestart set.
func TestTriggerRestart_BypassesShouldRestart(t *testing.T) {
	var called bool
	mp := &master{Config: &Config{
		ShouldRestart: func() bool { called = true; return false },
	}}
	mp.pendingRestart = true
	mp.triggerRestart() //slaveCmd nil so no actual restart work happens
	if called {
		t.Fatal("ShouldRestart must not be consulted by the manual path")
	}
	if mp.pendingRestart {
		t.Fatal("triggerRestart must clear pendingRestart")
	}
}
