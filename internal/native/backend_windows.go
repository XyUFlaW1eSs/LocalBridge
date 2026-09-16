//go:build windows

package native

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/eventbus"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/modules/settings"
	"golang.org/x/sys/windows"
)

const (
	wmDestroy         = 0x0002
	wmClose           = 0x0010
	wmCommand         = 0x0111
	wmNull            = 0x0000
	wmUser            = 0x0400
	wmTray            = wmUser + 1
	wmRButtonUp       = 0x0205
	wmLButtonDblClk   = 0x0203
	nimAdd            = 0x00000000
	nimDelete         = 0x00000002
	nimModify         = 0x00000001
	nifMessage        = 0x00000001
	nifIcon           = 0x00000002
	nifTip            = 0x00000004
	nifInfo           = 0x00000010
	mfString          = 0x00000000
	tpmLeftAlign      = 0x0000
	tpmBottomAlign    = 0x0020
	menuOpenShare     = 41001
	menuOpenReceive   = 41002
	menuOpenSettings  = 41003
	menuExit          = 41004
	idiApplication    = 32512
	mbIconInformation = 0x00000040
	niifInfo          = 0x00000001
	niifNoSound       = 0x00000010
)

type point struct{ X, Y int32 }

type wndClassEx struct {
	CbSize     uint32
	Style      uint32
	WndProc    uintptr
	CbClsExtra int32
	CbWndExtra int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSmall  uintptr
}

type message struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             uintptr
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	TimeoutOrVersion uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	Guid             [16]byte
	BalloonIcon      uintptr
}

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	shell32                 = windows.NewLazySystemDLL("shell32.dll")
	kernel32                = windows.NewLazySystemDLL("kernel32.dll")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procLoadIconW           = user32.NewProc("LoadIconW")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procMessageBeep         = user32.NewProc("MessageBeep")
	procShellNotifyIconW    = shell32.NewProc("Shell_NotifyIconW")
)

var trayWindows sync.Map

type platformBackend struct {
	cfg       Config
	logger    *slog.Logger
	registry  Registry
	mu        sync.RWMutex
	settings  settings.Settings
	hwnd      uintptr
	done      chan struct{}
	stop      chan struct{}
	className *uint16
}

func newBackend(cfg Config, logger *slog.Logger) (backend, error) {
	executable := cfg.Executable
	if executable == "" {
		var err error
		executable, err = os.Executable()
		if err != nil {
			return nil, fmt.Errorf("resolve native executable: %w", err)
		}
	}
	if executable, err := filepath.Abs(executable); err == nil {
		cfg.Executable = executable
	}
	return &platformBackend{cfg: cfg, logger: logger, registry: NewWindowsRegistry(), done: make(chan struct{}), stop: make(chan struct{})}, nil
}

func (b *platformBackend) ApplySettings(value settings.Settings) error {
	if err := SyncRegistry(b.registry, b.cfg.Executable, b.cfg.ConfigPath, value); err != nil {
		return err
	}
	b.mu.Lock()
	b.settings = value
	b.mu.Unlock()
	return nil
}

func (b *platformBackend) Start(ctx context.Context) error {
	go b.trayLoop()
	go func() {
		select {
		case <-ctx.Done():
			b.requestStop()
		case <-b.stop:
		}
	}()
	return nil
}

func (b *platformBackend) Stop(ctx context.Context) error {
	b.requestStop()
	select {
	case <-b.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *platformBackend) requestStop() {
	select {
	case <-b.stop:
	default:
		close(b.stop)
		b.mu.RLock()
		hwnd := b.hwnd
		b.mu.RUnlock()
		if hwnd != 0 {
			_, _, _ = procPostMessageW.Call(hwnd, wmClose, 0, 0)
		}
	}
}

func (b *platformBackend) Notify(event eventbus.Event, value settings.Settings) {
	if event.Type != eventbus.FileReceived && event.Type != eventbus.FileSent {
		return
	}
	b.mu.RLock()
	nidHwnd := b.hwnd
	b.mu.RUnlock()
	if nidHwnd == 0 {
		return
	}
	nid := b.notifyData(nidHwnd)
	nid.Flags = nifInfo
	nid.InfoFlags = niifInfo | niifNoSound
	title, _ := windows.UTF16FromString("LocalBridge")
	copy(nid.InfoTitle[:], title)
	messageText := "文件分享完成"
	if event.Type == eventbus.FileReceived {
		messageText = "文件接收完成"
	} else if event.Type == eventbus.FileReceiveRequested {
		messageText = "有文件等待接收确认"
	}
	message, _ := windows.UTF16FromString(messageText)
	copy(nid.Info[:], message)
	_, _, _ = procShellNotifyIconW.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
	if value.NotificationSound && ((event.Type == eventbus.FileSent && value.SendSound) || (event.Type == eventbus.FileReceived && value.ReceiveSound)) {
		_, _, _ = procMessageBeep.Call(mbIconInformation)
	}
}

func (b *platformBackend) OpenGUI(view string) error {
	b.openGUI(view)
	return nil
}

func (b *platformBackend) trayLoop() {
	defer close(b.done)
	select {
	case <-b.stop:
		return
	default:
	}
	class, err := windows.UTF16PtrFromString("LocalBridgeTrayWindow")
	if err != nil {
		b.logger.Warn("native tray unavailable", "error", err)
		return
	}
	b.className = class
	callback := windows.NewCallback(b.wndProc)
	instance, _, _ := procGetModuleHandleW.Call(0)
	wc := wndClassEx{CbSize: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: callback, Instance: instance, ClassName: class}
	if _, _, callErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); callErr != syscall.Errno(0) && callErr != windows.ERROR_CLASS_ALREADY_EXISTS {
		b.logger.Warn("native tray class registration failed", "error", callErr)
		return
	}
	hwnd, _, callErr := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, 0, instance, 0)
	if hwnd == 0 {
		b.logger.Warn("native tray window creation failed", "error", callErr)
		return
	}
	trayWindows.Store(hwnd, b)
	b.mu.Lock()
	b.hwnd = hwnd
	b.mu.Unlock()
	defer func() {
		b.removeIcon(hwnd)
		trayWindows.Delete(hwnd)
		b.mu.Lock()
		b.hwnd = 0
		b.mu.Unlock()
		_, _, _ = procDestroyWindow.Call(hwnd)
	}()
	b.addIcon(hwnd)
	select {
	case <-b.stop:
		_, _, _ = procPostMessageW.Call(hwnd, wmClose, 0, 0)
	default:
	}
	var msg message
	for {
		result, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(result) <= 0 {
			return
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (b *platformBackend) wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	switch uint32(msg) {
	case wmTray:
		switch uint32(lparam) {
		case wmRButtonUp:
			b.showMenu(hwnd)
		case wmLButtonDblClk:
			b.openGUI("shares")
		}
	case wmCommand:
		switch wparam & 0xffff {
		case menuOpenShare:
			b.openGUI("shares")
		case menuOpenReceive:
			b.openGUI("receives")
		case menuOpenSettings:
			b.openGUI("settings")
		case menuExit:
			if b.cfg.OnExit != nil {
				b.cfg.OnExit()
			}
		}
	case wmClose:
		_, _, _ = procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		_, _, _ = procPostQuitMessage.Call(0)
		return 0
	}
	return callDefWindowProc(hwnd, msg, wparam, lparam)
}

func callDefWindowProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	result, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return result
}

func (b *platformBackend) addIcon(hwnd uintptr) {
	nid := b.notifyData(hwnd)
	nid.Flags = nifMessage | nifIcon | nifTip
	_, _, _ = procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
}

func (b *platformBackend) removeIcon(hwnd uintptr) {
	nid := b.notifyData(hwnd)
	_, _, _ = procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

func (b *platformBackend) notifyData(hwnd uintptr) notifyIconData {
	nid := notifyIconData{CbSize: uint32(unsafe.Sizeof(notifyIconData{})), HWnd: hwnd, UID: 1, CallbackMessage: wmTray}
	nid.Flags = nifMessage | nifIcon | nifTip
	nid.Icon, _, _ = procLoadIconW.Call(0, idiApplication)
	tip, _ := windows.UTF16FromString("LocalBridge")
	copy(nid.Tip[:], tip)
	return nid
}

func (b *platformBackend) showMenu(hwnd uintptr) {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)
	appendMenu := func(id uintptr, text string) {
		label, _ := windows.UTF16PtrFromString(text)
		_, _, _ = procAppendMenuW.Call(menu, mfString, id, uintptr(unsafe.Pointer(label)))
	}
	appendMenu(menuOpenShare, "打开文件分享")
	appendMenu(menuOpenReceive, "打开接收记录")
	appendMenu(menuOpenSettings, "打开设置")
	appendMenu(menuExit, "退出 LocalBridge")
	var cursor point
	_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
	_, _, _ = procSetForegroundWindow.Call(hwnd)
	_, _, _ = procTrackPopupMenu.Call(menu, tpmLeftAlign|tpmBottomAlign, uintptr(cursor.X), uintptr(cursor.Y), 0, hwnd, 0)
	_, _, _ = procPostMessageW.Call(hwnd, wmNull, 0, 0)
}

func (b *platformBackend) openGUI(view string) {
	url := b.cfg.GUIURL
	if url == "" {
		url = "http://127.0.0.1:8899/app/"
	}
	if view != "" {
		url += "#" + view
	}
	if b.cfg.OnOpenGUI != nil {
		if err := b.cfg.OnOpenGUI(url); err != nil {
			b.logger.Warn("open native GUI failed", "error", err)
		}
		return
	}
	if err := OpenURL(url); err != nil {
		b.logger.Warn("open browser GUI failed", "error", err)
	}
}
