package natsws

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/maxence-charriere/go-app/v9/pkg/errors"
	"github.com/nats-io/nats.go"
	"net"
	"net/url"
	"nhooyr.io/websocket"
	"runtime"
	"strings"
	"time"
)

const State = "natsws.Connection"
const StateClientName = State + ".clientName"
const Ping = State + ".ping"

type ChangeReason string

const Connect ChangeReason = "connect"
const Reconnect ChangeReason = "reconnect"
const Disconnect ChangeReason = "disconnect"

const UseDialer = "GOAPP_NATSWS_DIALER"

type Connection struct {
	appContext    app.Context
	clientName    string
	wsConn        *websocket.Conn
	natsConn      *nats.Conn
	subscriptions []*nats.Subscription
	changeReason  ChangeReason
}

// Observe simplifies observing the State of the Connection.
//
//		The pointer to the Connection must not be nil.
//	 See example for ways to initialize the Connection pointer.
func Observe(ctx app.Context, value *Connection) app.Observer {
	logNilConnection(value)
	observer := ctx.ObserveState(State)
	observer.Value(value)
	return observer
}

func logNilConnection(value *Connection) {
	if value == nil {
		// print a warning and show where the call came from
		_, file, line, ok := runtime.Caller(2)
		if ok {
			fileLine := fmt.Sprintf("%s:%d", file, line)
			e := errors.Newf("natsws.Observe got nil value from file:line %x", fileLine)
			app.Logf("%v", e)
		}
	}
}

func (c *Connection) setState() {
	c.appContext.SetState(State, c)
}

func (c *Connection) run() {
	c.ctx().GetState(StateClientName, &c.clientName)
	if c.clientName == "" {
		c.clientName = uuid.NewString()
		c.ctx().SetState(StateClientName, c.clientName, app.Persist)
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
			if c.natsConn.IsReconnecting() {
				break
			}
			_ = c.natsConn.Publish(Ping, []byte("ping from "+c.clientName))
		}
	}

}

func (c *Connection) unsubscribe() {
	for _, sub := range c.subscriptions {
		_ = sub.Unsubscribe()
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

func (c *Connection) Publish(subject string, message []byte) (err error) {
	var conn *nats.Conn
	if conn, err = c.Nats(); err != nil {
		return
	}

	err = conn.Publish(subject, message)
	return
}

func (c *Connection) connect() {

	var opts []nats.Option

	if app.Getenv(UseDialer) != "" {
		opts = append(opts, nats.SetCustomDialer(&customDialer{connection: c}))
	} else {
		opts = append(opts, nats.InProcessServer(c))
	}

	opts = append(opts, nats.RetryOnFailedConnect(true))
	opts = append(opts, nats.Name(c.ClientName()))

	opts = append(opts, nats.ConnectHandler(func(conn *nats.Conn) {
		c.changeReason = Connect
		c.setState()
	}))
	opts = append(opts, nats.ReconnectHandler(func(conn *nats.Conn) {
		c.changeReason = Reconnect
		c.setState()
	}))
	opts = append(opts, nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
		c.changeReason = Disconnect
		c.setState()
	}))

	// remove the wss or ws scheme from the connection url to prevent
	// websocket upgrade negotiation in the client
	natsUrl := c.wsUrl()
	natsUrl = strings.TrimPrefix(natsUrl, "ws://")
	natsUrl = strings.TrimPrefix(natsUrl, "wss://")
	c.natsConn, _ = nats.Connect(natsUrl, opts...)

	return
}

type customDialer struct {
	connection *Connection
}

func (d *customDialer) SkipTLSHandshake() bool {
	return true
}

func (d *customDialer) Dial(_, _ string) (net.Conn, error) {
	c := d.connection
	u := fmt.Sprintf("%s/natsws/%s", c.wsUrl(), c.clientName)
	conn, _, err := websocket.Dial(c.ctx(), u, nil)
	if err != nil {
		fmt.Println("customDialer websocket.Dial error", err)
		return nil, err
	}
	netConn := websocket.NetConn(c.ctx(), conn, websocket.MessageBinary)
	return netConn, nil
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
	natsWsConn := app.Getenv(UseDialer)
	if natsWsConn != "" {
		return natsWsConn
	}
	scheme := "ws" + strings.TrimPrefix(c.windowUrl().Scheme, "http")

	u := fmt.Sprintf("%s://%s/natsws/%s", scheme, c.windowUrl().Host, c.ClientName())

	return u
}

func (c *Connection) Nats() (conn *nats.Conn, err error) {
	if c.natsConn == nil || !c.natsConn.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}
	return c.natsConn, nil
}

func (c *Connection) windowUrl() *url.URL {
	return app.Window().URL()
}

func (c *Connection) ctx() app.Context {
	return c.appContext
}

func (c *Connection) ClientName() string {
	return c.clientName
}

func (c *Connection) IsConnected() bool {
	if c.natsConn == nil {
		return false
	}
	return c.natsConn.IsConnected()
}

func (c *Connection) ChangeReason() ChangeReason {
	return c.changeReason
}
