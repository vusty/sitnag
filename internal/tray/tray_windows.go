// Package tray puts sitnag in the notification area: a chair, a tooltip and
// a menu line with how long you have been sitting, a way to start the warm-up
// early, and a way to quit.
package tray

import (
	_ "embed"
	"sync"

	"github.com/getlantern/systray"
)

// The same chair the exe carries; icons/make.py draws it.
//
//go:embed icons/chair.ico
var chair []byte

// State is what the tool is doing, as far as the menu cares.
type State int

const (
	Work  State = iota // counting
	Ask                // time to get up
	Break              // warm-up running
)

// Options wires the tray to the rest of the tool.
type Options struct {
	// Quit is the caption of the menu item that exits.
	Quit string
	// BreakNow is the caption of the menu item that starts the warm-up
	// without waiting for the reminder.
	BreakNow string
	// OnBreakNow runs on the tray's goroutine when that item is clicked.
	OnBreakNow func()
	// Status is the first status line, until the tool sets its own.
	Status string
	// OnReady runs once the icon exists; Set is safe to call from then on.
	OnReady func()
	// OnQuit runs after the tray closes.
	OnQuit func()
}

var (
	mu         sync.Mutex
	statusItem *systray.MenuItem
	breakItem  *systray.MenuItem
	lastState  = State(-1)
	lastStatus string
)

// Run shows the icon and blocks until the user quits. It must be called on the
// main goroutine: the tray owns a window message loop.
func Run(o Options) {
	systray.Run(func() { ready(o) }, o.OnQuit)
}

// Set tells the tray what the tool is doing and changes the status line. Safe
// from any goroutine; repeating the same values costs nothing.
func Set(state State, status string) {
	mu.Lock()
	defer mu.Unlock()
	if state != lastState {
		// A warm-up is already running: nothing to start.
		if breakItem != nil {
			if state == Break {
				breakItem.Disable()
			} else {
				breakItem.Enable()
			}
		}
		lastState = state
	}
	if status != lastStatus {
		systray.SetTooltip("sitnag: " + status)
		if statusItem != nil {
			statusItem.SetTitle(status)
		}
		lastStatus = status
	}
}

func ready(o Options) {
	systray.SetIcon(chair)
	systray.SetTitle("sitnag")
	item := systray.AddMenuItem("", "")
	item.Disable()
	breakNow := systray.AddMenuItem(o.BreakNow, "")
	systray.AddSeparator()
	quit := systray.AddMenuItem(o.Quit, "")

	mu.Lock()
	statusItem = item
	breakItem = breakNow
	mu.Unlock()
	Set(Work, o.Status)

	go func() {
		for {
			select {
			case <-breakNow.ClickedCh:
				if o.OnBreakNow != nil {
					o.OnBreakNow()
				}
			case <-quit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
	if o.OnReady != nil {
		o.OnReady()
	}
}
