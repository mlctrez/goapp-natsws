package natsws

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/nats-io/nats.go"
	"net"
	"net/url"
	"nhooyr.io/websocket"
	"strings"
	"time"
)

const State = "natsws.Connection"
const StateClientName = State + ".clientName"
const Ping = State + ".ping"

type Connection struct {
	appContext    app.Context
	clientName    string
	connected     bool
	wsConn        *websocket.Conn
	natsConn      *nats.Conn
	subscriptions []*nats.Subscription
}

func Observe(ctx app.Context, value *Connection) app.Observer {
	observer := ctx.ObserveState(State)
	observer.Value(value)
	return observer
}

func (c *Connection) setState() {
	c.appContext.SetState(State, c)
}

func (c *Connection) run() {
	c.ctx().GetState(StateClientName, &c.clientName)
	if c.clientName == "" {
		c.clientName = uuid.NewString()
		c.ctx().SetState(StateClientName, c.clientName)
	}

	defer c.unsubscribe()

	keepAlive := time.NewTicker(time.Second * 5)
	defer keepAlive.Stop()

	initialConnect := make(chan bool, 1)
	initialConnect <- true
	defer close(initialConnect)

	for {
		select {
		case <-c.ctx().Done():
			return
		case <-initialConnect:
			c.connect()
		case <-keepAlive.C:
			if err := c.natsConn.Publish(Ping, []byte("ping")); err != nil {
				c.connect()
			}
		}
	}

}
func (c *Connection) unsubscribe() {
	fmt.Println("Connection: unsubscribe")
	for _, sub := range c.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			fmt.Printf("sub.Unsubscribe() error : %#v", err)
		}
	}
}
func (c *Connection) Subscribe(subject string, cb nats.MsgHandler) (err error) {

	var conn *nats.Conn
	if conn, err = c.Nats(); err != nil {
		return
	}
	var sub *nats.Subscription
	if sub, err = conn.Subscribe(subject, cb); err != nil {
		return
	}
	c.subscriptions = append(c.subscriptions, sub)
	return
}

func (c *Connection) connect() {

	opts := []nats.Option{nats.Name(c.ClientName()), nats.InProcessServer(c)}

	var err error
	if c.natsConn, err = nats.Connect(c.wsUrl(), opts...); err != nil {
		c.natsConn = nil
		c.connected = false
		return
	} else {
		c.connected = true
	}

	c.setState()
	return
}

func (c *Connection) InProcessConn() (netConn net.Conn, err error) {
	opts := &websocket.DialOptions{}

	if c.wsConn, _, err = websocket.Dial(c.ctx(), c.wsUrl(), opts); err != nil {
		return
	}

	netConn = websocket.NetConn(c.ctx(), c.wsConn, websocket.MessageBinary)
	return
}

func (c *Connection) wsUrl() string {
	scheme := "ws" + strings.TrimPrefix(c.windowUrl().Scheme, "http")

	u := fmt.Sprintf("%s://%s/natsws/%s", scheme, c.windowUrl().Host, c.ClientName())

	return u
}

func (c *Connection) Publish(subject string, message []byte) (err error) {
	var conn *nats.Conn
	if conn, err = c.Nats(); err != nil {
		return err
	}
	return conn.Publish(subject, message)
}

func (c *Connection) Nats() (conn *nats.Conn, err error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}
	return c.natsConn, nil
}

func (c *Connection) windowUrl() *url.URL { return app.Window().URL() }
func (c *Connection) ClientName() string  { return c.clientName }
func (c *Connection) IsConnected() bool   { return c.connected }
func (c *Connection) ctx() app.Context    { return c.appContext }
