package device

import "time"

type Peer struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Address      string    `json:"address,omitempty"`
	Port         int       `json:"port,omitempty"`
	Capabilities []string  `json:"capabilities,omitempty"`
	Status       string    `json:"status"`
	PairedAt     time.Time `json:"paired_at"`
	LastSeen     time.Time `json:"last_seen"`
	Token        string    `json:"-"`
}

type pairRequest struct {
	Code         string   `json:"code"`
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Address      string   `json:"address"`
	Port         int      `json:"port"`
	Capabilities []string `json:"capabilities"`
}

type publicPeer struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Address      string    `json:"address,omitempty"`
	Port         int       `json:"port,omitempty"`
	Capabilities []string  `json:"capabilities,omitempty"`
	Status       string    `json:"status"`
	PairedAt     time.Time `json:"paired_at"`
	LastSeen     time.Time `json:"last_seen"`
}

type registryFile struct {
	Version int    `json:"version"`
	Peers   []Peer `json:"peers"`
}
