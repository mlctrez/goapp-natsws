package natsws

import (
	"crypto/tls"
	"log"
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
}

var _ Manager = (*staticManager)(nil)

func StaticManager(backends ...string) Manager {
	return &staticManager{backends: backends}
}

type staticManager struct {
	backends []string
}

func (s *staticManager) Randomize() bool {
	return true
}

func (s *staticManager) OnError(message string, err error) {
	log.Printf("natsws.Manager message=%s err=%v", message, err)
}

func (s *staticManager) Backends() []string {
	return s.backends
}

func (s *staticManager) TLSConfig() *tls.Config {
	return nil
}
