// Package win is the part of sitnag that talks to Windows: how long the
// keyboard and mouse have been idle, the system sounds, and the one window
// the tool shows.
package win

import (
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	gdiplus  = windows.NewLazySystemDLL("gdiplus.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")

	procGetLastInputInfo       = user32.NewProc("GetLastInputInfo")
	procMessageBeep            = user32.NewProc("MessageBeep")
	procSetProcessDPIAware     = user32.NewProc("SetProcessDPIAware")
	procGetDpiForSystem        = user32.NewProc("GetDpiForSystem")
	procRegisterClassEx        = user32.NewProc("RegisterClassExW")
	procCreateWindowEx         = user32.NewProc("CreateWindowExW")
	procDefWindowProc          = user32.NewProc("DefWindowProcW")
	procGetMessage             = user32.NewProc("GetMessageW")
	procTranslateMessage       = user32.NewProc("TranslateMessage")
	procDispatchMessage        = user32.NewProc("DispatchMessageW")
	procPostMessage            = user32.NewProc("PostMessageW")
	procSendMessage            = user32.NewProc("SendMessageW")
	procShowWindow             = user32.NewProc("ShowWindow")
	procSetWindowPos           = user32.NewProc("SetWindowPos")
	procSystemParametersInfo   = user32.NewProc("SystemParametersInfoW")
	procLoadCursor             = user32.NewProc("LoadCursorW")
	procSetCursor              = user32.NewProc("SetCursor")
	procBeginPaint             = user32.NewProc("BeginPaint")
	procEndPaint               = user32.NewProc("EndPaint")
	procInvalidateRect         = user32.NewProc("InvalidateRect")
	procDrawText               = user32.NewProc("DrawTextW")
	procSetCapture             = user32.NewProc("SetCapture")
	procReleaseCapture         = user32.NewProc("ReleaseCapture")
	procTrackMouseEvent        = user32.NewProc("TrackMouseEvent")
	procGetTickCount           = kernel32.NewProc("GetTickCount")
	procGetModuleHandle        = kernel32.NewProc("GetModuleHandleW")
	procCreateFont             = gdi32.NewProc("CreateFontW")
	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procSetBkMode              = gdi32.NewProc("SetBkMode")
	procSetTextColor           = gdi32.NewProc("SetTextColor")

	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

type lastInputInfo struct {
	size uint32
	time uint32
}

// Idle is how long since the last keyboard or mouse input in this session.
// A locked screen reads as idle too, which is what it should be.
func Idle() time.Duration {
	lii := lastInputInfo{size: uint32(unsafe.Sizeof(lastInputInfo{}))}
	if r, _, _ := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&lii))); r == 0 {
		return 0
	}
	now, _, _ := procGetTickCount.Call()
	// Both are GetTickCount readings; unsigned subtraction survives the wrap
	// every 49.7 days.
	return time.Duration(uint32(now)-lii.time) * time.Millisecond
}

// Sound picks one of the system sounds.
type Sound uintptr

const (
	// Attention is the "exclamation" sound, for the reminder.
	Attention Sound = 0x30
	// Done is the "asterisk" sound, for the end of the countdown.
	Done Sound = 0x40
)

// Beep plays a system sound. It is silent if the sound scheme is.
func Beep(s Sound) {
	procMessageBeep.Call(uintptr(s))
}

// DPIAware stops Windows from stretching the window into a blur on a scaled
// display. It has to run before the first window in the process exists, the
// tray's included.
func DPIAware() {
	procSetProcessDPIAware.Call()
}

func systemDPI() int {
	if procGetDpiForSystem.Find() != nil {
		return 96
	}
	if dpi, _, _ := procGetDpiForSystem.Call(); dpi != 0 {
		return int(dpi)
	}
	return 96
}
