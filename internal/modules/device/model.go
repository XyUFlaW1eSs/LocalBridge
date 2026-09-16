package device

import "time"

type Peer struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Address                string    `json:"address,omitempty"`
	Port                   int       `json:"port,omitempty"`
	Capabilities           []string  `json:"capabilities,omitempty"`
	Status                 string    `json:"status"`
	Secure                 bool      `json:"secure"`
	Scheme                 string    `json:"scheme"`
	CertificateSHA256      string    `json:"certificate_sha256,omitempty"`
	PairedAt               time.Time `json:"paired_at"`
	LastSeen               time.Time `json:"last_seen"`
	Token                  string    `json:"-"`
	TokenIssuedAt          time.Time `json:"-"`
	TokenExpiresAt         time.Time `json:"-"`
	PreviousToken          string    `json:"-"`
	PreviousTokenExpiresAt time.Time `json:"-"`
}

type DiscoveredPeer struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Address           string    `json:"address"`
	Port              int       `json:"port"`
	Capabilities      []string  `json:"capabilities,omitempty"`
	Status            string    `json:"status"`
	Secure            bool      `json:"secure"`
	Scheme            string    `json:"scheme"`
	CertificateSHA256 string    `json:"certificate_sha256,omitempty"`
	LastSeen          time.Time `json:"last_seen"`
}

type pairRequest struct {
	Code              string   `json:"code"`
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Address           string   `json:"address"`
	Port              int      `json:"port"`
	Capabilities      []string `json:"capabilities"`
	Secure            bool     `json:"secure"`
	Scheme            string   `json:"scheme"`
	CertificateSHA256 string   `json:"certificate_sha256"`
}

type publicPeer struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Address           string     `json:"address,omitempty"`
	Port              int        `json:"port,omitempty"`
	Capabilities      []string   `json:"capabilities,omitempty"`
	Status            string     `json:"status"`
	Secure            bool       `json:"secure"`
	Scheme            string     `json:"scheme"`
	CertificateSHA256 string     `json:"certificate_sha256,omitempty"`
	PairedAt          time.Time  `json:"paired_at"`
	LastSeen          time.Time  `json:"last_seen"`
	TokenIssuedAt     *time.Time `json:"token_issued_at,omitempty"`
	TokenExpiresAt    *time.Time `json:"token_expires_at,omitempty"`
}

type persistedPeer struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Address                string    `json:"address,omitempty"`
	Port                   int       `json:"port,omitempty"`
	Capabilities           []string  `json:"capabilities,omitempty"`
	Status                 string    `json:"status"`
	Secure                 bool      `json:"secure"`
	Scheme                 string    `json:"scheme"`
	CertificateSHA256      string    `json:"certificate_sha256,omitempty"`
	PairedAt               time.Time `json:"paired_at"`
	LastSeen               time.Time `json:"last_seen"`
	Token                  string    `json:"token"`
	TokenIssuedAt          time.Time `json:"token_issued_at"`
	TokenExpiresAt         time.Time `json:"token_expires_at"`
	PreviousToken          string    `json:"previous_token,omitempty"`
	PreviousTokenExpiresAt time.Time `json:"previous_token_expires_at,omitempty"`
}

type registryFile struct {
	Version int             `json:"version"`
	Peers   []persistedPeer `json:"peers"`
}

type discoveryAnnouncement struct {
	Type              string   `json:"type"`
	ProtocolVersion   int      `json:"protocol_version"`
	DeviceID          string   `json:"device_id"`
	DeviceName        string   `json:"device_name"`
	APIPort           int      `json:"api_port"`
	Capabilities      []string `json:"capabilities,omitempty"`
	Nonce             string   `json:"nonce"`
	Secure            bool     `json:"secure"`
	Scheme            string   `json:"scheme"`
	CertificateSHA256 string   `json:"certificate_sha256,omitempty"`
	Fingerprint       string   `json:"fingerprint,omitempty"`
}
