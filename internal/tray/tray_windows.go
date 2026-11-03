// Package tray puts sitnag in the notification area: a dot whose colour says
// what the tool is doing, a tooltip and a menu line with how long you have
// been sitting, and a way to quit.
package tray

import (
	_ "embed"
	"sync"

	"github.com/getlantern/systray"
)

var (
	//go:embed icons/work.ico
	iconWork []byte
	//go:embed icons/ask.ico
	iconAsk []byte
	//go:embed icons/break.ico
	iconBreak []byte
)

// Icon is the colour of the dot.
type Icon int

const (
	Work  Icon = iota // green: counting
	Ask               // orange: time to get up
	Break             // blue: warm-up running
)

var icons = map[Icon][]byte{Work: iconWork, Ask: iconAsk, Break: iconBreak}

// Options wires the tray to the rest of the tool.
type Options struct {
	// OnReady runs once the icon exists; Set is safe to call from then on.
	OnReady func()
	// OnQuit runs after the tray closes.
	OnQuit func()
}

var (
	mu         sync.Mutex
	statusItem *systray.MenuItem
	lastIcon   = Icon(-1)
	lastStatus string
)

// Run shows the icon and blocks until the user quits. It must be called on the
// main goroutine: the tray owns a window message loop.
func Run(o Options) {
	systray.Run(func() { ready(o) }, o.OnQuit)
}

// Set changes the dot and the status line. Safe from any goroutine; repeating
// the same values costs nothing.
func Set(icon Icon, status string) {
	mu.Lock()
	defer mu.Unlock()
	if icon != lastIcon {
		systray.SetIcon(icons[icon])
		lastIcon = icon
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
	systray.SetTitle("sitnag")
	item := systray.AddMenuItem("", "")
	item.Disable()
	systray.AddSeparator()
	quit := systray.AddMenuItem("Выход", "")

	mu.Lock()
	statusItem = item
	mu.Unlock()
	Set(Work, "считаю время за компом")

	go func() {
		<-quit.ClickedCh
		systray.Quit()
	}()
	if o.OnReady != nil {
		o.OnReady()
	}
}
