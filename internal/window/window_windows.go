//go:build windows

package window

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/example/bidirect/internal/config"
	"github.com/example/bidirect/internal/websocket"
	"golang.org/x/sys/windows"
)

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procUpdateWindow        = user32.NewProc("UpdateWindow")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procLoadCursorW         = user32.NewProc("LoadCursorW")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procScreenToClient      = user32.NewProc("ScreenToClient")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	procUpdateLayeredWindow = user32.NewProc("UpdateLayeredWindow")
	procMessageBoxW         = user32.NewProc("MessageBoxW")
)

const (
	WS_POPUP        = 0x80000000
	WS_VISIBLE      = 0x10000000
	WS_EX_LAYERED   = 0x00080000
	WS_EX_APPWINDOW = 0x00040000
	WS_EX_TOPMOST   = 0x00000008

	WM_DESTROY       = 0x0002
	WM_SIZE          = 0x0005
	WM_PAINT         = 0x000F
	WM_CLOSE         = 0x0010
	WM_NCHITTEST     = 0x0084
	WM_NCRBUTTONUP   = 0x00A5
	WM_GETMINMAXINFO = 0x0024
	WM_SIZING        = 0x0214
	WM_COMMAND       = 0x0111
	WM_RBUTTONUP     = 0x0205

	HTTRANSPARENT = ^uintptr(0)
	HTCLIENT      = 1
	HTCAPTION     = 2
	HTLEFT        = 10
	HTRIGHT       = 11
	HTTOP         = 12
	HTTOPLEFT     = 13
	HTTOPRIGHT    = 14
	HTBOTTOM      = 15
	HTBOTTOMLEFT  = 16
	HTBOTTOMRIGHT = 17

	SW_SHOW = 5

	IDC_ARROW = 32512

	SM_CXSCREEN = 0
	SM_CYSCREEN = 1

	SWP_NOMOVE   = 0x0002
	SWP_NOSIZE   = 0x0001
	HWND_TOPMOST = ^uintptr(0)

	MF_STRING     = 0x0000
	MF_SEPARATOR  = 0x0800
	TPM_LEFTALIGN = 0x0000
	TPM_RETURNCMD = 0x0100

	IDM_QUIT       = 1001
	IDM_ALWAYS_TOP = 1003
	IDM_ABOUT      = 1004

	MB_OK       = 0x00000000
	MB_ICONINFO = 0x00000040

	AC_SRC_OVER  = 0x00
	AC_SRC_ALPHA = 0x01
	ULW_ALPHA    = 0x02

	WMSZ_LEFT        = 1
	WMSZ_RIGHT       = 2
	WMSZ_TOP         = 3
	WMSZ_TOPLEFT     = 4
	WMSZ_TOPRIGHT    = 5
	WMSZ_BOTTOM      = 6
	WMSZ_BOTTOMLEFT  = 7
	WMSZ_BOTTOMRIGHT = 8
)

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type MSG struct {
	Hwnd    windows.HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type POINT struct {
	X, Y int32
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

type MINMAXINFO struct {
	PtReserved     POINT
	PtMaxSize      POINT
	PtMaxPosition  POINT
	PtMinTrackSize POINT
	PtMaxTrackSize POINT
}

type BLENDFUNCTION struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type SIZE struct {
	CX, CY int32
}

type Window struct {
	hwnd      windows.HWND
	cfg       config.Config
	width     int
	height    int
	hdcMem    windows.Handle
	hBitmap   windows.Handle
	dibPixels []byte
	stride    int
	mu        sync.RWMutex
	isTopmost bool
	quitCh    chan struct{}
	wsServer  *websocket.Server
}

var windowInstance *Window
var windowInstanceMu sync.Mutex

func NewWindow(cfg config.Config) (*Window, error) {
	width := cfg.InitialSize
	height := cfg.InitialSize

	w := &Window{
		cfg:    cfg,
		width:  width,
		height: height,
		quitCh: make(chan struct{}),
	}
	return w, nil
}

func (w *Window) Run() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	windowInstanceMu.Lock()
	windowInstance = w
	windowInstanceMu.Unlock()

	className, _ := syscall.UTF16PtrFromString("HeartWindowClass")
	windowTitle, _ := syscall.UTF16PtrFromString(w.cfg.WindowTitle)

	hInstance := windows.Handle(0)

	cursor, _, _ := procLoadCursorW.Call(0, uintptr(IDC_ARROW))

	wndProc := syscall.NewCallback(wndProcCallback)

	wcx := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         0,
		LpfnWndProc:   wndProc,
		HInstance:     hInstance,
		HCursor:       windows.Handle(cursor),
		LpszClassName: className,
	}

	ret, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wcx)))
	if ret == 0 {
		return fmt.Errorf("RegisterClassExW failed: %v", err)
	}

	screenW, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	screenH, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)

	x := (int(screenW) - w.width) / 2
	y := (int(screenH) - w.height) / 2

	exStyle := uint32(WS_EX_LAYERED | WS_EX_APPWINDOW)
	style := uint32(WS_POPUP | WS_VISIBLE)

	hwnd, _, err := procCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		uintptr(style),
		uintptr(x), uintptr(y),
		uintptr(w.width), uintptr(w.height),
		0, 0,
		uintptr(hInstance),
		0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW failed: %v", err)
	}

	w.hwnd = windows.HWND(hwnd)

	var dibErr error
	w.hdcMem, w.hBitmap, w.dibPixels, w.stride, dibErr = createDIBSection(w.width, w.height)
	if dibErr != nil {
		return fmt.Errorf("createDIBSection failed: %v", dibErr)
	}

	// Show initial logo
	logo := websocket.CreateBiDirectLogo(w.width)
	w.applyFrameDirect(logo, w.width, w.height)

	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)

	// Start WebSocket server
	w.wsServer = websocket.NewServer(w.cfg.WSPort)
	if err := w.wsServer.Start(); err != nil {
		return fmt.Errorf("WebSocket server failed: %v", err)
	}
	go w.wsRenderLoop()

	var msg MSG
	for {
		ret, _, _ := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&msg)),
			0, 0, 0,
		)
		if ret == 0 || int32(ret) == -1 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}

	destroyDIBSection(w.hdcMem, w.hBitmap)
	if w.wsServer != nil {
		w.wsServer.Stop()
	}
	close(w.quitCh)

	return nil
}

func (w *Window) applyFrame(frame []byte, width, height int) error {
	w.mu.Lock()
	copy(w.dibPixels, frame)
	w.mu.Unlock()

	srcPt := POINT{0, 0}
	sz := SIZE{int32(width), int32(height)}
	blend := BLENDFUNCTION{
		BlendOp:             AC_SRC_OVER,
		BlendFlags:          0,
		SourceConstantAlpha: 255,
		AlphaFormat:         AC_SRC_ALPHA,
	}

	ret, _, err := procUpdateLayeredWindow.Call(
		uintptr(w.hwnd),
		0,
		0,
		uintptr(unsafe.Pointer(&sz)),
		uintptr(w.hdcMem),
		uintptr(unsafe.Pointer(&srcPt)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		ULW_ALPHA,
	)
	if ret == 0 {
		return fmt.Errorf("UpdateLayeredWindow failed: %v", err)
	}
	return nil
}

func wndProcCallback(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	windowInstanceMu.Lock()
	w := windowInstance
	windowInstanceMu.Unlock()

	switch msg {
	case WM_NCHITTEST:
		if w != nil {
			return w.handleNCHitTest(lParam)
		}
	case WM_GETMINMAXINFO:
		if w != nil {
			w.handleGetMinMaxInfo(lParam)
			return 0
		}
	case WM_SIZING:
		if w != nil {
			w.handleSizing(wParam, lParam)
			return 1
		}
	case WM_SIZE:
		if w != nil {
			newWidth := int(lParam & 0xFFFF)
			newHeight := int((lParam >> 16) & 0xFFFF)
			if newWidth > 0 && newHeight > 0 && (newWidth != w.width || newHeight != w.height) {
				w.resize(newWidth, newHeight)
			}
		}
	case WM_RBUTTONUP:
		if w != nil {
			w.showContextMenu()
			return 0
		}
	case WM_NCRBUTTONUP:
		if w != nil {
			w.showContextMenu()
			return 0
		}
	case WM_COMMAND:
		if w != nil {
			w.handleCommand(int(wParam & 0xFFFF))
			return 0
		}
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func (w *Window) handleNCHitTest(lParam uintptr) uintptr {
	x := int32(int16(lParam & 0xFFFF))
	y := int32(int16((lParam >> 16) & 0xFFFF))

	pt := POINT{X: x, Y: y}
	procScreenToClient.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(&pt)))

	cx, cy := int(pt.X), int(pt.Y)

	if cx < 0 || cy < 0 || cx >= w.width || cy >= w.height {
		return HTTRANSPARENT
	}

	w.mu.RLock()
	idx := (cy*w.stride + cx*4) + 3
	alpha := uint8(0)
	if idx < len(w.dibPixels) {
		alpha = w.dibPixels[idx]
	}
	w.mu.RUnlock()

	if alpha < w.cfg.AlphaThreshold {
		return HTTRANSPARENT
	}

	grab := w.cfg.BorderGrabSize

	onLeft := cx < grab
	onRight := cx >= w.width-grab
	onTop := cy < grab
	onBottom := cy >= w.height-grab

	if onTop && onLeft {
		return HTTOPLEFT
	}
	if onTop && onRight {
		return HTTOPRIGHT
	}
	if onBottom && onLeft {
		return HTBOTTOMLEFT
	}
	if onBottom && onRight {
		return HTBOTTOMRIGHT
	}
	if onLeft {
		return HTLEFT
	}
	if onRight {
		return HTRIGHT
	}
	if onTop {
		return HTTOP
	}
	if onBottom {
		return HTBOTTOM
	}

	return HTCAPTION
}

func (w *Window) handleGetMinMaxInfo(lParam uintptr) {
	mmi := (*MINMAXINFO)(unsafe.Pointer(lParam))

	screenW, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	screenH, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)

	maxW := int(screenW) - 50
	maxH := int(screenH) - 50

	mmi.PtMinTrackSize.X = int32(w.cfg.MinSize)
	mmi.PtMinTrackSize.Y = int32(w.cfg.MinSize)
	mmi.PtMaxTrackSize.X = int32(maxW)
	mmi.PtMaxTrackSize.Y = int32(maxH)
}

func (w *Window) handleSizing(wParam, lParam uintptr) {
	if !w.cfg.KeepAspect {
		return
	}

	rect := (*RECT)(unsafe.Pointer(lParam))
	width := rect.Right - rect.Left
	height := rect.Bottom - rect.Top

	switch wParam {
	case WMSZ_LEFT, WMSZ_RIGHT:
		rect.Bottom = rect.Top + width
	case WMSZ_TOP, WMSZ_BOTTOM:
		rect.Right = rect.Left + height
	case WMSZ_TOPLEFT:
		if width > height {
			rect.Left = rect.Right - height
		} else {
			rect.Top = rect.Bottom - width
		}
	case WMSZ_TOPRIGHT:
		if width > height {
			rect.Right = rect.Left + height
		} else {
			rect.Top = rect.Bottom - width
		}
	case WMSZ_BOTTOMLEFT:
		if width > height {
			rect.Left = rect.Right - height
		} else {
			rect.Bottom = rect.Top + width
		}
	case WMSZ_BOTTOMRIGHT:
		if width > height {
			rect.Right = rect.Left + height
		} else {
			rect.Bottom = rect.Top + width
		}
	}
}

func (w *Window) resize(newWidth, newHeight int) {
	destroyDIBSection(w.hdcMem, w.hBitmap)

	w.width = newWidth
	w.height = newHeight

	var err error
	w.hdcMem, w.hBitmap, w.dibPixels, w.stride, err = createDIBSection(w.width, w.height)
	if err != nil {
		return
	}
}

func (w *Window) showContextMenu() {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}

	var topText string
	if w.isTopmost {
		topText = "✓ Always On Top"
	} else {
		topText = "Always On Top"
	}
	alwaysTop, _ := syscall.UTF16PtrFromString(topText)
	about, _ := syscall.UTF16PtrFromString("About")
	quit, _ := syscall.UTF16PtrFromString("Quit")

	procAppendMenuW.Call(hMenu, MF_STRING, IDM_ALWAYS_TOP, uintptr(unsafe.Pointer(alwaysTop)))
	procAppendMenuW.Call(hMenu, MF_STRING, IDM_ABOUT, uintptr(unsafe.Pointer(about)))
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	procAppendMenuW.Call(hMenu, MF_STRING, IDM_QUIT, uintptr(unsafe.Pointer(quit)))

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	procSetForegroundWindow.Call(uintptr(w.hwnd))

	cmd, _, _ := procTrackPopupMenu.Call(
		hMenu,
		TPM_LEFTALIGN|TPM_RETURNCMD,
		uintptr(pt.X), uintptr(pt.Y),
		0, uintptr(w.hwnd), 0,
	)

	procDestroyMenu.Call(hMenu)

	if cmd != 0 {
		w.handleCommand(int(cmd))
	}
}

func (w *Window) handleCommand(id int) {
	switch id {
	case IDM_QUIT:
		procPostQuitMessage.Call(0)
	case IDM_ALWAYS_TOP:
		w.toggleAlwaysOnTop()
	case IDM_ABOUT:
		w.showAbout()
	}
}

func (w *Window) showAbout() {
	title, _ := syscall.UTF16PtrFromString("About BiDirect")
	msg, _ := syscall.UTF16PtrFromString("BiDirect v1.0\n\nWebSocket streaming receiver with per-pixel alpha transparency.\n\nReceives WebP/PNG/JPEG images via WebSocket and displays them as a borderless overlay window.")
	procMessageBoxW.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(msg)), uintptr(unsafe.Pointer(title)), MB_OK|MB_ICONINFO)
}

func (w *Window) toggleAlwaysOnTop() {
	w.isTopmost = !w.isTopmost

	var hwndInsertAfter uintptr
	if w.isTopmost {
		hwndInsertAfter = HWND_TOPMOST
	} else {
		hwndInsertAfter = ^uintptr(1)
	}

	procSetWindowPos.Call(
		uintptr(w.hwnd),
		hwndInsertAfter,
		0, 0, 0, 0,
		SWP_NOMOVE|SWP_NOSIZE,
	)
}

func (w *Window) wsRenderLoop() {
	ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
	defer ticker.Stop()

	ringBuffer := w.wsServer.GetRingBuffer()

	for {
		select {
		case <-w.quitCh:
			return
		case <-ticker.C:
			frame, ok := ringBuffer.ReadLatest()
			if !ok {
				continue
			}

			if frame.Width != w.width || frame.Height != w.height {
				w.resizeWindow(frame.Width, frame.Height)
			}

			w.applyFrameDirect(frame.Data, frame.Width, frame.Height)
		}
	}
}

func (w *Window) resizeWindow(newWidth, newHeight int) {
	destroyDIBSection(w.hdcMem, w.hBitmap)

	w.width = newWidth
	w.height = newHeight

	var err error
	w.hdcMem, w.hBitmap, w.dibPixels, w.stride, err = createDIBSection(w.width, w.height)
	if err != nil {
		return
	}

	procSetWindowPos.Call(
		uintptr(w.hwnd),
		0,
		0, 0,
		uintptr(w.width), uintptr(w.height),
		SWP_NOMOVE,
	)
}

func (w *Window) applyFrameDirect(frame []byte, width, height int) error {
	expectedSize := width * height * 4
	if len(frame) < expectedSize || len(w.dibPixels) < expectedSize {
		return nil
	}

	w.mu.Lock()
	copy(w.dibPixels, frame[:expectedSize])
	w.mu.Unlock()

	srcPt := POINT{0, 0}
	sz := SIZE{int32(width), int32(height)}
	blend := BLENDFUNCTION{
		BlendOp:             AC_SRC_OVER,
		BlendFlags:          0,
		SourceConstantAlpha: 255,
		AlphaFormat:         AC_SRC_ALPHA,
	}

	procUpdateLayeredWindow.Call(
		uintptr(w.hwnd),
		0,
		0,
		uintptr(unsafe.Pointer(&sz)),
		uintptr(w.hdcMem),
		uintptr(unsafe.Pointer(&srcPt)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		ULW_ALPHA,
	)

	return nil
}
