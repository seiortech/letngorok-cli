package letngorokcli

// Tunnel Message type
const (
	TunnelCreated = iota
	TunnelDestroyed

	TunnelRequest
	TunnelResponse

	TunnelAuthRequest
	TunnelAuthResponse
	TunnelAuthFailure
)

// Used to communicate between the client and the server
type TunnelMessage struct {
	Type    int               `json:"type"`
	Method  string            `json:"method,omitempty"`
	ID      string            `json:"id,omitempty"`
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}
