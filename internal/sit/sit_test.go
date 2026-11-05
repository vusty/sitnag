package sit

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)

// clock drives a State through simulated time, one tick every 10 seconds.
type clock struct {
	t   *testing.T
	s   *State
	now time.Time
}

// newClock ticks once at t0, the way the tool ticks right at startup.
func newClock(t *testing.T) *clock {
	c := &clock{t: t, s: New(DefaultConfig()), now: t0}
	c.s.Tick(t0, 0)
	return c
}

const step = 10 * time.Second

// work ticks for d with constant input and returns the first event seen.
func (c *clock) work(d time.Duration) Event {
	return c.run(d, func(time.Duration) time.Duration { return 0 })
}

// away ticks for d with no input at all.
func (c *clock) away(d time.Duration) Event {
	return c.run(d, func(elapsed time.Duration) time.Duration { return elapsed })
}

func (c *clock) run(d time.Duration, idle func(elapsed time.Duration) time.Duration) Event {
	first := None
	for elapsed := step; elapsed <= d; elapsed += step {
		c.now = c.now.Add(step)
		if ev := c.s.Tick(c.now, idle(elapsed)); ev != None && first == None {
			first = ev
		}
	}
	return first
}

// at returns minutes since start, for messages.
func (c *clock) at() time.Duration { return c.now.Sub(t0) }

func TestAsksAfterAnHour(t *testing.T) {
	c := newClock(t)
	if ev := c.work(59 * time.Minute); ev != None {
		t.Fatalf("event %v before the hour, at %v", ev, c.at())
	}
	if ev := c.work(time.Minute); ev != Ask {
		t.Fatalf("want Ask at the hour, got %v", ev)
	}
	if c.s.Mode() != Asking {
		t.Fatalf("mode %v, want Asking", c.s.Mode())
	}
}

func TestShortPauseKeepsCounting(t *testing.T) {
	c := newClock(t)
	c.work(40 * time.Minute)
	// The bathroom and a glass of water: not a warm-up.
	c.away(9 * time.Minute)
	if ev := c.work(11 * time.Minute); ev != Ask {
		t.Fatalf("a 9-minute pause should not reset the count, got %v at %v", ev, c.at())
	}
}

func TestWalkingAwayIsABreak(t *testing.T) {
	c := newClock(t)
	c.work(50 * time.Minute)
	c.away(11 * time.Minute)
	if got := c.s.Sitting(c.now); got != 0 {
		t.Fatalf("sitting %v while away, want 0", got)
	}
	// The count restarts on the first tick with input, one step in.
	if ev := c.work(60 * time.Minute); ev != None {
		t.Fatalf("count should start over after leaving, got %v", ev)
	}
	if ev := c.work(step); ev != Ask {
		t.Fatalf("want Ask an hour after coming back, got %v", ev)
	}
}

func TestSleepIsABreak(t *testing.T) {
	c := newClock(t)
	c.work(50 * time.Minute)
	// The ticker does not run while the machine sleeps, and the key press
	// that wakes it makes the first idle reading small.
	c.now = c.now.Add(20 * time.Minute)
	c.s.Tick(c.now, 0)
	if got := c.s.Sitting(c.now); got != 0 {
		t.Fatalf("sitting %v after sleep, want 0", got)
	}
}

func TestShortSnoozeComesBack(t *testing.T) {
	c := newClock(t)
	c.work(60 * time.Minute)
	if !c.s.Snooze(c.now, false) {
		t.Fatal("snooze refused while asking")
	}
	if ev := c.work(4*time.Minute + 50*time.Second); ev != None {
		t.Fatalf("back too early: %v at %v", ev, c.at())
	}
	if ev := c.work(10 * time.Second); ev != Ask {
		t.Fatalf("want Ask five minutes later, got %v", ev)
	}
}

func TestLongSnoozeUntilHardLimit(t *testing.T) {
	c := newClock(t)
	c.work(60 * time.Minute)
	if !c.s.LongAllowed(c.now) {
		t.Fatal("long snooze should be on offer at the hour")
	}
	c.s.Snooze(c.now, true)
	if ev := c.work(30 * time.Minute); ev != Ask {
		t.Fatalf("want Ask at 90 minutes, got %v", ev)
	}
	if c.s.LongAllowed(c.now) {
		t.Fatalf("long snooze still on offer at %v", c.at())
	}
	// Asking for the long one anyway gives the short one.
	c.s.Snooze(c.now, true)
	if got, want := c.s.SnoozedUntil(), c.now.Add(5*time.Minute); !got.Equal(want) {
		t.Fatalf("snoozed until %v, want %v", got, want)
	}
}

func TestLongSnoozeIsCutAtHardLimit(t *testing.T) {
	c := newClock(t)
	c.work(60 * time.Minute)
	c.s.Snooze(c.now, false)
	c.work(5 * time.Minute) // asks again at 65
	c.s.Snooze(c.now, true)
	if got, want := c.s.SnoozedUntil(), t0.Add(90*time.Minute); !got.Equal(want) {
		t.Fatalf("long snooze at 65 ends %v, want the 90-minute mark %v", got, want)
	}
}

func TestLeavingWhileAsked(t *testing.T) {
	c := newClock(t)
	c.work(60 * time.Minute)
	if ev := c.away(10 * time.Minute); ev != Left {
		t.Fatalf("want Left, got %v", ev)
	}
	if c.s.Mode() != Working || c.s.Sitting(c.now) != 0 {
		t.Fatalf("mode %v sitting %v, want a fresh count", c.s.Mode(), c.s.Sitting(c.now))
	}
}

func TestBreak(t *testing.T) {
	c := newClock(t)
	c.work(60 * time.Minute)
	if !c.s.StartBreak(c.now) {
		t.Fatal("break refused while asking")
	}
	if c.s.StartBreak(c.now) {
		t.Fatal("second start accepted")
	}
	if got := c.s.BreakLeft(); got != 5*time.Minute {
		t.Fatalf("break left %v, want 5m", got)
	}
	// The click that started it is not use: away from the desk the countdown
	// runs at once.
	if ev := c.away(4*time.Minute + 50*time.Second); ev != None {
		t.Fatalf("event %v during the break", ev)
	}
	if got := c.s.BreakLeft(); got != 10*time.Second {
		t.Fatalf("break left %v after 4m50s away, want 10s", got)
	}
	if c.s.BreakPaused() {
		t.Fatal("paused with nobody at the keyboard")
	}
	// Back at the keyboard the countdown stands still, however long.
	if ev := c.work(10 * time.Minute); ev != None {
		t.Fatalf("event %v while using the computer during the break", ev)
	}
	if got := c.s.BreakLeft(); got != 10*time.Second {
		t.Fatalf("break left %v after working through it, want still 10s", got)
	}
	if !c.s.BreakPaused() {
		t.Fatal("not paused while the keyboard is in use")
	}
	// It resumes Busy after the last input: 5s + 10s more.
	if ev := c.away(10 * time.Second); ev != None {
		t.Fatalf("done too early: %v", ev)
	}
	if ev := c.away(20 * time.Second); ev != BreakDone {
		t.Fatalf("want BreakDone, got %v", ev)
	}
	if ev := c.work(59 * time.Minute); ev != None {
		t.Fatalf("count should start over after the break, got %v", ev)
	}
}

func TestBreakWorkedThroughNeverEnds(t *testing.T) {
	c := newClock(t)
	c.work(60 * time.Minute)
	c.s.StartBreak(c.now)
	// Pressing the button and carrying on typing is the trick this closes.
	if ev := c.work(30 * time.Minute); ev != None {
		t.Fatalf("event %v while typing through the warm-up", ev)
	}
	if got := c.s.BreakLeft(); got != 5*time.Minute {
		t.Fatalf("break left %v, want the full 5m", got)
	}
}

func TestButtonsIgnoredWhenNotAsking(t *testing.T) {
	c := newClock(t)
	c.work(10 * time.Minute)
	if c.s.Snooze(c.now, false) || c.s.StartBreak(c.now) {
		t.Fatal("a button worked with no reminder on screen")
	}
}
