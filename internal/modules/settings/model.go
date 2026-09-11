package settings

const version = 1

type Settings struct {
	Version             int  `json:"version"`
	AutoStart           bool `json:"auto_start"`
	MinimizeToTray      bool `json:"minimize_to_tray"`
	ExplorerContextMenu bool `json:"explorer_context_menu"`
	AutoAccept          bool `json:"auto_accept"`
	NotificationSound   bool `json:"notification_sound"`
	SendSound           bool `json:"send_sound"`
	ReceiveSound        bool `json:"receive_sound"`
}

func Defaults() Settings {
	return Settings{Version: version, MinimizeToTray: true, NotificationSound: true, SendSound: true, ReceiveSound: true}
}
