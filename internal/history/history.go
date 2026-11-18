// Package history keeps a record of what happened, one JSON object per line,
// to build statistics from. The text log is for reading; this is for counting,
// so durations are plain seconds and nothing depends on the interface language.
package history

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Kind names an event in the file.
type Kind string

const (
	// Start and Quit bracket the time the tool was running: outside them
	// nothing was counted.
	Start Kind = "start"
	Quit  Kind = "quit"
	// Remind is the reminder coming up, with Sat.
	Remind Kind = "remind"
	// Snooze hides the reminder for Len, with Sat.
	Snooze Kind = "snooze"
	// Left is walking away with the reminder up, with Sat. It counts as the
	// break.
	Left Kind = "left"
	// Away is a break taken without a reminder: no input long enough, a locked
	// screen or sleep, with Sat.
	Away Kind = "away"
	// WarmupStart begins the countdown, with Sat and From.
	WarmupStart Kind = "warmup_start"
	// WarmupPause and WarmupResume are the countdown stopping for the keyboard
	// and mouse and going on, with Left.
	WarmupPause  Kind = "warmup_pause"
	WarmupResume Kind = "warmup_resume"
	// WarmupPutOff drops the countdown for Len, with Left.
	WarmupPutOff Kind = "warmup_putoff"
	// WarmupDone is the countdown run out.
	WarmupDone Kind = "warmup_done"
)

// What started a warm-up, for Event.From.
const (
	FromReminder = "reminder"
	FromTray     = "tray"
)

// Event is one line of the file. Zero fields are left out of it.
type Event struct {
	Time time.Time
	Kind Kind
	// Sat is how long the stretch at the computer had lasted.
	Sat time.Duration
	// Len is how long a snooze or a put-off warm-up hides the reminder.
	Len time.Duration
	// Left is what remained of the warm-up countdown.
	Left time.Duration
	// From is FromReminder or FromTray.
	From string
}

// line is Event as it is written: durations in whole seconds.
type line struct {
	Time time.Time `json:"t"`
	Kind Kind      `json:"ev"`
	Sat  int64     `json:"sat,omitempty"`
	Len  int64     `json:"len,omitempty"`
	Left int64     `json:"left,omitempty"`
	From string    `json:"from,omitempty"`
}

// MarshalJSON writes the time to the second and durations as whole seconds.
func (e Event) MarshalJSON() ([]byte, error) {
	return json.Marshal(line{
		Time: e.Time.Round(0).Truncate(time.Second),
		Kind: e.Kind,
		Sat:  seconds(e.Sat),
		Len:  seconds(e.Len),
		Left: seconds(e.Left),
		From: e.From,
	})
}

func seconds(d time.Duration) int64 {
	return int64(d.Round(time.Second) / time.Second)
}

// Writer appends events to the file. It is safe to call from several
// goroutines.
type Writer struct {
	mu sync.Mutex
	f  *os.File
}

// Open appends to the file at path, creating it and its directory if needed.
func Open(path string) (*Writer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	// A line cut short by a crash or a power cut would swallow the next one;
	// ending it here costs one broken line instead of two.
	if st, err := f.Stat(); err == nil && st.Size() > 0 {
		last := make([]byte, 1)
		if _, err := f.ReadAt(last, st.Size()-1); err != nil && err != io.EOF {
			f.Close()
			return nil, err
		}
		if last[0] != '\n' {
			if _, err := f.Write([]byte{'\n'}); err != nil {
				f.Close()
				return nil, err
			}
		}
	}
	return &Writer{f: f}, nil
}

// Add appends one event as a single write, so a crash can cut only the line
// being written.
func (w *Writer) Add(e Event) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = w.f.Write(append(b, '\n'))
	return err
}

// Close closes the file.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.f.Close()
}
