package win

import (
	"errors"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Choice is the button the user pressed.
type Choice int

const (
	Start Choice = iota + 1 // the values double as control IDs
	Short
	Long
)

// Labels are the button captions.
type Labels struct {
	Start, Short, Long string
}

const (
	wsPopup   = 0x80000000
	wsCaption = 0x00C00000
	wsChild   = 0x40000000
	wsVisible = 0x10000000

	wsExTopmost    = 0x00000008
	wsExToolWindow = 0x00000080
	wsExNoActivate = 0x08000000

	ssCenter   = 0x1
	ssNoPrefix = 0x80

	wmClose   = 0x0010
	wmSetFont = 0x0030
	wmCommand = 0x0111
	wmApp     = 0x8000

	swHide           = 0
	swShowNoActivate = 4

	swpNoSize       = 0x1
	swpNoActivate   = 0x10
	swpShowWindow   = 0x40
	hwndTopmost     = ^uintptr(0) // (HWND)-1
	spiGetWorkArea  = 0x30
	colorBtnFace    = 15
	idcArrow        = 32512
	fwNormal        = 400
	fwSemiBold      = 600
	defaultCharset  = 1
	cleartypeQualty = 5
)

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   windows.Handle
	icon       windows.Handle
	cursor     windows.Handle
	background windows.Handle
	menuName   *uint16
	className  *uint16
	iconSm     windows.Handle
}

type msg struct {
	hwnd    windows.HWND
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
	private uint32
}

type rect struct{ left, top, right, bottom int32 }

type viewKind int

const (
	hidden viewKind = iota
	asking
	countdown
)

// view is what the window should look like. It is comparable, so an update
// that changes nothing costs nothing.
type view struct {
	kind  viewKind
	title string
	text  string // the reminder's message, or the countdown's clock
	long  bool   // whether the long snooze button is on offer
}

// Window is the reminder and the countdown in one: a small always-on-top
// window that never takes the keyboard focus, so a keystroke meant for
// another app cannot press one of its buttons.
type Window struct {
	choices chan<- Choice

	mu      sync.Mutex
	pending view

	// Owned by the window thread.
	hwnd                        windows.HWND
	shown                       view
	title, text, clock          windows.HWND
	btnStart, btnShort, btnLong windows.HWND
}

// current is the one window, for the window procedure. Windows hands the
// procedure a handle, not a Go value.
var current *Window

// Open creates the window, hidden, on a thread of its own, and returns once it
// exists. Button presses arrive on choices; a press is dropped rather than
// block the window if nobody is reading.
func Open(choices chan<- Choice, l Labels) (*Window, error) {
	if current != nil {
		return nil, errors.New("window already open")
	}
	w := &Window{choices: choices}
	ready := make(chan error)
	go w.loop(l, ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return w, nil
}

// Ask shows the reminder. long says whether the long snooze is on offer.
func (w *Window) Ask(title, text string, long bool) {
	w.set(view{kind: asking, title: title, text: text, long: long})
}

// Countdown shows the warm-up clock in place of the buttons.
func (w *Window) Countdown(title, clock string) {
	w.set(view{kind: countdown, title: title, text: clock})
}

// Hide takes the window off the screen.
func (w *Window) Hide() { w.set(view{}) }

func (w *Window) set(v view) {
	w.mu.Lock()
	changed := w.pending != v
	w.pending = v
	w.mu.Unlock()
	if changed {
		procPostMessage.Call(uintptr(w.hwnd), wmApp, 0, 0)
	}
}

func (w *Window) loop(l Labels, ready chan<- error) {
	// A window belongs to the thread that created it, and its messages are
	// only ever delivered there.
	runtime.LockOSThread()
	if err := w.create(l); err != nil {
		ready <- err
		return
	}
	current = w
	ready <- nil

	var m msg
	for {
		r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// Layout in 96-DPI pixels, scaled at creation.
const (
	clientW  = 520
	clientH  = 176
	margin   = 24
	buttonsY = 120
	buttonH  = 36
	gap      = 8
)

func (w *Window) create(l Labels) error {
	enableVisualStyles()
	dpi := systemDPI()
	px := func(v int) uintptr { return uintptr(v * dpi / 96) }
	font := func(pt, weight int) uintptr {
		f, _, _ := procCreateFont.Call(
			uintptr(int32(-pt*dpi/72)), 0, 0, 0, uintptr(weight), 0, 0, 0,
			defaultCharset, 0, 0, cleartypeQualty, 0,
			uintptr(unsafe.Pointer(windows.StringToUTF16Ptr("Segoe UI"))))
		return f
	}

	instance, _, _ := procGetModuleHandle.Call(0)
	cursor, _, _ := procLoadCursor.Call(0, idcArrow)
	className := windows.StringToUTF16Ptr("sitnag")
	wc := wndClassEx{
		wndProc:    windows.NewCallback(wndProc),
		instance:   windows.Handle(instance),
		cursor:     windows.Handle(cursor),
		background: windows.Handle(colorBtnFace + 1),
		className:  className,
	}
	wc.size = uint32(unsafe.Sizeof(wc))
	if r, _, err := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return err
	}

	const style = wsPopup | wsCaption
	const exStyle = wsExTopmost | wsExToolWindow | wsExNoActivate
	r := rect{0, 0, int32(px(clientW)), int32(px(clientH))}
	procAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&r)), style, 0, exStyle)
	hwnd, _, err := procCreateWindowEx.Call(exStyle,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr("sitnag"))),
		style, 0, 0, uintptr(r.right-r.left), uintptr(r.bottom-r.top),
		0, 0, instance, 0)
	if hwnd == 0 {
		return err
	}
	w.hwnd = windows.HWND(hwnd)

	child := func(class, text string, style uintptr, id, x, y, cx, cy int, f uintptr) windows.HWND {
		h, _, _ := procCreateWindowEx.Call(0,
			uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(class))),
			uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(text))),
			wsChild|style, px(x), px(y), px(cx), px(cy),
			hwnd, uintptr(id), instance, 0)
		procSendMessage.Call(h, wmSetFont, f, 1)
		return windows.HWND(h)
	}

	titleFont, textFont := font(18, fwSemiBold), font(12, fwNormal)
	clockFont, buttonFont := font(40, fwSemiBold), font(11, fwNormal)
	inner := clientW - 2*margin
	w.title = child("STATIC", "", wsVisible|ssNoPrefix, 0, margin, 16, inner, 40, titleFont)
	w.text = child("STATIC", "", wsVisible|ssNoPrefix, 0, margin, 60, inner, 52, textFont)
	w.clock = child("STATIC", "", ssCenter|ssNoPrefix, 0, margin, 58, inner, 90, clockFont)

	startW, snoozeW := 168, (inner-168-2*gap)/2
	w.btnStart = child("BUTTON", l.Start, wsVisible, int(Start), margin, buttonsY, startW, buttonH, buttonFont)
	w.btnShort = child("BUTTON", l.Short, wsVisible, int(Short), margin+startW+gap, buttonsY, snoozeW, buttonH, buttonFont)
	w.btnLong = child("BUTTON", l.Long, wsVisible, int(Long), margin+startW+snoozeW+2*gap, buttonsY, snoozeW, buttonH, buttonFont)
	return nil
}

func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	w := current
	switch message {
	case wmCommand:
		// HIWORD 0 is BN_CLICKED; LOWORD is the control ID.
		if w != nil && wParam>>16 == 0 {
			if c := Choice(wParam & 0xffff); c >= Start && c <= Long {
				select {
				case w.choices <- c:
				default:
				}
			}
		}
		return 0
	case wmApp:
		if w != nil {
			w.apply()
		}
		return 0
	case wmClose:
		// Alt+F4 does nothing: the window goes away through its buttons.
		return 0
	}
	r, _, _ := procDefWindowProc.Call(hwnd, message, wParam, lParam)
	return r
}

// apply brings the window in line with the pending view. Window thread only.
func (w *Window) apply() {
	w.mu.Lock()
	v := w.pending
	w.mu.Unlock()
	old := w.shown
	if v == old {
		return
	}
	w.shown = v

	if v.kind == hidden {
		procShowWindow.Call(uintptr(w.hwnd), swHide)
		return
	}
	if v.title != old.title {
		setText(w.title, v.title)
	}
	isAsk := v.kind == asking
	if isAsk {
		if v.text != old.text || old.kind != asking {
			setText(w.text, v.text)
		}
	} else if v.text != old.text || old.kind != countdown {
		setText(w.clock, v.text)
	}
	show(w.text, isAsk)
	show(w.btnStart, isAsk)
	show(w.btnShort, isAsk)
	show(w.btnLong, isAsk && v.long)
	show(w.clock, !isAsk)

	if old.kind == hidden {
		w.center()
	}
}

// center puts the window in the middle of the primary screen's work area, on
// top of everything, without activating it.
func (w *Window) center() {
	var work, win rect
	procSystemParametersInfo.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&work)), 0)
	procGetWindowRect.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(&win)))
	width, height := win.right-win.left, win.bottom-win.top
	x := work.left + (work.right-work.left-width)/2
	y := work.top + (work.bottom-work.top-height)/2
	procSetWindowPos.Call(uintptr(w.hwnd), hwndTopmost,
		uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoActivate|swpShowWindow)
}

func setText(h windows.HWND, s string) {
	procSetWindowText.Call(uintptr(h), uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(s))))
}

func show(h windows.HWND, on bool) {
	cmd := uintptr(swHide)
	if on {
		cmd = swShowNoActivate
	}
	procShowWindow.Call(uintptr(h), cmd)
}
