package win

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/vusty/sitnag/internal/figure"
)

var (
	procGetDC     = user32.NewProc("GetDC")
	procReleaseDC = user32.NewProc("ReleaseDC")
	procGetDIBits = gdi32.NewProc("GetDIBits")
)

// TestRender paints every look of the card into PNGs, for checking the design
// by eye without putting a window on anyone's screen. It runs only with
// SITNAG_RENDER set to an output directory, on Windows:
//
//	GOOS=windows go test -c -o win.test.exe ./internal/win
//	SITNAG_RENDER=C:\temp\sitnag win.test.exe -test.run Render
func TestRender(t *testing.T) {
	dir := os.Getenv("SITNAG_RENDER")
	if dir == "" {
		t.Skip("SITNAG_RENDER not set")
	}
	// GDI batches drawing per thread; a goroutine that hops threads between
	// DrawText and GetDIBits reads the bitmap before the text lands in it.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := startGDIPlus(); err != nil {
		t.Fatal(err)
	}
	w := &Window{dpi: 96, labels: Labels{Start: "Start the warm-up", Short: "Snooze 5 min",
		Long: "Snooze 30 min", FactHeading: "WHILE YOU SIT"}}
	w.makeFonts()

	// The English strings from lang.go, for the README.
	views := map[string]view{
		"1-ask": {kind: asking, ask: Ask{Title: "Time to move", Hero: "1 h 02 min",
			Caption: "at the computer without a break. Stand up, walk, stretch: 5 min, then back to it.", Long: true}},
		"2-urgent": {kind: asking, ask: Ask{Title: "Enough sitting",
			Caption: "1 h 35 min at the computer without a break. Only a 5 min snooze is left.",
			Source:  "Beasley et al., Eur Respir J, 2003",
			Fact:    "Idle calves stop pumping blood back to the heart, and it pools in the veins of the lower legs. That is how clots form: embolism after long hours at a computer is in the medical journals."}},
		// The longest fact and the longest cue there are, in Russian: Cyrillic
		// runs wider, so if these fit everything does.
		"5-urgent-long": {kind: asking, ask: Ask{Title: "Хватит сидеть",
			Caption: "За компом уже 1 ч 35 мин без перерыва. Отложить можно только на 5 мин.",
			Source:  "Schmid & Leitzmann, J Natl Cancer Inst, 2014",
			Fact:    "Икры перестают качать кровь к сердцу, и она застаивается в венах голеней. Так образуются тромбы: тромбоэмболия после долгого сидения за компьютером описана в медицинских журналах."}},
		"6-warmup-long": {kind: countdown, cd: Countdown{Title: "Разминка", Clock: "4:58",
			Move: int(figure.ForwardBend), Name: "Наклоны в стороны", Cue: "Спина расслаблена, руки висят к полу. Задержись внизу, поднимайся плавно.",
			Note: "Отсчёт идёт, пока клавиатура и мышь не трогаются.", Progress: 0.02}},
		"7-paused-long": {kind: countdown, cd: Countdown{Title: "Разминка на паузе", Clock: "4:58", Move: int(figure.Squat),
			Cue: "Ты пользуешься компьютером, отсчёт стоит. Отойди, и он пойдёт дальше.", Paused: true, Progress: 0.02}},
		"3-warmup": {kind: countdown, cd: Countdown{Title: "Warm-up", Clock: "3:12",
			Move: int(figure.SideBend), Name: "Side bends", Cue: "One arm over the head, reach sideways after it. Hips stay where they are.",
			Note: "The clock runs while the keyboard and mouse are left alone.", Progress: 0.36}},
		"4-paused": {kind: countdown, cd: Countdown{Title: "Warm-up paused", Clock: "3:12", Move: int(figure.SideBend),
			Cue: "You are using the computer, so the clock stands still. Step away and it goes on.", Paused: true, Progress: 0.36}},
	}
	for name, v := range views {
		w.shown = v
		w.hover = 0
		w.moveHeld = 1200 * time.Millisecond
		if err := saveCard(w, filepath.Join(dir, name+".png")); err != nil {
			t.Fatal(err)
		}
	}
}

// TestRenderMoves paints every move of the mannequin as a strip of frames,
// one PNG per move, for checking a pose by eye. Same switch as TestRender.
func TestRenderMoves(t *testing.T) {
	dir := os.Getenv("SITNAG_RENDER")
	if dir == "" {
		t.Skip("SITNAG_RENDER not set")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := startGDIPlus(); err != nil {
		t.Fatal(err)
	}
	const frames, cellW, cellH = 8, 190, 250
	w := &Window{dpi: 96}
	for m := 0; m < figure.Count; m++ {
		draw := func(dc uintptr) {
			g, err := newGraphics(dc)
			if err != nil {
				t.Fatal(err)
			}
			defer g.close()
			g.gradient(0, 0, frames*cellW, cellH, themeBreak.from, themeBreak.to)
			for i := 0; i < frames; i++ {
				// A repetition never takes longer than eight seconds.
				at := time.Duration(i) * time.Second
				w.drawFigure(g, figure.At(m, at), float64(i*cellW+cellW/2), cellH-14, 1)
			}
		}
		if err := savePNG(filepath.Join(dir, fmt.Sprintf("move-%02d.png", m)), frames*cellW, cellH, draw); err != nil {
			t.Fatal(err)
		}
	}
}

func saveCard(w *Window, path string) error {
	return savePNG(path, w.px(cardW), w.px(cardH), func(dc uintptr) {
		w.drawShapes(dc)
		w.drawText(dc)
	})
}

func savePNG(path string, width, height int, draw func(dc uintptr)) error {
	screen, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screen)
	mem, _, _ := procCreateCompatibleDC.Call(screen)
	defer procDeleteDC.Call(mem)
	bmp, _, _ := procCreateCompatibleBitmap.Call(screen, uintptr(width), uintptr(height))
	defer procDeleteObject.Call(bmp)
	old, _, _ := procSelectObject.Call(mem, bmp)
	draw(mem)
	procSelectObject.Call(mem, old)

	info := struct {
		size                   uint32
		width, height          int32
		planes, bitCount       uint16
		compression, sizeImage uint32
		xPels, yPels           int32
		clrUsed, clrImportant  uint32
		colors                 [4]uint32
	}{width: int32(width), height: -int32(height), planes: 1, bitCount: 32}
	info.size = 40
	pix := make([]byte, width*height*4)
	procGetDIBits.Call(mem, bmp, 0, uintptr(height), uintptr(unsafe.Pointer(&pix[0])), uintptr(unsafe.Pointer(&info)), 0)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < width*height; i++ {
		img.Set(i%width, i/width, color.RGBA{pix[4*i+2], pix[4*i+1], pix[4*i], 255})
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
