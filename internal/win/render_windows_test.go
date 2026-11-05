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
	w := &Window{dpi: 96, labels: Labels{Start: "Начать разминку", Short: "Отложить на 5 мин", Long: "Отложить на 30 мин"}}
	w.makeFonts()

	views := map[string]view{
		"1-calm": {kind: asking, ask: Ask{Title: "Пора размяться", Hero: "1 ч 02 мин",
			Caption: "за компом без перерыва. Встань, пройдись, потянись: 5 мин, и снова за дело.", Long: true}},
		"2-urgent": {kind: asking, ask: Ask{Title: "Хватит сидеть",
			Caption: "За компом уже 1 ч 35 мин без перерыва. Отложить можно только на 5 мин.",
			Fact:    "Икры не качают кровь обратно к сердцу, и она застаивается в венах голеней. Так образуются тромбы: известны случаи тромбоэмболии у геймеров после многочасовых сессий."}},
		"3-break": {kind: countdown, cd: Countdown{Title: "Разминка", Clock: "3:12",
			Note: "Отсчёт идёт, пока клавиатура и мышь не трогаются.", Progress: 0.36}},
		"4-paused": {kind: countdown, cd: Countdown{Title: "Разминка на паузе", Clock: "3:12",
			Note: "Ты пользуешься компьютером, отсчёт стоит. Отойди, и он пойдёт дальше.", Paused: true, Progress: 0.36}},
	}
	for name, v := range views {
		w.shown = v
		w.hover = 0
		if name == "1-calm" {
			w.hover = Short
		}
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
