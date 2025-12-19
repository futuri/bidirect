//go:build windows

package window

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	gdi32                  = windows.NewLazySystemDLL("gdi32.dll")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
)

const (
	BI_RGB         = 0
	DIB_RGB_COLORS = 0
)

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]uint32
}

func createDIBSection(width, height int) (hdcMem windows.Handle, hBitmap windows.Handle, pixels []byte, stride int, err error) {
	hdcScreen := windows.Handle(0)

	hdcMemPtr, _, e := procCreateCompatibleDC.Call(uintptr(hdcScreen))
	if hdcMemPtr == 0 {
		return 0, 0, nil, 0, fmt.Errorf("CreateCompatibleDC failed: %v", e)
	}
	hdcMem = windows.Handle(hdcMemPtr)

	bmi := BITMAPINFO{
		BmiHeader: BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:       int32(width),
			BiHeight:      int32(-height), // top-down DIB
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: BI_RGB,
		},
	}

	var ppvBits unsafe.Pointer

	hBitmapPtr, _, e := procCreateDIBSection.Call(
		uintptr(hdcMem),
		uintptr(unsafe.Pointer(&bmi)),
		DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(&ppvBits)),
		0, 0,
	)
	if hBitmapPtr == 0 {
		procDeleteDC.Call(uintptr(hdcMem))
		return 0, 0, nil, 0, fmt.Errorf("CreateDIBSection failed: %v", e)
	}
	hBitmap = windows.Handle(hBitmapPtr)

	procSelectObject.Call(uintptr(hdcMem), hBitmapPtr)

	stride = width * 4
	byteCount := stride * height
	pixels = unsafe.Slice((*byte)(ppvBits), byteCount)

	return hdcMem, hBitmap, pixels, stride, nil
}

func destroyDIBSection(hdcMem windows.Handle, hBitmap windows.Handle) error {
	if hBitmap != 0 {
		procDeleteObject.Call(uintptr(hBitmap))
	}
	if hdcMem != 0 {
		procDeleteDC.Call(uintptr(hdcMem))
	}
	return nil
}
