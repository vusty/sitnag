//go:build windows

// sitnag counts the time spent at the computer and, after an hour, asks for a
// five-minute warm-up. See README.md.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/vusty/sitnag/internal/sit"
	"github.com/vusty/sitnag/internal/tray"
	"github.com/vusty/sitnag/internal/win"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "run":
		err = run(os.Args[2:])
	case "autostart":
		err = autostartCmd(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "sitnag:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: sitnag <run|autostart> [flags]")
	os.Exit(2)
}

func run(args []string) error {
	cfg := sit.DefaultConfig()
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	logPath := fs.String("log", "", "append the log to this file instead of stdout")
	langFlag := fs.String("lang", "auto", "auto, ru or en; auto follows the first language in the Windows language list")
	fs.DurationVar(&cfg.RemindAfter, "remind", cfg.RemindAfter, "time at the computer before the reminder")
	fs.DurationVar(&cfg.HardLimit, "limit", cfg.HardLimit, "past this only the short snooze is offered")
	fs.DurationVar(&cfg.Away, "away", cfg.Away, "no input this long counts as a break")
	fs.DurationVar(&cfg.Break, "break", cfg.Break, "length of the warm-up countdown")
	fs.DurationVar(&cfg.Busy, "busy", cfg.Busy, "input this recent during the warm-up pauses it")
	fs.DurationVar(&cfg.ShortSnooze, "short", cfg.ShortSnooze, "the short snooze")
	fs.DurationVar(&cfg.LongSnooze, "long", cfg.LongSnooze, "the long snooze")
	if err := fs.Parse(args); err != nil {
		return err
	}
	for name, d := range map[string]time.Duration{"remind": cfg.RemindAfter, "limit": cfg.HardLimit,
		"away": cfg.Away, "break": cfg.Break, "busy": cfg.Busy, "short": cfg.ShortSnooze, "long": cfg.LongSnooze} {
		if d <= 0 {
			return fmt.Errorf("-%s must be positive, got %v", name, d)
		}
	}

	lang, err := pickLanguage(*langFlag)
	if err != nil {
		return err
	}
	t = lang

	// Autostarted there is no console, so a log file is the only way to see
	// afterwards what the tool thought was happening.
	if *logPath != "" {
		if err := os.MkdirAll(filepath.Dir(*logPath), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(*logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		logOut = f
	}

	// One copy per session: a second one would mean two reminders at once.
	name, _ := windows.UTF16PtrFromString(`Local\sitnag`)
	if _, err := windows.CreateMutex(nil, false, name); errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		logf("already running, exiting")
		return nil
	}

	win.DPIAware()
	choices := make(chan win.Choice, 4)
	w, err := win.Open(choices, win.Labels{
		Start:       t.start,
		Short:       fmt.Sprintf(t.snooze, human(cfg.ShortSnooze)),
		Long:        fmt.Sprintf(t.snooze, human(cfg.LongSnooze)),
		FactHeading: t.factHeading,
	})
	if err != nil {
		return fmt.Errorf("window: %w", err)
	}

	logf("start: remind after %v, limit %v, away %v, break %v (paused by input within %v), snooze %v/%v, language %s",
		cfg.RemindAfter, cfg.HardLimit, cfg.Away, cfg.Break, cfg.Busy, cfg.ShortSnooze, cfg.LongSnooze, t.lang)
	tray.Run(tray.Options{
		Quit:    t.quit,
		Status:  t.trayStarting,
		OnReady: func() { go loop(sit.New(cfg), cfg, w, choices) },
		OnQuit:  func() { logf("quit") },
	})
	return nil
}

// loop owns the state: every tick and every button press goes through here,
// so nothing needs a lock.
func loop(state *sit.State, cfg sit.Config, w *win.Window, choices <-chan win.Choice) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	// Each reminder past the hard limit shows the next fact; the start is
	// random so a restart does not open on the same one.
	fact := int(time.Now().Unix()) % len(t.facts)
	var paused bool

	for {
		// Wall-clock time: see sit.State.Tick.
		now := time.Now().Round(0)
		before := state.Sitting(now)
		switch state.Tick(now, win.Idle()) {
		case sit.Ask:
			fact = (fact + 1) % len(t.facts)
			logf("reminder: at the computer for %s", human(state.Sitting(now)))
			win.Beep(win.Attention)
		case sit.Left:
			logf("left with the reminder up after %s: counted as the break", human(before))
		case sit.BreakDone:
			logf("warm-up done")
			win.Beep(win.Done)
		default:
			if before > 0 && state.Sitting(now) == 0 {
				logf("away after %s at the computer: count restarted", human(before))
			}
		}
		if p := state.BreakPaused(); p != paused {
			if p {
				logf("warm-up paused at %s left: the computer is in use", clock(state.BreakLeft()))
			} else if state.Mode() == sit.Breaking {
				logf("warm-up resumed")
			}
			paused = p
		}
		render(state, cfg, w, now, t.facts[fact])

		// A press goes straight back round the loop, so the window reacts at
		// once instead of on the next tick.
		select {
		case <-tick.C:
		case c := <-choices:
			now := time.Now().Round(0)
			sat := human(state.Sitting(now))
			switch c {
			case win.Start:
				if state.StartBreak(now) {
					logf("warm-up started after %s at the computer", sat)
				}
			case win.Short, win.Long:
				if state.Snooze(now, c == win.Long) {
					logf("snoozed until %s after %s at the computer", state.SnoozedUntil().Format("15:04:05"), sat)
				}
			}
		}
	}
}

// render makes the window and the tray match the state. fact is the one to
// show if the reminder comes past the hard limit.
func render(state *sit.State, cfg sit.Config, w *win.Window, now time.Time, fact string) {
	sat := state.Sitting(now)
	switch state.Mode() {
	case sit.Asking:
		if state.LongAllowed(now) {
			w.Ask(win.Ask{
				Title:   t.askTitle,
				Hero:    human(sat),
				Caption: fmt.Sprintf(t.askCaption, human(cfg.Break)),
				Long:    true,
			})
		} else {
			w.Ask(win.Ask{
				Title:   t.urgentTitle,
				Caption: fmt.Sprintf(t.urgentCaption, human(sat), human(cfg.ShortSnooze)),
				Fact:    fact,
			})
		}
		tray.Set(tray.Ask, fmt.Sprintf(t.trayAsk, human(sat)))
	case sit.Breaking:
		leftD := state.BreakLeft()
		left := clock(leftD)
		cd := win.Countdown{
			Title:    t.breakTitle,
			Clock:    left,
			Note:     t.breakNote,
			Progress: 1 - float64(leftD)/float64(cfg.Break),
		}
		status := fmt.Sprintf(t.trayBreak, left)
		if state.BreakPaused() {
			cd.Title = t.pausedTitle
			cd.Note = t.pausedNote
			cd.Paused = true
			status = fmt.Sprintf(t.trayPaused, left)
		}
		w.Countdown(cd)
		tray.Set(tray.Break, status)
	default:
		w.Hide()
		if sat == 0 {
			tray.Set(tray.Work, t.trayIdle)
			return
		}
		status := fmt.Sprintf(t.traySitting, human(sat))
		if until := state.SnoozedUntil(); until.After(now) {
			status += fmt.Sprintf(t.trayReturns, until.Format("15:04"))
		}
		tray.Set(tray.Work, status)
	}
}

// human renders a duration the way the window speaks: "42 min", "1 h 05 min".
// Seconds only show up with test-sized flags.
func human(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf(t.sec, int(d/time.Second))
	}
	m := int(d / time.Minute)
	switch {
	case m < 60:
		return fmt.Sprintf(t.min, m)
	case m%60 == 0:
		return fmt.Sprintf(t.hour, m/60)
	default:
		return fmt.Sprintf(t.hourMin, m/60, m%60)
	}
}

// clock renders the countdown as m:ss, rounding up so it never shows 0:00
// while time is left.
func clock(d time.Duration) string {
	s := int((d + time.Second - 1) / time.Second)
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

const autostartKey = `Software\Microsoft\Windows\CurrentVersion\Run`

// autostartCmd turns logon startup on and off through the Run key, the same
// way granag does it.
func autostartCmd(args []string) error {
	action := "status"
	if len(args) > 0 {
		action = args[0]
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, autostartKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	switch action {
	case "on":
		self, err := os.Executable()
		if err != nil {
			return err
		}
		// Anything after "on" is passed to run at logon, which is how the log
		// file gets set.
		value := `"` + self + `" run`
		for _, arg := range args[1:] {
			if strings.ContainsAny(arg, ` "`) {
				value += ` "` + arg + `"`
				continue
			}
			value += " " + arg
		}
		if err := k.SetStringValue("sitnag", value); err != nil {
			return err
		}
		fmt.Println("autostart on:", value)
	case "off":
		if err := k.DeleteValue("sitnag"); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		fmt.Println("autostart off")
	case "status":
		v, _, err := k.GetStringValue("sitnag")
		if errors.Is(err, registry.ErrNotExist) {
			fmt.Println("autostart off")
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Println("autostart on:", v)
	default:
		return fmt.Errorf("autostart: want on, off or status, got %q", action)
	}
	return nil
}

// logOut is where logf writes; -log points it at a file.
var logOut io.Writer = os.Stdout

func logf(format string, args ...any) {
	fmt.Fprintf(logOut, "%s  %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}
