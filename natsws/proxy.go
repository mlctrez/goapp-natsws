package natsws

import (
	"context"
	"fmt"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/nats-io/nats.go"
	"net"
	"net/url"
	"nhooyr.io/websocket"
	"strings"
)

var _ nats.InProcessConnProvider = &Proxy{}

type Proxy struct {
	Context  context.Context
	Backend  *url.URL
	ClientId string
	Conn     *websocket.Conn
}

func (p *Proxy) InProcessConn() (netConn net.Conn, err error) {
	var u string
	if u, err = p.BackendURl(); err != nil {
		return
	}

	opts := &websocket.DialOptions{}
	app.Logf("websocket.Dial %q", u)
	if p.Conn, _, err = websocket.Dial(p.Context, u, opts); err != nil {
		return
	}

	netConn = websocket.NetConn(p.Context, p.Conn, websocket.MessageBinary)
	return
}

func (p *Proxy) BackendURl() (u string, err error) {
	var urlCopy *url.URL
	if urlCopy, err = url.Parse(p.Backend.String()); err != nil {
		return
	}

	urlCopy.Path = fmt.Sprintf("/natsws/%s", p.ClientId)
	urlCopy.Fragment = ""
	urlCopy.RawQuery = ""
	if !strings.Contains(urlCopy.Host, ":") {
		switch urlCopy.Scheme {
		case "https":
			urlCopy.Host += ":443"
		default:
			urlCopy.Host += ":80"
		}
	}
	if urlCopy.Scheme == "https" {
		urlCopy.Scheme = "wss"
	} else {
		urlCopy.Scheme = "ws"
	}

	return urlCopy.String(), nil
}
