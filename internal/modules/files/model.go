package files

import "time"

const (
	shareStatusActive  = "active"
	shareStatusExpired = "expired"
	uploadStatusActive = "active"
	uploadStatusDone   = "completed"
	uploadStatusFailed = "failed"
)

// File describes a file without exposing its local source path. It is safe to
// return from both the management API and the public mobile metadata endpoint.
type File struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	MIMEType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
}

type Share struct {
	ID        string    `json:"id"`
	Token     string    `json:"token,omitempty"`
	Files     []File    `json:"files"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	URL       string    `json:"url,omitempty"`
}

// QRPayload is the stable, renderer-neutral contract for a future QR image.
// The GUI can encode URL as-is without depending on a network QR service.
type QRPayload struct {
	Version   int       `json:"version"`
	Type      string    `json:"type"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Receiver struct {
	ID        string    `json:"id"`
	Token     string    `json:"token,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	URL       string    `json:"url,omitempty"`
}

type Upload struct {
	ID             string     `json:"id"`
	ReceiverID     string     `json:"receiver_id"`
	IdempotencyKey string     `json:"idempotency_key,omitempty"`
	Name           string     `json:"name"`
	Size           int64      `json:"size"`
	SHA256         string     `json:"sha256,omitempty"`
	ReceivedBytes  int64      `json:"received_bytes"`
	Status         string     `json:"status"`
	Error          string     `json:"error,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

type ReceiveRecord struct {
	ID          string     `json:"id"`
	UploadID    string     `json:"upload_id"`
	Name        string     `json:"name"`
	Size        int64      `json:"size"`
	SHA256      string     `json:"sha256"`
	Path        string     `json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

type storedFile struct {
	File
	SourcePath string `json:"source_path"`
}

type storedShare struct {
	ID             string       `json:"id"`
	Token          string       `json:"token"`
	IdempotencyKey string       `json:"idempotency_key,omitempty"`
	Files          []storedFile `json:"files"`
	Status         string       `json:"status"`
	CreatedAt      time.Time    `json:"created_at"`
	ExpiresAt      time.Time    `json:"expires_at"`
}

type fileState struct {
	Version   int             `json:"version"`
	Shares    []storedShare   `json:"shares"`
	Receivers []Receiver      `json:"receivers"`
	Uploads   []Upload        `json:"uploads"`
	Receives  []ReceiveRecord `json:"receives"`
}
