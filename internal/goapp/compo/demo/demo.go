package demo

import (
	"fmt"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/mlctrez/goapp-natsws/natsws"
	"github.com/nats-io/nats.go"
	"time"
)

var _ app.Mounter = (*Demo)(nil)
var _ app.Dismounter = (*Demo)(nil)

type Demo struct {
	app.Compo
	conn     *nats.Conn
	messages []string
	cts      *nats.Subscription
}

func (d *Demo) OnDismount() {
	if d.cts != nil {
		if err := d.cts.Unsubscribe(); err != nil {
			app.Log("d.cts.Unsubscribe()", err)
		}
	}
}

func (d *Demo) WithConn(_ app.Context, conn *nats.Conn) {
	d.conn = conn
}

func (d *Demo) OnMount(ctx app.Context) {

	go func() {
		start := time.Now()
		for d.conn == nil {
			if time.Since(start) > 3*time.Second {
				app.Logf("get nats connection aborting after %s", time.Since(start))
				return
			}
			natsws.Action(ctx).WithConn(d)
			time.Sleep(100 * time.Millisecond)
		}
		app.Logf("got nats connection %s after %s", d.conn.DiscoveredServers(), time.Since(start))

		var err error
		d.cts, err = d.conn.Subscribe("currentTime", func(msg *nats.Msg) {
			d.messages = append(d.messages, string(msg.Data))
			d.Update()
		})
		if err != nil {
			app.Log("error on subscribe", err)
		}

	}()

}

func (d *Demo) Render() app.UI {
	return app.Div().Body(
		app.Button().Text("publish").OnClick(func(ctx app.Context, e app.Event) {
			if d.conn == nil {
				fmt.Println("can't publish, no nats connection")
				return
			}
			err := d.conn.Publish("currentTime", []byte(time.Now().String()))
			if err != nil {
				app.Log("publish error", err)
			}
		}),
		app.Range(d.messages).Slice(func(i int) app.UI {
			return app.Pre().Text(d.messages[i])
		}),
	)
}
