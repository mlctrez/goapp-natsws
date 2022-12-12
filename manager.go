package natsws

import (
	"crypto/tls"
	"log"
	"nhooyr.io/websocket"
	"strings"
)

type Manager interface {
	// Backends should return a list of current nats websocket endpoints.
	//   These can only be in the format wss://host:port or ws://host:port
	Backends() []string

	// TLSConfig passed to tls.Dial for testing wss:// backends.
	TLSConfig() *tls.Config

	// OnError will be called when errors occur within the proxy.
	OnError(message string, err error)

	// Randomize indicates Backends() should be shuffled before connection attempts.
	Randomize() bool

	// IsDebug will log all payloads on the server when true.
	IsDebug() bool
}

var _ Manager = (*staticManager)(nil)

func StaticManager(debug bool, backends ...string) Manager {
	return &staticManager{
		backends: backends,
		debug:    debug,
	}
}

type staticManager struct {
	backends []string
	debug    bool
}

func (s *staticManager) Backends() []string {
	return s.backends
}

func (s *staticManager) TLSConfig() *tls.Config {
	return nil
}

func (s *staticManager) OnError(message string, err error) {
	switch websocket.CloseStatus(err) {
	case websocket.StatusGoingAway, websocket.StatusNormalClosure:
	default:
		if !strings.Contains(err.Error(), "failed to read frame header: EOF") {
			log.Printf("natsws.Manager message=%s err=%v", message, err)
		}
	}
}

func (s *staticManager) Randomize() bool {
	return true
}

func (s *staticManager) IsDebug() bool {
	return s.debug
}
