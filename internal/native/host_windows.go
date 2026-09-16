//go:build windows

package native

import (
	"context"
	"errors"
	"log/slog"
	"runtime"
	"sync"
	"syscall"

	webview2 "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

const (
	gwlpWndProc = ^uintptr(3) // -4 represented as uintptr
	swHide      = 0
	swRestore   = 9
)

var (
	procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW")
	procCallWindowProcW   = user32.NewProc("CallWindowProcW")
	procShowWindow        = user32.NewProc("ShowWindow")
)

// windowsHost owns the WebView2 window and keeps every WebView call on its
// dedicated locked OS thread. The service and tray remain independent of the
// window lifecycle.
type windowsHost struct {
	logger      *slog.Logger
	settings    func() bool
	onClosed    func()
	mu          sync.Mutex
	view        webview2.WebView
	hwnd        uintptr
	originalWnd uintptr
	callback    uintptr
	url         string
	starting    bool
	stopping    bool
	done        chan struct{}
}

func newWindowsHost(logger *slog.Logger, minimizeToTray func() bool, onClosed func()) *windowsHost {
	return &windowsHost{logger: logger, settings: minimizeToTray, onClosed: onClosed}
}

func (h *windowsHost) Open(url string) error {
	h.mu.Lock()
	h.url = url
	if h.view != nil {
		view := h.view
		hwnd := h.hwnd
		h.mu.Unlock()
		view.Dispatch(func() {
			view.Navigate(url)
			_, _, _ = procShowWindow.Call(hwnd, swRestore)
			_, _, _ = procSetForegroundWindow.Call(hwnd)
		})
		return nil
	}
	if h.starting {
		h.mu.Unlock()
		return nil
	}
	h.starting = true
	h.stopping = false
	h.done = make(chan struct{})
	ready := make(chan error, 1)
	h.mu.Unlock()
	go h.run(ready)
	return <-ready
}

func (h *windowsHost) RequestStop() {
	h.mu.Lock()
	h.stopping = true
	view := h.view
	h.mu.Unlock()
	if view != nil {
		view.Dispatch(func() { view.Destroy() })
	}
}

func (h *windowsHost) Stop(ctx context.Context) error {
	h.RequestStop()
	h.mu.Lock()
	done := h.done
	starting := h.starting
	h.mu.Unlock()
	if !starting || done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *windowsHost) run(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	view := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "LocalBridge",
			Width:  1180,
			Height: 760,
			Center: true,
		},
	})
	if view == nil {
		h.finishStart(errors.New("WebView2 runtime is unavailable"), ready)
		return
	}
	hwnd := uintptr(view.Window())
	callback := windows.NewCallback(h.windowProc)
	original, _, callErr := procSetWindowLongPtrW.Call(hwnd, gwlpWndProc, callback)
	if original == 0 {
		view.Destroy()
		view.Run()
		err := errors.New("subclass WebView2 window failed")
		if callErr != syscall.Errno(0) {
			err = callErr
		}
		h.finishStart(err, ready)
		return
	}

	h.mu.Lock()
	h.view = view
	h.hwnd = hwnd
	h.originalWnd = original
	h.callback = callback
	url := h.url
	stopping := h.stopping
	h.mu.Unlock()
	if stopping {
		view.Destroy()
	} else {
		view.Navigate(url)
	}
	ready <- nil
	view.Run()

	h.mu.Lock()
	wasStopping := h.stopping
	h.view = nil
	h.hwnd = 0
	h.originalWnd = 0
	h.callback = 0
	h.starting = false
	done := h.done
	h.mu.Unlock()
	if done != nil {
		close(done)
	}
	if !wasStopping && h.onClosed != nil {
		h.onClosed()
	}
}

func (h *windowsHost) finishStart(err error, ready chan<- error) {
	h.mu.Lock()
	h.starting = false
	done := h.done
	h.mu.Unlock()
	if done != nil {
		close(done)
	}
	ready <- err
}

func (h *windowsHost) windowProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	h.mu.Lock()
	original := h.originalWnd
	stopping := h.stopping
	h.mu.Unlock()
	if uint32(msg) == wmClose && !stopping && h.settings != nil && h.settings() {
		_, _, _ = procShowWindow.Call(hwnd, swHide)
		return 0
	}
	if original == 0 {
		return callDefWindowProc(hwnd, msg, wparam, lparam)
	}
	result, _, _ := procCallWindowProcW.Call(original, hwnd, msg, wparam, lparam)
	return result
}
