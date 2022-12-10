package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/nats-io/nats.go"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"nhooyr.io/websocket"
	"strings"
)

var _ http.Handler = (*NatsProxy)(nil)

type NatsProxy struct {
	backends []string
	monitor  *nats.Conn
}

func New(backends ...string) *NatsProxy {

	a := &NatsProxy{backends: backends}

	var monitorOpt []nats.Option
	monitorOpt = append(monitorOpt, nats.RetryOnFailedConnect(true))
	monitorOpt = append(monitorOpt, nats.ConnectHandler(a.updateServers))
	monitorOpt = append(monitorOpt, nats.DiscoveredServersHandler(a.updateServers))

	monitor, err := nats.Connect(strings.Join(a.backends, ","), monitorOpt...)
	if err != nil {
		log.Println(err)
	}
	a.monitor = monitor

	return a
}

func (a *NatsProxy) updateServers(conn *nats.Conn) {
	var newBackends []string
	connectedUrl := conn.ConnectedUrl()
	if connectedUrl != "" {
		newBackends = append(newBackends, connectedUrl)
	}
	a.backends = append(newBackends, conn.DiscoveredServers()...)
	//fmt.Println("backends are now", a.backends)
}

func (a *NatsProxy) pickNatsURL() *url.URL {

	hosts := append([]string{}, a.backends...)

	rand.Shuffle(len(hosts), func(i, j int) {
		hosts[i], hosts[j] = hosts[j], hosts[i]
	})

	for _, host := range hosts {
		if parse, err := url.Parse(host); err != nil {
			continue
		} else {
			if conn, dialErr := tls.Dial("tcp4", parse.Host, nil); dialErr == nil {
				_ = conn.Close()
				return parse
			}
		}
	}
	return nil
}

func (a *NatsProxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	var natsUrl *url.URL
	if natsUrl = a.pickNatsURL(); natsUrl == nil {
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	options := a.buildAcceptOptions(request)

	var err error
	var client *websocket.Conn
	var backend *websocket.Conn

	ctx := context.TODO()

	if backend, _, err = websocket.Dial(ctx, natsUrl.String(), nil); err != nil {
		fmt.Println("NatsProxy websocket.Dial error :", err)
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	defer func() { _ = backend.Close(websocket.StatusNormalClosure, "") }()

	if client, err = websocket.Accept(writer, request, options); err != nil {
		// websocket.Accept takes care of writing the status code
		return
	}
	defer func() { _ = client.Close(websocket.StatusNormalClosure, "") }()

	errClient := make(chan error, 1)
	errBackend := make(chan error, 1)

	//fmt.Println("connecting client", request.RemoteAddr, "to", natsUrl.String())
	//fmt.Println("client headers", request.Header)

	go copyWebSocketFrames(ctx, client, backend, errClient, errBackend)
	go copyWebSocketFrames(ctx, backend, client, errBackend, errClient)

	var msg string
	select {
	case err = <-errClient:
		msg = "NatsProxy: Error copying from client to backend : %v"
	case err = <-errBackend:
		msg = "NatsProxy: Error copying from backend to client : %v"
	}

	switch websocket.CloseStatus(err) {
	case websocket.StatusGoingAway, websocket.StatusNormalClosure:
	default:
		if !strings.Contains(err.Error(), "failed to read frame header: EOF") {
			log.Printf(msg, err)
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

func (a *NatsProxy) buildAcceptOptions(request *http.Request) *websocket.AcceptOptions {
	var options *websocket.AcceptOptions
	// https://github.com/gorilla/websocket/issues/731
	// Compression in certain Safari browsers is broken, turn it off
	if strings.Contains(request.UserAgent(), "Safari") {
		options = &websocket.AcceptOptions{CompressionMode: websocket.CompressionDisabled}
	}
	return options
}
