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
	if ev := c.away(4 * time.Minute); ev != None {
		t.Fatalf("event %v during the break", ev)
	}
	if got := c.s.BreakLeft(); got != time.Minute {
		t.Fatalf("break left %v after 4m away, want 1m", got)
	}
	if c.s.BreakPaused() {
		t.Fatal("paused with nobody at the keyboard")
	}
	// Back at the keyboard the countdown stands still, however long, and the
	// seconds it ran before the input counted as use are handed back.
	if ev := c.work(10 * time.Minute); ev != None {
		t.Fatalf("event %v while using the computer during the break", ev)
	}
	if got := c.s.BreakLeft(); got != time.Minute {
		t.Fatalf("break left %v after working through it, want still 1m", got)
	}
	if !c.s.BreakPaused() {
		t.Fatal("not paused while the keyboard is in use")
	}
	// It resumes Busy after the last input: 5s + 1m more.
	if ev := c.away(time.Minute); ev != None {
		t.Fatalf("done too early: %v", ev)
	}
	if ev := c.away(10 * time.Second); ev != BreakDone {
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

func TestBreakNowWithoutReminder(t *testing.T) {
	c := newClock(t)
	c.work(20 * time.Minute)
	if !c.s.BreakNow(c.now) {
		t.Fatal("break now refused while working")
	}
	if c.s.Mode() != Breaking {
		t.Fatalf("mode %v, want Breaking", c.s.Mode())
	}
	if c.s.BreakNow(c.now) {
		t.Fatal("break now accepted during a break")
	}
	if ev := c.away(5 * time.Minute); ev != BreakDone {
		t.Fatalf("want BreakDone, got %v", ev)
	}
	if ev := c.work(59 * time.Minute); ev != None {
		t.Fatalf("count should start over after an early break, got %v", ev)
	}
	if ev := c.work(2 * time.Minute); ev != Ask {
		t.Fatalf("want Ask an hour after the early break, got %v", ev)
	}
}

func TestBreakNowEndsSnooze(t *testing.T) {
	c := newClock(t)
	c.work(60 * time.Minute)
	c.s.Snooze(c.now, true)
	c.work(5 * time.Minute)
	if !c.s.BreakNow(c.now) {
		t.Fatal("break now refused during a snooze")
	}
	if ev := c.away(5 * time.Minute); ev != BreakDone {
		t.Fatalf("want BreakDone, got %v", ev)
	}
	if !c.s.SnoozedUntil().IsZero() {
		t.Fatalf("snooze still pending until %v after the break", c.s.SnoozedUntil())
	}
	if ev := c.work(59 * time.Minute); ev != None {
		t.Fatalf("the old snooze or count came back: %v at %v", ev, c.at())
	}
}

func TestBreakNowWhileAsking(t *testing.T) {
	c := newClock(t)
	c.work(60 * time.Minute)
	if !c.s.BreakNow(c.now) {
		t.Fatal("break now refused with the reminder up")
	}
	if c.s.StartBreak(c.now) {
		t.Fatal("the window's start button worked during a break")
	}
}

// seconds ticks for d once a second, the way the tool does, with input at
// the given moments, counted from origin.
func (c *clock) seconds(origin time.Time, d time.Duration, inputs ...time.Duration) {
	for end := c.now.Add(d); c.now.Before(end); {
		c.now = c.now.Add(time.Second)
		idle := time.Hour // nothing since long before the click
		for _, in := range inputs {
			if at := origin.Add(in); !at.After(c.now) {
				idle = c.now.Sub(at)
			}
		}
		c.s.Tick(c.now, idle)
	}
}

func every(from, to, by time.Duration) []time.Duration {
	var out []time.Duration
	for at := from; at <= to; at += by {
		out = append(out, at)
	}
	return out
}

func TestStrayTouchDoesNotPauseTheBreak(t *testing.T) {
	c := newClock(t)
	c.work(60 * time.Minute)
	c.s.StartBreak(c.now)
	// The hand leaves the mouse right after the click, the chair bumps the
	// desk on the way up, and something nudges the mouse later.
	ms := time.Millisecond
	touches := []time.Duration{300 * ms, 900 * ms, 2500 * ms, 20 * time.Second, 40 * time.Second}
	for start := c.now; c.now.Sub(start) < time.Minute; {
		c.seconds(start, time.Second, touches...)
		if c.s.BreakPaused() {
			t.Fatalf("a stray touch paused the warm-up at %v", c.now.Sub(start))
		}
	}
	if got := c.s.BreakLeft(); got != 4*time.Minute {
		t.Fatalf("break left %v after a minute with stray touches, want 4m", got)
	}
}

func TestSustainedUsePausesTheBreak(t *testing.T) {
	c := newClock(t)
	c.work(60 * time.Minute)
	c.s.StartBreak(c.now)
	c.seconds(c.now, 30*time.Second)
	// Ten seconds of typing: not use for the first two, paused soon after,
	// and what the countdown ran in the meantime is handed back.
	start := c.now
	typing := every(100*time.Millisecond, 10*time.Second, 200*time.Millisecond)
	c.seconds(start, 2*time.Second, typing...)
	if c.s.BreakPaused() {
		t.Fatal("paused after two seconds of input, before the grace ran out")
	}
	c.seconds(start, 3*time.Second, typing...)
	if !c.s.BreakPaused() {
		t.Fatal("not paused after five seconds of typing")
	}
	c.seconds(start, 5*time.Second, typing...)
	if got := c.s.BreakLeft(); got != 4*time.Minute+30*time.Second {
		t.Fatalf("break left %v after typing, want the 4m30s it had before", got)
	}
	// Five seconds after the last key it goes on.
	c.seconds(start, 4*time.Second, typing...)
	if !c.s.BreakPaused() {
		t.Fatal("resumed four seconds after the last key")
	}
	c.seconds(start, 3*time.Second, typing...)
	if c.s.BreakPaused() || c.s.BreakLeft() >= 4*time.Minute+30*time.Second {
		t.Fatalf("still standing seven seconds after the last key: paused %v, left %v", c.s.BreakPaused(), c.s.BreakLeft())
	}
}

func TestPutOffBringsTheReminderBack(t *testing.T) {
	c := newClock(t)
	// Started from the tray twenty minutes in, and that was a slip.
	c.work(20 * time.Minute)
	c.s.BreakNow(c.now)
	c.away(time.Minute)
	if !c.s.PutOff(c.now) {
		t.Fatal("put-off refused during the warm-up")
	}
	if c.s.PutOff(c.now) {
		t.Fatal("second put-off accepted")
	}
	if c.s.Mode() != Working {
		t.Fatalf("mode %v after the put-off, want Working", c.s.Mode())
	}
	// The reminder is back after the short snooze, an hour at the computer
	// or not.
	if ev := c.work(5*time.Minute - step); ev != None {
		t.Fatalf("event %v before the snooze ran out, at %v", ev, c.at())
	}
	if ev := c.work(step); ev != Ask {
		t.Fatalf("want Ask when the snooze runs out, got %v", ev)
	}
	// Snoozing it does not make it go away either.
	c.s.Snooze(c.now, false)
	if ev := c.work(5 * time.Minute); ev != Ask {
		t.Fatalf("want Ask again after a snooze, got %v", ev)
	}
	// The warm-up runs from the top, and once done nothing is owed.
	if !c.s.StartBreak(c.now) {
		t.Fatal("break refused while asking")
	}
	if got := c.s.BreakLeft(); got != 5*time.Minute {
		t.Fatalf("break left %v after a put-off, want the whole 5m", got)
	}
	if ev := c.away(5 * time.Minute); ev != BreakDone {
		t.Fatalf("want BreakDone, got %v", ev)
	}
	if ev := c.work(59 * time.Minute); ev != None {
		t.Fatalf("count should start over after the break, got %v", ev)
	}
}

func TestWalkingAwaySettlesAPutOffWarmUp(t *testing.T) {
	c := newClock(t)
	c.work(60 * time.Minute)
	c.s.StartBreak(c.now)
	c.s.PutOff(c.now)
	if c.s.PutOff(c.now) || c.s.Snooze(c.now, false) {
		t.Fatal("nothing is on screen to put off or snooze")
	}
	// Ten minutes away is the break, asked for or not.
	c.away(11 * time.Minute)
	if c.s.Mode() != Working {
		t.Fatalf("mode %v after walking away, want Working", c.s.Mode())
	}
	if ev := c.work(59 * time.Minute); ev != None {
		t.Fatalf("the put-off warm-up is still owed after a real break: %v at %v", ev, c.at())
	}
}
