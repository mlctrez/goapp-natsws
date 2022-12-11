package demo

import (
	"fmt"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	natsws "github.com/mlctrez/goapp-natsws"
	"github.com/nats-io/nats.go"
	"time"
)

var _ app.Mounter = (*Demo)(nil)

type Demo struct {
	app.Compo
	messages   []string
	natswsConn *natsws.Connection
}

func (d *Demo) OnMount(ctx app.Context) {
	d.natswsConn = &natsws.Connection{}
	observer := natsws.Observe(ctx, d.natswsConn)
	observer.OnChange(func() {
		fmt.Println("natswsConn updated")
		err := d.natswsConn.Subscribe("currentTime", func(msg *nats.Msg) {
			d.messages = append(d.messages, string(msg.Data))
			d.Update()
		})
		if err != nil {
			fmt.Println("subscribe error", err)
		}
		d.Update()
	})
}

func (d *Demo) Render() app.UI {
	return app.Div().Body(
		app.Button().Text("publish").OnClick(func(ctx app.Context, e app.Event) {
			err := d.natswsConn.Publish("currentTime", []byte(time.Now().String()))
			if err != nil {
				app.Log("publish error", err)
			}
		}),
		app.Range(d.messages).Slice(func(i int) app.UI {
			return app.Pre().Text(d.messages[i])
		}),
	)
}
