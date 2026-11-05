// Package sit decides when it is time to get up. It knows nothing about
// Windows: the caller feeds it the clock and how long the keyboard and mouse
// have been untouched, and acts on what comes back.
package sit

import "time"

// Config holds every duration the reminder runs on.
type Config struct {
	// RemindAfter is how long at the computer before the first reminder.
	RemindAfter time.Duration
	// HardLimit is the longest stretch the long snooze can reach. Past it only
	// the short snooze is offered, and a long snooze never ends later than it.
	HardLimit time.Duration
	// Away is how long without input counts as having left the computer. That
	// is a break in its own right and starts the count over.
	Away time.Duration
	// Break is the length of the warm-up countdown.
	Break time.Duration
	// Busy is how recent an input has to be, during the warm-up, to count as
	// using the computer. The countdown stands still until there has been no
	// input for this long.
	Busy time.Duration
	// ShortSnooze and LongSnooze are the two "later" buttons.
	ShortSnooze, LongSnooze time.Duration
}

// DefaultConfig is a reminder every hour that cannot be put off past ninety
// minutes by more than five at a time.
//
// Away is ten minutes, not five: the bathroom and a glass of water take five,
// and that is not a warm-up.
func DefaultConfig() Config {
	return Config{
		RemindAfter: 60 * time.Minute,
		HardLimit:   90 * time.Minute,
		Away:        10 * time.Minute,
		Break:       5 * time.Minute,
		Busy:        5 * time.Second,
		ShortSnooze: 5 * time.Minute,
		LongSnooze:  30 * time.Minute,
	}
}

// Mode is what the tool is doing right now.
type Mode int

const (
	// Working counts time at the computer.
	Working Mode = iota
	// Asking has the reminder on screen, waiting for a button.
	Asking
	// Breaking runs the warm-up countdown.
	Breaking
)

// Event tells the caller what changed on a tick.
type Event int

const (
	// None means nothing to show or hide.
	None Event = iota
	// Ask means the reminder is due.
	Ask
	// Left means the user walked away while the reminder was up. That was the
	// break, so the reminder goes away and the count starts over.
	Left
	// BreakDone means the countdown ran out.
	BreakDone
)

// State is the reminder's memory between ticks. The zero value is not usable;
// call New.
type State struct {
	cfg     Config
	mode    Mode
	since   time.Time // start of the current stretch at the computer; zero while away
	snoozed time.Time // no reminder before this
	last    time.Time // previous tick, to notice that the machine slept

	// The warm-up counts down only time spent away from the keyboard and
	// mouse, so it keeps what is left rather than an end time.
	breakLeft    time.Duration
	breakFrom    time.Time // the click that started it; input up to here is not "using"
	breakCounted time.Time // the countdown has been settled up to here
	busy         bool      // input during the warm-up, within Busy
}

// New starts counting from the first tick.
func New(cfg Config) *State {
	return &State{cfg: cfg}
}

// Tick advances the clock. idle is how long since the last keyboard or mouse
// input. Times must be wall-clock (time.Round(0)): a monotonic reading may not
// advance while the machine sleeps, and sleep has to count as being away.
func (s *State) Tick(now time.Time, idle time.Duration) Event {
	var gap time.Duration
	if !s.last.IsZero() {
		gap = now.Sub(s.last)
	}
	s.last = now

	if s.mode == Breaking {
		s.countBreak(now, idle)
		if s.breakLeft > 0 {
			return None
		}
		s.mode = Working
		s.busy = false
		s.restart()
		return BreakDone
	}

	// A gap this long between ticks means the machine slept, and nobody sits
	// at a sleeping computer.
	if idle >= s.cfg.Away || gap >= s.cfg.Away {
		asking := s.mode == Asking
		s.mode = Working
		s.restart()
		if asking {
			return Left
		}
		return None
	}

	if s.since.IsZero() {
		s.since = now.Add(-idle)
	}
	if s.mode == Working && s.Sitting(now) >= s.cfg.RemindAfter && !now.Before(s.snoozed) {
		s.mode = Asking
		return Ask
	}
	return None
}

// StartBreak begins the countdown. It does nothing unless the reminder is up,
// so a second click on the same button is harmless.
func (s *State) StartBreak(now time.Time) bool {
	if s.mode != Asking {
		return false
	}
	s.mode = Breaking
	s.breakLeft = s.cfg.Break
	s.breakFrom = now
	s.breakCounted = now
	s.busy = false
	return true
}

// countBreak takes off the countdown the part of the time since the last tick
// that had no input for at least Busy. A warm-up started and then spent at the
// keyboard is not a warm-up, so while the keyboard or mouse is in use the
// countdown stands still. The click on the start button does not count as use.
func (s *State) countBreak(now time.Time, idle time.Duration) {
	elapsed := now.Sub(s.breakCounted)
	s.breakCounted = now
	if elapsed <= 0 {
		return
	}
	lastInput := now.Add(-idle)
	quiet := elapsed
	if lastInput.After(s.breakFrom) {
		// The countdown runs from Busy after the last input.
		quiet = min(elapsed, max(0, idle-s.cfg.Busy))
	}
	s.busy = lastInput.After(s.breakFrom) && idle < s.cfg.Busy
	s.breakLeft = max(0, s.breakLeft-quiet)
}

// Snooze hides the reminder for the short or the long delay. The long one is
// honoured only before HardLimit and is cut short so it ends there.
func (s *State) Snooze(now time.Time, long bool) bool {
	if s.mode != Asking {
		return false
	}
	until := now.Add(s.cfg.ShortSnooze)
	if long && s.LongAllowed(now) {
		until = now.Add(s.cfg.LongSnooze)
		if limit := s.since.Add(s.cfg.HardLimit); until.After(limit) {
			until = limit
		}
	}
	s.snoozed = until
	s.mode = Working
	return true
}

// Mode reports what the tool is doing.
func (s *State) Mode() Mode { return s.mode }

// Sitting is how long the current stretch at the computer has lasted, zero
// while away.
func (s *State) Sitting(now time.Time) time.Duration {
	if s.since.IsZero() {
		return 0
	}
	return now.Sub(s.since)
}

// LongAllowed says whether the long snooze is still on offer.
func (s *State) LongAllowed(now time.Time) bool {
	return s.Sitting(now) < s.cfg.HardLimit
}

// BreakLeft is what remains of the countdown, as of the last tick.
func (s *State) BreakLeft() time.Duration {
	return s.breakLeft
}

// BreakPaused says whether the warm-up countdown stands still because the
// keyboard or mouse is in use.
func (s *State) BreakPaused() bool {
	return s.mode == Breaking && s.busy
}

// SnoozedUntil is when a snoozed reminder comes back, zero if none is pending.
func (s *State) SnoozedUntil() time.Time { return s.snoozed }

func (s *State) restart() {
	s.since = time.Time{}
	s.snoozed = time.Time{}
}
