package syncstore

import (
	"encoding/json"
	"time"
)

type State string

const (
	StatePending    State = "pending"
	StateDelivering State = "delivering"
	StateDelivered  State = "delivered"
	StateFailed     State = "failed"
	StateCanceled   State = "canceled"
	StateExpired    State = "expired"
)

type Envelope struct {
	ID             string          `json:"id"`
	Kind           string          `json:"kind"`
	Type           string          `json:"type"`
	MIMEType       string          `json:"mime_type"`
	Hash           string          `json:"hash"`
	SourceDeviceID string          `json:"source_device_id"`
	TargetDeviceID string          `json:"target_device_id,omitempty"`
	CorrelationID  string          `json:"correlation_id,omitempty"`
	Size           int             `json:"size"`
	Payload        json.RawMessage `json:"payload"`
	CreatedAt      time.Time       `json:"created_at"`
	ExpiresAt      *time.Time      `json:"expires_at,omitempty"`
}

type Job struct {
	ID        string    `json:"id"`
	Envelope  Envelope  `json:"envelope"`
	State     State     `json:"state"`
	Attempts  int       `json:"attempts"`
	LastError string    `json:"last_error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type fileState struct {
	Version int   `json:"version"`
	Jobs    []Job `json:"jobs"`
}
