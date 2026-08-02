//go:build windows

package clipboard

import (
	"context"
	"fmt"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

const (
	cfUnicodeText = 13
	gmemeMoveable = 0x0002
)

var (
	user32                = syscall.NewLazyDLL("user32.dll")
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard     = user32.NewProc("OpenClipboard")
	procCloseClipboard    = user32.NewProc("CloseClipboard")
	procEmptyClipboard    = user32.NewProc("EmptyClipboard")
	procGetClipboardData  = user32.NewProc("GetClipboardData")
	procSetClipboardData  = user32.NewProc("SetClipboardData")
	procIsClipboardFormat = user32.NewProc("IsClipboardFormatAvailable")
	procGlobalAlloc       = kernel32.NewProc("GlobalAlloc")
	procGlobalLock        = kernel32.NewProc("GlobalLock")
	procGlobalUnlock      = kernel32.NewProc("GlobalUnlock")
	procGlobalFree        = kernel32.NewProc("GlobalFree")
	procGetSequence       = user32.NewProc("GetClipboardSequenceNumber")
)

type platformClipboard struct{}

func pointerFromAddress(address uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&address))
}

func (platformClipboard) ReadText() (string, error) {
	if ret, _, _ := procOpenClipboard.Call(0); ret == 0 {
		return "", fmt.Errorf("OpenClipboard failed")
	}
	defer procCloseClipboard.Call()
	if ret, _, _ := procIsClipboardFormat.Call(cfUnicodeText); ret == 0 {
		return "", fmt.Errorf("clipboard has no Unicode text")
	}
	handle, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if handle == 0 {
		return "", fmt.Errorf("GetClipboardData failed")
	}
	ptr, _, _ := procGlobalLock.Call(handle)
	if ptr == 0 {
		return "", fmt.Errorf("GlobalLock failed")
	}
	defer procGlobalUnlock.Call(handle)
	words := unsafe.Slice((*uint16)(pointerFromAddress(ptr)), 1<<20)
	length := 0
	for length < len(words) && words[length] != 0 {
		length++
	}
	return string(utf16.Decode(words[:length])), nil
}

func (platformClipboard) WriteText(value string) error {
	encoded := utf16.Encode([]rune(value + "\x00"))
	size := uintptr(len(encoded) * 2)
	handle, _, _ := procGlobalAlloc.Call(gmemeMoveable, size)
	if handle == 0 {
		return fmt.Errorf("GlobalAlloc failed")
	}
	ptr, _, _ := procGlobalLock.Call(handle)
	if ptr == 0 {
		procGlobalFree.Call(handle)
		return fmt.Errorf("GlobalLock failed")
	}
	copy(unsafe.Slice((*uint16)(pointerFromAddress(ptr)), len(encoded)), encoded)
	procGlobalUnlock.Call(handle)
	if ret, _, _ := procOpenClipboard.Call(0); ret == 0 {
		procGlobalFree.Call(handle)
		return fmt.Errorf("OpenClipboard failed")
	}
	defer procCloseClipboard.Call()
	if ret, _, _ := procEmptyClipboard.Call(); ret == 0 {
		procGlobalFree.Call(handle)
		return fmt.Errorf("EmptyClipboard failed")
	}
	if ret, _, _ := procSetClipboardData.Call(cfUnicodeText, handle); ret == 0 {
		procGlobalFree.Call(handle)
		return fmt.Errorf("SetClipboardData failed")
	}
	return nil
}

func (platformClipboard) Watch(ctx context.Context, interval time.Duration, onChange func(string)) error {
	if interval <= 0 {
		interval = 300 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var lastSequence uintptr
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			sequence, _, _ := procGetSequence.Call()
			if sequence == 0 || sequence == lastSequence {
				continue
			}
			lastSequence = sequence
			if value, err := (platformClipboard{}).ReadText(); err == nil {
				onChange(value)
			}
		}
	}
}
