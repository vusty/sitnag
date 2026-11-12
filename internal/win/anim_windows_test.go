package win

import (
	"runtime"
	"testing"
	"time"
	"unsafe"
)

var procPeekMessage = user32.NewProc("PeekMessageW")

// TestMannequinClock drives a real window, never shown, through a warm-up
// that is paused and resumed: the frame timer has to tick only while the
// mannequin moves, and its clock has to stand still in between.
func TestMannequinClock(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	w := &Window{choices: make(chan Choice, 1)}
	if err := w.create(); err != nil {
		t.Fatal(err)
	}
	current = w
	defer func() { current = nil }()

	// pump handles messages for d and counts the frame timer's.
	pump := func(d time.Duration) (frames int) {
		var m msg
		for end := time.Now().Add(d); time.Now().Before(end); time.Sleep(5 * time.Millisecond) {
			for {
				if r, _, _ := procPeekMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, 1); r == 0 {
					break
				}
				if m.message == wmTimer {
					frames++
				}
				procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
			}
		}
		return frames
	}
	// show does what Countdown and the posted message do, minus putting the
	// window on screen: coming from a countdown, apply does not centre it.
	show := func(c Countdown) {
		w.pending = view{kind: countdown, cd: c}
		w.apply()
	}

	w.shown = view{kind: countdown, cd: Countdown{Move: 1, Paused: true}}
	show(Countdown{Move: 2})
	if n := pump(400 * time.Millisecond); n < 5 {
		t.Fatalf("%d frames in 400ms of warm-up, want about ten", n)
	}
	if got := w.moveTime(); got < 350*time.Millisecond || got > time.Second {
		t.Fatalf("mannequin %v into its move after 400ms", got)
	}

	show(Countdown{Move: 2, Paused: true})
	held := w.moveTime()
	if n := pump(300 * time.Millisecond); n > 1 {
		t.Fatalf("%d frames while paused, want none", n)
	}
	if got := w.moveTime(); got != held {
		t.Fatalf("mannequin moved on from %v to %v while paused", held, got)
	}

	show(Countdown{Move: 2})
	if n := pump(200 * time.Millisecond); n < 2 {
		t.Fatalf("%d frames after the pause, want the timer back", n)
	}
	if got := w.moveTime(); got < held+150*time.Millisecond {
		t.Fatalf("mannequin at %v after resuming from %v", got, held)
	}

	show(Countdown{Move: 3})
	if got := w.moveTime(); got > 100*time.Millisecond {
		t.Fatalf("a new move started %v in, want from its beginning", got)
	}

	w.pending = view{}
	w.apply()
	if n := pump(200 * time.Millisecond); n > 1 {
		t.Fatalf("%d frames with the window hidden", n)
	}
}
