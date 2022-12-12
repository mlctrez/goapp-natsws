package demo

import (
	"fmt"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	natsws "github.com/mlctrez/goapp-natsws"
	"github.com/nats-io/nats.go"
	"strings"
	"time"
)

var _ app.Mounter = (*Demo)(nil)

type Demo struct {
	app.Compo
	messages []string
	conn     natsws.Connection
}

const Subject = "testSubject"

func (d *Demo) OnMount(ctx app.Context) {
	natsws.Observe(ctx, &d.conn).OnChange(func() {
		// clients should subscribe only ChangeReason Connect to avoid creating duplicate subscriptions
		if d.conn.ChangeReason() == natsws.Connect {
			err := d.conn.Subscribe(Subject, func(msg *nats.Msg) {
				d.messages = append(d.messages, string(msg.Data))
				d.Update()
			})
			if err != nil {
				fmt.Println("subscribe error", err)
			}
		}
		_ = d.conn.Publish(Subject, []byte(time.Now().String()+" OnChange "+string(d.conn.ChangeReason())))
		d.Update()
	})
}

func (d *Demo) Render() app.UI {

	var reversed []string

	reversed = append(reversed, d.messages...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}

	return app.Div().Body(
		app.Text("changeReason "+d.conn.ChangeReason()),
		app.Br(),
		app.Button().Text("disconnect").OnClick(func(ctx app.Context, e app.Event) {
			if conn, err := d.conn.Nats(); err == nil {
				_ = conn.Publish("demo.disconnect", []byte{})
			}
		}),
		app.Br(),
		app.Button().Text("publish").OnClick(func(ctx app.Context, e app.Event) {
			err := d.conn.Publish(Subject, []byte(time.Now().String()))
			if err != nil {
				app.Log("publish error", err)
			}
		}),
		app.Br(),
		app.Textarea().Cols(80).Rows(20).Text(strings.Join(reversed, "\n")),
	)
}
