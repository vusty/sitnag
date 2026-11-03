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
	// Not a system DLL on purpose: loaded by bare name, comctl32 goes through
	// side-by-side redirection to version 6, the one that draws themed buttons.
	// A full system32 path would bypass that and give the Windows 95 look.
	comctl32 = windows.NewLazyDLL("comctl32.dll")

	procGetLastInputInfo     = user32.NewProc("GetLastInputInfo")
	procMessageBeep          = user32.NewProc("MessageBeep")
	procSetProcessDPIAware   = user32.NewProc("SetProcessDPIAware")
	procGetDpiForSystem      = user32.NewProc("GetDpiForSystem")
	procRegisterClassEx      = user32.NewProc("RegisterClassExW")
	procCreateWindowEx       = user32.NewProc("CreateWindowExW")
	procDefWindowProc        = user32.NewProc("DefWindowProcW")
	procGetMessage           = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessage      = user32.NewProc("DispatchMessageW")
	procPostMessage          = user32.NewProc("PostMessageW")
	procSendMessage          = user32.NewProc("SendMessageW")
	procShowWindow           = user32.NewProc("ShowWindow")
	procSetWindowPos         = user32.NewProc("SetWindowPos")
	procSetWindowText        = user32.NewProc("SetWindowTextW")
	procGetWindowRect        = user32.NewProc("GetWindowRect")
	procAdjustWindowRectEx   = user32.NewProc("AdjustWindowRectEx")
	procSystemParametersInfo = user32.NewProc("SystemParametersInfoW")
	procLoadCursor           = user32.NewProc("LoadCursorW")
	procGetTickCount         = kernel32.NewProc("GetTickCount")
	procGetModuleHandle      = kernel32.NewProc("GetModuleHandleW")
	procCreateActCtx         = kernel32.NewProc("CreateActCtxW")
	procActivateActCtx       = kernel32.NewProc("ActivateActCtx")
	procCreateFont           = gdi32.NewProc("CreateFontW")
	procInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")
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

type actCtx struct {
	size                  uint32
	flags                 uint32
	source                *uint16
	processorArchitecture uint16
	langID                uint16
	assemblyDirectory     *uint16
	resourceName          uintptr
	applicationName       *uint16
	module                windows.Handle
}

// enableVisualStyles activates, on the calling thread, the comctl32 v6
// manifest that shell32.dll carries as resource 124. Without a manifest of its
// own an executable gets classic grey buttons; this borrows the system's. If it
// fails the window still works, just in the old look.
func enableVisualStyles() {
	dir, err := windows.GetSystemDirectory()
	if err != nil {
		return
	}
	ctx := actCtx{
		flags:             0x4 | 0x8, // ASSEMBLY_DIRECTORY_VALID | RESOURCE_NAME_VALID
		source:            windows.StringToUTF16Ptr(dir + `\shell32.dll`),
		assemblyDirectory: windows.StringToUTF16Ptr(dir),
		resourceName:      124,
	}
	ctx.size = uint32(unsafe.Sizeof(ctx))
	h, _, _ := procCreateActCtx.Call(uintptr(unsafe.Pointer(&ctx)))
	if h == uintptr(windows.InvalidHandle) {
		return
	}
	var cookie uintptr
	if r, _, _ := procActivateActCtx.Call(h, uintptr(unsafe.Pointer(&cookie))); r == 0 {
		return
	}
	icc := struct{ size, classes uint32 }{8, 0x4000} // ICC_STANDARD_CLASSES
	if procInitCommonControlsEx.Find() == nil {
		procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))
	}
}
