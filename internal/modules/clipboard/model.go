package clipboard

import "time"

type Item struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	MimeType  string    `json:"mime_type"`
	Content   string    `json:"content"`
	Hash      string    `json:"hash"`
	DeviceID  string    `json:"device_id"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

type pushRequest struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	MimeType string `json:"mime_type"`
	Content  string `json:"content"`
	Text     string `json:"text"`
	Hash     string `json:"hash"`
	DeviceID string `json:"device_id"`
}

type pushResponse struct {
	Accepted bool  `json:"accepted"`
	Item     *Item `json:"item,omitempty"`
}
