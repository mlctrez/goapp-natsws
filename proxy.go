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
	Manager Manager
}

func (a *Proxy) pickNatsURL() string {

	hosts := a.Manager.Backends()

	if a.Manager.Randomize() {
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
		if strings.HasPrefix(host, "ws://") {
			hostOnly := strings.TrimPrefix(host, "ws://")
			if conn, dialErr := tls.Dial(network, hostOnly, a.Manager.TLSConfig()); dialErr == nil {
				_ = conn.Close()
				return host
			}
		}
	}
	return ""
}

func (a *Proxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	var natsUrl string
	if natsUrl = a.pickNatsURL(); natsUrl == "" {
		a.Manager.OnError("pickNatsURL", fmt.Errorf("none available"))
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	options := a.buildAcceptOptions(request)

	var err error
	var client *websocket.Conn
	var backend *websocket.Conn

	ctx := context.TODO()

	if backend, _, err = websocket.Dial(ctx, natsUrl, nil); err != nil {
		a.Manager.OnError("Proxy websocket.Dial", err)
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	defer func() { _ = backend.Close(websocket.StatusNormalClosure, "") }()

	if client, err = websocket.Accept(writer, request, options); err != nil {
		a.Manager.OnError("Proxy websocket.Accept", err)
		// websocket.Accept takes care of writing the status code
		return
	}
	defer func() { _ = client.Close(websocket.StatusNormalClosure, "") }()

	errClient := make(chan error, 1)
	errBackend := make(chan error, 1)

	go copyWebSocketFrames(ctx, client, backend, errClient, errBackend)
	go copyWebSocketFrames(ctx, backend, client, errBackend, errClient)

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
			a.Manager.OnError(msg, err)
		}
	}

}

func copyWebSocketFrames(ctx context.Context, from, to *websocket.Conn, fromChan chan<- error, toChan chan<- error) {

	for {
		messageType, bytes, err := from.Read(ctx)
		if err != nil {
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
		err = to.Write(ctx, messageType, bytes)
		if err != nil {
			toChan <- err
			break
		}
	}

}

func (a *Proxy) buildAcceptOptions(request *http.Request) *websocket.AcceptOptions {
	var options *websocket.AcceptOptions
	// https://github.com/gorilla/websocket/issues/731
	// Compression in certain Safari browsers is broken, turn it off
	if strings.Contains(request.UserAgent(), "Safari") {
		options = &websocket.AcceptOptions{CompressionMode: websocket.CompressionDisabled}
	}
	return options
}
