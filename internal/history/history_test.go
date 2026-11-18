package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var at = time.Date(2026, 11, 18, 12, 42, 33, 400_000_000, time.FixedZone("", 5*3600))

func TestLine(t *testing.T) {
	for _, tc := range []struct {
		e    Event
		want string
	}{
		{Event{Time: at, Kind: Start}, `{"t":"2026-11-18T12:42:33+05:00","ev":"start"}`},
		{
			Event{Time: at, Kind: Snooze, Sat: time.Hour + 1500*time.Millisecond, Len: 30 * time.Minute},
			`{"t":"2026-11-18T12:42:33+05:00","ev":"snooze","sat":3602,"len":1800}`,
		},
		{
			Event{Time: at, Kind: WarmupStart, Sat: 90 * time.Minute, From: FromTray},
			`{"t":"2026-11-18T12:42:33+05:00","ev":"warmup_start","sat":5400,"from":"tray"}`,
		},
		{
			Event{Time: at, Kind: WarmupPutOff, Left: 252 * time.Second, Len: 5 * time.Minute},
			`{"t":"2026-11-18T12:42:33+05:00","ev":"warmup_putoff","len":300,"left":252}`,
		},
	} {
		b, err := tc.e.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.e.Kind, b, tc.want)
		}
	}
}

func TestAppends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sitnag", "history.jsonl")
	for _, k := range []Kind{Start, Quit} {
		w, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.Add(Event{Time: at, Kind: k}); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	}
	got := lines(t, path)
	if len(got) != 2 || !strings.Contains(got[0], `"start"`) || !strings.Contains(got[1], `"quit"`) {
		t.Errorf("got %q", got)
	}
}

// A line cut short by a crash stays broken, but the next event starts a line
// of its own.
func TestCutLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	if err := os.WriteFile(path, []byte(`{"t":"2026-11-18T12:42:33+05:00","ev":"start"}`+"\n"+`{"t":"2026-11-18T12:4`), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Add(Event{Time: at, Kind: Remind, Sat: time.Hour}); err != nil {
		t.Fatal(err)
	}
	w.Close()
	got := lines(t, path)
	want := `{"t":"2026-11-18T12:42:33+05:00","ev":"remind","sat":3600}`
	if len(got) != 3 || got[2] != want {
		t.Errorf("got %q, want the last line %s", got, want)
	}
}

func lines(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}
