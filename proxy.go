package natsws

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"nhooyr.io/websocket"
	"strings"
)

var _ http.Handler = (*Proxy)(nil)

type Proxy struct {
	Context context.Context
	Manager Manager

	proxyContext context.Context
	proxyCancel  context.CancelFunc
}

func (p *Proxy) pickNatsURL() string {

	hosts := p.Manager.Backends()

	if p.Manager.Randomize() {
		rand.Shuffle(len(hosts), func(i, j int) {
			hosts[i], hosts[j] = hosts[j], hosts[i]
		})
	}

	network := "tcp4"

	for _, host := range hosts {
		if strings.HasPrefix(host, "ws://") {
			hostOnly := strings.TrimPrefix(host, "ws://")
			if conn, dialErr := net.Dial(network, hostOnly); dialErr == nil {
				_ = conn.Close()
				return host
			}
		}
		if strings.HasPrefix(host, "wss://") {
			hostOnly := strings.TrimPrefix(host, "wss://")
			if conn, dialErr := tls.Dial(network, hostOnly, p.Manager.TLSConfig()); dialErr == nil {
				_ = conn.Close()
				return host
			}
		}
	}
	return ""
}

func (p *Proxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	var natsUrl string
	if natsUrl = p.pickNatsURL(); natsUrl == "" {
		p.Manager.OnError("pickNatsURL", fmt.Errorf("none available"))
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	options := p.buildAcceptOptions(request)

	var err error
	var client *websocket.Conn
	var backend *websocket.Conn

	if p.Context == nil {
		p.Context = context.TODO()
	}

	p.proxyContext, p.proxyCancel = context.WithCancel(p.Context)

	if backend, _, err = websocket.Dial(p.proxyContext, natsUrl, nil); err != nil {
		p.Manager.OnError("Proxy websocket.Dial", err)
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	defer func() { _ = backend.Close(websocket.StatusNormalClosure, "") }()

	if client, err = websocket.Accept(writer, request, options); err != nil {
		p.Manager.OnError("Proxy websocket.Accept", err)
		// websocket.Accept takes care of writing the status code
		return
	}
	defer func() { _ = client.Close(websocket.StatusNormalClosure, "") }()

	errClient := make(chan error, 1)
	errBackend := make(chan error, 1)

	go p.copyWebSocketFrames("client->backend", client, backend, errClient, errBackend)
	go p.copyWebSocketFrames("client<-backend", backend, client, errBackend, errClient)

	var msg string
	select {
	case err = <-errClient:
		msg = "natsws.Proxy: Error copying from client to backend : %v"
	case err = <-errBackend:
		msg = "natsws.Proxy: Error copying from backend to client : %v"
	}

	switch websocket.CloseStatus(err) {
	case websocket.StatusGoingAway, websocket.StatusNormalClosure:
	default:
		if !strings.Contains(err.Error(), "failed to read frame header: EOF") {
			p.Manager.OnError(msg, err)
		}
	}

}

func (p *Proxy) copyWebSocketFrames(direction string, from, to *websocket.Conn, fromChan chan<- error, toChan chan<- error) {

	for {
		messageType, bytes, err := from.Read(p.proxyContext)
		if err != nil {
			p.Manager.OnError(direction, err)
			closeStatus := websocket.StatusNormalClosure
			closeMessage := err.Error()
			if len(closeMessage) > 123 {
				closeMessage = closeMessage[0:123]
			}
			if e, ok := err.(*websocket.CloseError); ok {
				if e.Code != websocket.StatusNoStatusRcvd {
					closeStatus = e.Code
					closeMessage = e.Reason
				}
			}
			fromChan <- err
			_ = to.Close(closeStatus, closeMessage)
			break
		}
		if p.Manager.IsDebug() {
			fmt.Printf("%s : %q\n", direction, string(bytes))
			// demo simulating a server disconnect
			if string(bytes) == "PUB demo.disconnect 0\r\n\r\n" {
				p.proxyCancel()
			}
		}
		err = to.Write(p.proxyContext, messageType, bytes)
		if err != nil {
			toChan <- err
			break
		}
	}

}

func (p *Proxy) buildAcceptOptions(request *http.Request) *websocket.AcceptOptions {
	var options *websocket.AcceptOptions
	// https://github.com/gorilla/websocket/issues/731
	// Compression in certain Safari browsers is broken, turn it off
	if strings.Contains(request.UserAgent(), "Safari") {
		options = &websocket.AcceptOptions{CompressionMode: websocket.CompressionDisabled}
	}
	return options
}
