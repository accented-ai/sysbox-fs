package nsenter

import (
	"testing"
	"time"
)

// TestReaperDoesNotBlockNewNsenter verifies that signaling the zombie reaper
// while an nsenter is active does not prevent a new nsenter from starting.
//
// This is a regression test for a deadlock caused by Go's sync.RWMutex
// writer-preference: when the reaper's Lock() was pending (waiting for an
// active nsenter's RLock to be released), all new RLock() callers — including
// FUSE handlers that needed to start their own nsenter — were blocked. If the
// active nsenter depended on one of those FUSE handlers to complete, the
// system deadlocked.
//
// The fix replaced Lock() with TryLock() in the reaper goroutine so it never
// registers a pending writer.
func TestReaperDoesNotBlockNewNsenter(t *testing.T) {
	zr := newZombieReaper()

	// Simulate an active nsenter holding the read lock.
	zr.nsenterStarted()

	// Signal the reaper multiple times to ensure its goroutine reaches a lock
	// attempt while the first reader is active.
	for i := 0; i < 5; i++ {
		zr.nsenterReapReq()
		time.Sleep(500 * time.Millisecond)
	}

	// Try to start a second nsenter from a new goroutine. This calls RLock().
	// If the reaper registered a pending writer via Lock(), this blocks
	// indefinitely due to writer-preference.
	done := make(chan struct{})
	go func() {
		zr.nsenterStarted()
		zr.nsenterEnded()
		close(done)
	}()

	select {
	case <-done:
		// New nsenter started successfully — no writer-preference deadlock.
	case <-time.After(3 * time.Second):
		t.Fatal("nsenterStarted() blocked: reaper's pending write lock is starving new readers")
	}

	zr.nsenterEnded()
}
