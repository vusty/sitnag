package win

import (
	"fmt"
	"math"
	"sync"
	"unsafe"
)

// GDI+ draws what plain GDI cannot: antialiased rounded corners, see-through
// fills, and a smooth gradient. Only its integer entry points are used, plus
// the arc angles, which sit past the fourth argument and so travel on the
// stack, where a float32 is just its bits in the low half of a slot.

var (
	procGdiplusStartup           = gdiplus.NewProc("GdiplusStartup")
	procGdipCreateFromHDC        = gdiplus.NewProc("GdipCreateFromHDC")
	procGdipDeleteGraphics       = gdiplus.NewProc("GdipDeleteGraphics")
	procGdipSetSmoothingMode     = gdiplus.NewProc("GdipSetSmoothingMode")
	procGdipCreateLineBrushRectI = gdiplus.NewProc("GdipCreateLineBrushFromRectI")
	procGdipCreateSolidFill      = gdiplus.NewProc("GdipCreateSolidFill")
	procGdipDeleteBrush          = gdiplus.NewProc("GdipDeleteBrush")
	procGdipFillRectangleI       = gdiplus.NewProc("GdipFillRectangleI")
	procGdipFillEllipseI         = gdiplus.NewProc("GdipFillEllipseI")
	procGdipCreatePath           = gdiplus.NewProc("GdipCreatePath")
	procGdipDeletePath           = gdiplus.NewProc("GdipDeletePath")
	procGdipAddPathArcI          = gdiplus.NewProc("GdipAddPathArcI")
	procGdipClosePathFigure      = gdiplus.NewProc("GdipClosePathFigure")
	procGdipFillPath             = gdiplus.NewProc("GdipFillPath")
)

const (
	smoothingAntiAlias   = 4
	gradientDiagonal     = 2 // LinearGradientModeForwardDiagonal
	wrapTileFlipXY       = 3 // no seam where the brush would repeat
	fillModeAlternate    = 0
	gdiplusStatusOK      = 0
	gdiplusStartupInputV = 1
)

var (
	gdiplusOnce sync.Once
	gdiplusErr  error
)

func startGDIPlus() error {
	gdiplusOnce.Do(func() {
		input := struct {
			version        uint32
			debugCallback  uintptr
			suppressThread int32
			suppressCodecs int32
		}{version: gdiplusStartupInputV}
		var token uintptr
		if r, _, _ := procGdiplusStartup.Call(uintptr(unsafe.Pointer(&token)), uintptr(unsafe.Pointer(&input)), 0); r != gdiplusStatusOK {
			gdiplusErr = fmt.Errorf("GdiplusStartup: status %d", r)
		}
	})
	return gdiplusErr
}

type graphics struct{ g uintptr }

func newGraphics(hdc uintptr) (*graphics, error) {
	var g uintptr
	if r, _, _ := procGdipCreateFromHDC.Call(hdc, uintptr(unsafe.Pointer(&g))); r != gdiplusStatusOK {
		return nil, fmt.Errorf("GdipCreateFromHDC: status %d", r)
	}
	procGdipSetSmoothingMode.Call(g, smoothingAntiAlias)
	return &graphics{g}, nil
}

// close has to come before plain GDI draws on the same DC again.
func (g *graphics) close() { procGdipDeleteGraphics.Call(g.g) }

func (g *graphics) gradient(x, y, w, h int, from, to uint32) {
	r := struct{ x, y, w, h int32 }{int32(x), int32(y), int32(w), int32(h)}
	var brush uintptr
	if s, _, _ := procGdipCreateLineBrushRectI.Call(uintptr(unsafe.Pointer(&r)),
		uintptr(from), uintptr(to), gradientDiagonal, wrapTileFlipXY,
		uintptr(unsafe.Pointer(&brush))); s != gdiplusStatusOK {
		return
	}
	defer procGdipDeleteBrush.Call(brush)
	procGdipFillRectangleI.Call(g.g, brush, uintptr(x), uintptr(y), uintptr(w), uintptr(h))
}

func (g *graphics) solid(argb uint32) (uintptr, bool) {
	var brush uintptr
	s, _, _ := procGdipCreateSolidFill.Call(uintptr(argb), uintptr(unsafe.Pointer(&brush)))
	return brush, s == gdiplusStatusOK
}

func (g *graphics) ellipse(x, y, w, h int, argb uint32) {
	brush, ok := g.solid(argb)
	if !ok {
		return
	}
	defer procGdipDeleteBrush.Call(brush)
	procGdipFillEllipseI.Call(g.g, brush, uintptr(x), uintptr(y), uintptr(w), uintptr(h))
}

func (g *graphics) roundRect(x, y, w, h, r int, argb uint32) {
	brush, ok := g.solid(argb)
	if !ok {
		return
	}
	defer procGdipDeleteBrush.Call(brush)
	var path uintptr
	if s, _, _ := procGdipCreatePath.Call(fillModeAlternate, uintptr(unsafe.Pointer(&path))); s != gdiplusStatusOK {
		return
	}
	defer procGdipDeletePath.Call(path)

	d := min(2*r, w, h)
	arc := func(ax, ay int, start float32) {
		procGdipAddPathArcI.Call(path, uintptr(ax), uintptr(ay), uintptr(d), uintptr(d),
			uintptr(math.Float32bits(start)), uintptr(math.Float32bits(90)))
	}
	arc(x, y, 180)
	arc(x+w-d, y, 270)
	arc(x+w-d, y+h-d, 0)
	arc(x, y+h-d, 90)
	procGdipClosePathFigure.Call(path)
	procGdipFillPath.Call(g.g, brush, path)
}
