package win

import (
	"errors"
	"math"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/vusty/sitnag/internal/figure"
)

// Choice is the button the user pressed.
type Choice int

const (
	Start Choice = iota + 1
	Short
	Long
)

// Labels are the button captions and the heading over the fact.
type Labels struct {
	Start, Short, Long string
	FactHeading        string
}

// Ask is the reminder. With Fact set it takes the urgent look: red, and the
// fact in a panel of its own instead of the big number.
type Ask struct {
	Title   string
	Hero    string // the big line in the calm look: how long at the computer
	Caption string // under the hero, or under the title in the urgent look
	Fact    string
	Source  string // the study behind the fact
	Long    bool   // whether the long snooze is on offer
}

// Countdown is the warm-up: the clock, a mannequin showing what to do, and
// the short snooze as the one button, for a warm-up that has to wait.
type Countdown struct {
	Title    string
	Clock    string
	Move     int    // which move the mannequin does, a figure.Move
	Name     string // what the move is called; empty moves the cue up into its place
	Cue      string // how to do it, a line or two
	Note     string
	Paused   bool    // the keyboard or mouse is in use, the clock stands still
	Progress float64 // 0 at the start, 1 at the end
}

const (
	wsPopup = 0x80000000

	wsExTopmost    = 0x00000008
	wsExToolWindow = 0x00000080
	wsExNoActivate = 0x08000000

	csDropShadow = 0x00020000

	wmPaint          = 0x000F
	wmClose          = 0x0010
	wmEraseBkgnd     = 0x0014
	wmSetCursor      = 0x0020
	wmMouseMove      = 0x0200
	wmLButtonDown    = 0x0201
	wmLButtonUp      = 0x0202
	wmCaptureChanged = 0x0215
	wmMouseLeave     = 0x02A3
	wmTimer          = 0x0113
	wmApp            = 0x8000

	swHide = 0

	// The mannequin moves at 25 frames a second: smooth enough, and a repaint
	// of a card this size is cheap.
	animTimer = 1
	animEvery = 40 // milliseconds

	swpNoActivate  = 0x10
	swpShowWindow  = 0x40
	hwndTopmost    = ^uintptr(0) // (HWND)-1
	spiGetWorkArea = 0x30
	idcArrow       = 32512
	idcHand        = 32649
	tmeLeave       = 0x2

	dwmaWindowCornerPreference = 33
	dwmcpRound                 = 2

	fwNormal        = 400
	fwSemiBold      = 600
	fwBold          = 700
	defaultCharset  = 1
	cleartypeQualty = 5

	dtCenter     = 0x1
	dtRight      = 0x2
	dtVCenter    = 0x4
	dtWordBreak  = 0x10
	dtSingleLine = 0x20
	dtNoPrefix   = 0x800
	dtEllipsis   = 0x8000

	srcCopy     = 0x00CC0020
	transparent = 1
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

type paintStruct struct {
	hdc         uintptr
	erase       int32
	paint       rect
	restore     int32
	incUpdate   int32
	rgbReserved [32]byte
}

type trackMouseEvent struct {
	size      uint32
	flags     uint32
	hwnd      uintptr
	hoverTime uint32
}

type viewKind int

const (
	hidden viewKind = iota
	asking
	countdown
)

// view is what the window should look like. It is comparable, so an update
// that changes nothing costs nothing.
type view struct {
	kind viewKind
	ask  Ask
	cd   Countdown
}

// Window is the reminder and the countdown in one: an always-on-top card that
// never takes the keyboard focus, so a keystroke meant for another app cannot
// press one of its buttons. It paints itself, buttons included; the stock
// controls only come in white.
type Window struct {
	choices chan<- Choice
	labels  Labels

	mu      sync.Mutex
	pending view

	// Owned by the window thread.
	hwnd     windows.HWND
	dpi      int
	shown    view
	hover    Choice
	pressed  Choice
	tracking bool
	fonts    fonts

	// How far into its move the mannequin is: moveHeld, plus the time since
	// moveFrom while it is moving. It stands still while the warm-up is paused.
	moving   bool
	moveFrom time.Time
	moveHeld time.Duration
}

type fonts struct {
	title, hero, text, factHead, fact, clock, move, note, button uintptr
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
	w := &Window{choices: choices, labels: l}
	ready := make(chan error)
	go w.loop(ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return w, nil
}

// Ask shows the reminder.
func (w *Window) Ask(a Ask) { w.set(view{kind: asking, ask: a}) }

// Countdown shows the warm-up in place of the reminder.
func (w *Window) Countdown(c Countdown) { w.set(view{kind: countdown, cd: c}) }

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

func (w *Window) loop(ready chan<- error) {
	// A window belongs to the thread that created it, and its messages are
	// only ever delivered there.
	runtime.LockOSThread()
	if err := w.create(); err != nil {
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

// Layout in 96-DPI pixels, scaled when drawn.
const (
	cardW   = 860
	cardH   = 400
	margin  = 40
	buttonH = 56
	buttonY = cardH - 36 - buttonH
	gap     = 12
	startW  = 300
	radius  = 12
)

func (w *Window) create() error {
	if err := startGDIPlus(); err != nil {
		return err
	}
	w.dpi = systemDPI()
	w.makeFonts()

	instance, _, _ := procGetModuleHandle.Call(0)
	cursor, _, _ := procLoadCursor.Call(0, idcArrow)
	className := windows.StringToUTF16Ptr("sitnag")
	wc := wndClassEx{
		style:     csDropShadow,
		wndProc:   windows.NewCallback(wndProc),
		instance:  windows.Handle(instance),
		cursor:    windows.Handle(cursor),
		className: className,
	}
	wc.size = uint32(unsafe.Sizeof(wc))
	if r, _, err := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return err
	}

	const exStyle = wsExTopmost | wsExToolWindow | wsExNoActivate
	hwnd, _, err := procCreateWindowEx.Call(exStyle,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr("sitnag"))),
		wsPopup, 0, 0, uintptr(w.px(cardW)), uintptr(w.px(cardH)),
		0, 0, instance, 0)
	if hwnd == 0 {
		return err
	}
	w.hwnd = windows.HWND(hwnd)

	// Rounded corners on Windows 11; older systems ignore the attribute.
	if procDwmSetWindowAttribute.Find() == nil {
		pref := uint32(dwmcpRound)
		procDwmSetWindowAttribute.Call(hwnd, dwmaWindowCornerPreference,
			uintptr(unsafe.Pointer(&pref)), unsafe.Sizeof(pref))
	}
	return nil
}

func (w *Window) makeFonts() {
	font := func(pt, weight int) uintptr {
		f, _, _ := procCreateFont.Call(
			uintptr(int32(-pt*w.dpi/72)), 0, 0, 0, uintptr(weight), 0, 0, 0,
			defaultCharset, 0, 0, cleartypeQualty, 0,
			uintptr(unsafe.Pointer(windows.StringToUTF16Ptr("Segoe UI"))))
		return f
	}
	w.fonts = fonts{
		title:    font(26, fwSemiBold),
		hero:     font(54, fwSemiBold),
		text:     font(15, fwNormal),
		factHead: font(11, fwBold),
		fact:     font(15, fwNormal),
		clock:    font(68, fwSemiBold),
		move:     font(18, fwSemiBold),
		note:     font(11, fwNormal),
		button:   font(13, fwSemiBold),
	}
}

func (w *Window) px(v int) int { return v * w.dpi / 96 }

func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	w := current
	if w == nil {
		r, _, _ := procDefWindowProc.Call(hwnd, message, wParam, lParam)
		return r
	}
	x, y := int(int16(lParam&0xffff)), int(int16(lParam>>16&0xffff))
	switch message {
	case wmApp:
		w.apply()
		return 0
	case wmEraseBkgnd:
		return 1
	case wmPaint:
		var ps paintStruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		w.paint(hdc)
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case wmMouseMove:
		if !w.tracking {
			tme := trackMouseEvent{flags: tmeLeave, hwnd: hwnd}
			tme.size = uint32(unsafe.Sizeof(tme))
			procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
			w.tracking = true
		}
		w.setHover(w.hit(x, y))
		return 0
	case wmMouseLeave:
		w.tracking = false
		w.setHover(0)
		return 0
	case wmTimer:
		w.redraw()
		return 0
	case wmSetCursor:
		if w.hover != 0 {
			hand, _, _ := procLoadCursor.Call(0, idcHand)
			procSetCursor.Call(hand)
			return 1
		}
	case wmLButtonDown:
		if c := w.hit(x, y); c != 0 {
			w.pressed = c
			procSetCapture.Call(hwnd)
			w.redraw()
		}
		return 0
	case wmLButtonUp:
		if p := w.pressed; p != 0 {
			w.pressed = 0
			procReleaseCapture.Call()
			if w.hit(x, y) == p {
				select {
				case w.choices <- p:
				default:
				}
			}
			w.redraw()
		}
		return 0
	case wmCaptureChanged:
		if w.pressed != 0 {
			w.pressed = 0
			w.redraw()
		}
		return 0
	case wmClose:
		// Alt+F4 does nothing: the window goes away through its buttons.
		return 0
	}
	r, _, _ := procDefWindowProc.Call(hwnd, message, wParam, lParam)
	return r
}

func (w *Window) setHover(c Choice) {
	if c != w.hover {
		w.hover = c
		w.redraw()
	}
}

func (w *Window) redraw() {
	procInvalidateRect.Call(uintptr(w.hwnd), 0, 0)
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
	// The buttons change with the kind of view, not with the clock ticking
	// inside one: a press must survive the second it happens in.
	if v.kind != old.kind {
		w.hover = 0
		if w.pressed != 0 {
			w.pressed = 0
			procReleaseCapture.Call()
		}
	}
	w.animate(old, v)

	if v.kind == hidden {
		procShowWindow.Call(uintptr(w.hwnd), swHide)
		return
	}
	w.redraw()
	if old.kind == hidden {
		w.center()
	}
}

// animate keeps the mannequin's clock: a new move starts from its beginning,
// and a paused warm-up holds the pose it was caught in.
func (w *Window) animate(old, v view) {
	moving := v.kind == countdown && !v.cd.Paused
	switch {
	case v.kind != countdown || old.kind != countdown || old.cd.Move != v.cd.Move:
		w.moveHeld = 0
		w.moveFrom = time.Now()
	case w.moving && !moving:
		w.moveHeld += time.Since(w.moveFrom)
	case !w.moving && moving:
		w.moveFrom = time.Now()
	}
	if moving == w.moving {
		return
	}
	w.moving = moving
	if moving {
		procSetTimer.Call(uintptr(w.hwnd), animTimer, animEvery, 0)
	} else {
		procKillTimer.Call(uintptr(w.hwnd), animTimer)
	}
}

func (w *Window) moveTime() time.Duration {
	if !w.moving {
		return w.moveHeld
	}
	return w.moveHeld + time.Since(w.moveFrom)
}

// center puts the window in the middle of the primary screen's work area, on
// top of everything, without activating it.
func (w *Window) center() {
	var work rect
	procSystemParametersInfo.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&work)), 0)
	width, height := int32(w.px(cardW)), int32(w.px(cardH))
	x := work.left + (work.right-work.left-width)/2
	y := work.top + (work.bottom-work.top-height)/2
	procSetWindowPos.Call(uintptr(w.hwnd), hwndTopmost,
		uintptr(x), uintptr(y), uintptr(width), uintptr(height), swpNoActivate|swpShowWindow)
}

// button is one clickable area, in 96-DPI pixels.
type button struct {
	id      Choice
	label   string
	primary bool
	r       rect
}

func (w *Window) buttons() []button {
	v := w.shown
	switch v.kind {
	case asking:
	case countdown:
		return []button{{Short, w.labels.Short, false, laterButton}}
	default:
		return nil
	}
	inner := cardW - 2*margin
	snoozeW := (inner - startW - 2*gap) / 2
	shortW := snoozeW
	if !v.ask.Long {
		shortW = 2*snoozeW + gap
	}
	x0 := int32(margin)
	x1 := x0 + startW + gap
	x2 := x1 + int32(snoozeW) + gap
	out := []button{
		{Start, w.labels.Start, true, rect{x0, buttonY, x0 + startW, buttonY + buttonH}},
		{Short, w.labels.Short, false, rect{x1, buttonY, x1 + int32(shortW), buttonY + buttonH}},
	}
	if v.ask.Long {
		out = append(out, button{Long, w.labels.Long, false, rect{x2, buttonY, x2 + int32(snoozeW), buttonY + buttonH}})
	}
	return out
}

// hit returns the button under a point in client pixels.
func (w *Window) hit(x, y int) Choice {
	for _, b := range w.buttons() {
		if x >= w.px(int(b.r.left)) && x < w.px(int(b.r.right)) &&
			y >= w.px(int(b.r.top)) && y < w.px(int(b.r.bottom)) {
			return b.id
		}
	}
	return 0
}

// theme is the card's colouring for one state.
type theme struct {
	from, to uint32 // gradient, ARGB, top-left to bottom-right
	ink      uint32 // text on the primary button, ARGB
}

var (
	themeCalm   = theme{0xFF1E3A8A, 0xFF0F766E, 0xFF1E3A8A} // blue into teal
	themeUrgent = theme{0xFF7F1D1D, 0xFF4C1D95, 0xFF7F1D1D} // deep red into violet
	themeBreak  = theme{0xFF065F46, 0xFF0E7490, 0xFF065F46} // green into cyan
	themePaused = theme{0xFF92400E, 0xFF374151, 0xFF92400E} // amber into slate
)

func (w *Window) theme() theme {
	v := w.shown
	switch {
	case v.kind == countdown && v.cd.Paused:
		return themePaused
	case v.kind == countdown:
		return themeBreak
	case v.ask.Fact != "":
		return themeUrgent
	default:
		return themeCalm
	}
}

// paint draws the whole card into a bitmap first and copies it once, so the
// countdown ticking every second does not flicker.
func (w *Window) paint(hdc uintptr) {
	width, height := w.px(cardW), w.px(cardH)
	mem, _, _ := procCreateCompatibleDC.Call(hdc)
	bmp, _, _ := procCreateCompatibleBitmap.Call(hdc, uintptr(width), uintptr(height))
	old, _, _ := procSelectObject.Call(mem, bmp)

	w.drawShapes(mem)
	w.drawText(mem)

	procBitBlt.Call(hdc, 0, 0, uintptr(width), uintptr(height), mem, 0, 0, srcCopy)
	procSelectObject.Call(mem, old)
	procDeleteObject.Call(bmp)
	procDeleteDC.Call(mem)
}

// drawShapes is the GDI+ half: the gradient, the panels and the buttons, all
// antialiased and some see-through, which GDI alone cannot do.
func (w *Window) drawShapes(dc uintptr) {
	g, err := newGraphics(dc)
	if err != nil {
		return
	}
	defer g.close()
	th := w.theme()
	v := w.shown

	g.gradient(0, 0, w.px(cardW), w.px(cardH), th.from, th.to)
	// Two soft circles of light, so the card is not a flat slab.
	g.ellipse(w.px(cardW-260), w.px(-200), w.px(480), w.px(480), 0x14FFFFFF)
	g.ellipse(w.px(-160), w.px(cardH-150), w.px(340), w.px(340), 0x0FFFFFFF)

	switch v.kind {
	case asking:
		if v.ask.Fact != "" {
			p := factPanel
			g.roundRect(w.px(int(p.left)), w.px(int(p.top)), w.px(int(p.right-p.left)), w.px(int(p.bottom-p.top)), w.px(14), 0x38000000)
			g.roundRect(w.px(int(p.left)+14), w.px(int(p.top)+18), w.px(5), w.px(int(p.bottom-p.top)-36), w.px(2), 0xFFF87171)
		}
	case countdown:
		x, y, full := w.px(margin), w.px(progressY), w.px(textW)
		g.roundRect(x, y, full, w.px(8), w.px(4), 0x33FFFFFF)
		done := int(math.Round(float64(full) * min(1, max(0, v.cd.Progress))))
		if done >= w.px(8) {
			g.roundRect(x, y, done, w.px(8), w.px(4), 0xE6FFFFFF)
		}
		scale := figureScale * float64(w.dpi) / 96
		w.drawFigure(g, figure.At(v.cd.Move, w.moveTime()), float64(w.px(figureX)), float64(w.px(figureFloor)), scale)
	}

	for _, b := range w.buttons() {
		fill := uint32(0x29FFFFFF)
		switch {
		case b.primary && w.pressed == b.id:
			fill = 0xFFE2E8F0
		case b.primary && w.hover == b.id:
			fill = 0xFFFFFFFF
		case b.primary:
			fill = 0xFFF1F5F9
		case w.pressed == b.id:
			fill = 0x52FFFFFF
		case w.hover == b.id:
			fill = 0x3DFFFFFF
		}
		g.roundRect(w.px(int(b.r.left)), w.px(int(b.r.top)), w.px(int(b.r.right-b.r.left)), w.px(int(b.r.bottom-b.r.top)), w.px(radius), fill)
	}
}

var factPanel = rect{margin, 124, cardW - margin, 288}

// The warm-up: text in a column on the left, the mannequin on the right.
const (
	textW       = 470 // width of the text column
	progressY   = 292
	figureX     = cardW - margin - 138 // where the mannequin stands
	figureFloor = 364
	figureScale = 1.42 // pixels per unit: room for arms stretched overhead
)

// laterButton puts the warm-up off. It sits beside the clock, clear of the
// row where the reminder has its buttons: the warm-up opens under the cursor
// that just pressed "start", and a second click must not land on this.
var laterButton = rect{margin + textW - 210, 107, margin + textW, 147}

// drawFigure paints the mannequin standing at (x, floor), part by part from
// the back. Each part gets a dark rim first, so an arm in front of the trunk
// reads as an arm and not as more trunk.
func (w *Window) drawFigure(g *graphics, f figure.Frame, x, floor, scale float64) {
	g.ellipse(int(x-f.ShadowW*scale/2), int(floor-f.ShadowH*scale/2),
		int(f.ShadowW*scale), int(f.ShadowH*scale), 0x30000000)
	onScreen := func(s figure.Shape, pad float64) []float32 {
		pts := s.Outline(pad)
		out := make([]float32, len(pts))
		for i := 0; i < len(pts); i += 2 {
			out[i] = float32(x + pts[i]*scale)
			out[i+1] = float32(floor - pts[i+1]*scale)
		}
		return out
	}
	for _, p := range f.Parts {
		rim := make([][]float32, 0, len(p.Shapes))
		body := make([][]float32, 0, len(p.Shapes))
		for _, s := range p.Shapes {
			rim = append(rim, onScreen(s, 1.7))
			body = append(body, onScreen(s, 0))
		}
		g.fill(rim, 0x520B1B2B)
		g.fill(body, figureTone(p.Tone))
	}
}

// figureTone is white for the side of the body nearest the viewer, going to
// a cool grey for the far side.
func figureTone(tone float64) uint32 {
	k := math.Pow(min(1, max(0, tone)), 0.7)
	mix := func(far, near float64) uint32 { return uint32(math.Round(far + (near-far)*k)) }
	return 0xFF000000 | mix(0x86, 0xFF)<<16 | mix(0x9E, 0xFF)<<8 | mix(0xB8, 0xFF)
}

// drawText is the GDI half: ClearType text on top of the shapes.
func (w *Window) drawText(dc uintptr) {
	procSetBkMode.Call(dc, transparent)
	v := w.shown
	white, soft := rgb(0xFFFFFF), rgb(0xE2E8F0)
	in := func(l, t, r, b int) rect {
		return rect{int32(w.px(l)), int32(w.px(t)), int32(w.px(r)), int32(w.px(b))}
	}
	right := cardW - margin

	switch v.kind {
	case asking:
		a := v.ask
		w.text(dc, a.Title, w.fonts.title, white, in(margin, 30, right, 80), dtSingleLine|dtEllipsis)
		if a.Fact == "" {
			w.text(dc, a.Hero, w.fonts.hero, white, in(margin, 96, right, 178), dtSingleLine)
			w.text(dc, a.Caption, w.fonts.text, soft, in(margin, 182, right, 250), dtWordBreak)
		} else {
			w.text(dc, a.Caption, w.fonts.text, soft, in(margin, 82, right, 116), dtSingleLine|dtEllipsis)
			p := factPanel
			w.text(dc, w.labels.FactHeading, w.fonts.factHead, rgb(0xFCA5A5), in(int(p.left)+36, int(p.top)+18, int(p.right)-24, int(p.top)+42), dtSingleLine)
			w.text(dc, a.Source, w.fonts.note, rgb(0xCBD5E1), in(int(p.left)+220, int(p.top)+18, int(p.right)-28, int(p.top)+42), dtSingleLine|dtRight|dtEllipsis)
			w.text(dc, a.Fact, w.fonts.fact, white, in(int(p.left)+36, int(p.top)+46, int(p.right)-28, int(p.bottom)-14), dtWordBreak)
		}
	case countdown:
		c := v.cd
		col := margin + textW
		w.text(dc, c.Title, w.fonts.title, white, in(margin, 30, col, 80), dtSingleLine|dtEllipsis)
		w.text(dc, c.Clock, w.fonts.clock, white, in(margin-4, 68, col, 172), dtVCenter|dtSingleLine)
		cueY := 214
		if c.Name == "" {
			cueY = 186
		}
		w.text(dc, c.Name, w.fonts.move, white, in(margin, 178, col, 212), dtSingleLine|dtEllipsis)
		w.text(dc, c.Cue, w.fonts.text, soft, in(margin, cueY, col, 272), dtWordBreak)
		w.text(dc, c.Note, w.fonts.note, soft, in(margin, progressY+22, col, cardH-20), dtWordBreak)
	}

	for _, b := range w.buttons() {
		ink := white
		if b.primary {
			ink = argbToRef(w.theme().ink)
		}
		w.text(dc, b.label, w.fonts.button, ink, in(int(b.r.left), int(b.r.top), int(b.r.right), int(b.r.bottom)), dtCenter|dtVCenter|dtSingleLine)
	}
}

func (w *Window) text(dc uintptr, s string, font uintptr, color uint32, r rect, flags uintptr) {
	if s == "" {
		return
	}
	procSelectObject.Call(dc, font)
	procSetTextColor.Call(dc, uintptr(color))
	p, _ := windows.UTF16PtrFromString(s)
	procDrawText.Call(dc, uintptr(unsafe.Pointer(p)), ^uintptr(0), uintptr(unsafe.Pointer(&r)), flags|dtNoPrefix)
}

// rgb turns 0xRRGGBB into the COLORREF GDI wants, 0x00BBGGRR.
func rgb(c uint32) uint32 {
	return (c&0xFF)<<16 | c&0xFF00 | (c>>16)&0xFF
}

func argbToRef(c uint32) uint32 { return rgb(c & 0xFFFFFF) }
