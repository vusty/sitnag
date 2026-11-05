package win

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"
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
			Fact:    "Idle calves stop pumping blood back to the heart, and it pools in the veins of the lower legs. That is how clots form: gamers have had pulmonary embolisms after marathon sessions."}},
		"3-warmup": {kind: countdown, cd: Countdown{Title: "Warm-up", Clock: "3:12",
			Note: "The clock runs while the keyboard and mouse are left alone.", Progress: 0.36}},
		"4-paused": {kind: countdown, cd: Countdown{Title: "Warm-up paused", Clock: "3:12",
			Note: "You are using the computer, so the clock stands still. Step away and it goes on.", Paused: true, Progress: 0.36}},
	}
	for name, v := range views {
		w.shown = v
		w.hover = 0
		if err := savePNG(w, filepath.Join(dir, name+".png")); err != nil {
			t.Fatal(err)
		}
	}
}

func savePNG(w *Window, path string) error {
	width, height := w.px(cardW), w.px(cardH)
	screen, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screen)
	mem, _, _ := procCreateCompatibleDC.Call(screen)
	defer procDeleteDC.Call(mem)
	bmp, _, _ := procCreateCompatibleBitmap.Call(screen, uintptr(width), uintptr(height))
	defer procDeleteObject.Call(bmp)
	old, _, _ := procSelectObject.Call(mem, bmp)
	w.drawShapes(mem)
	w.drawText(mem)
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
