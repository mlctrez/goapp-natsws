package natsws

import "github.com/maxence-charriere/go-app/v9/pkg/app"

type Demo struct {
	app.Compo
	conn Connection
}

// OnMount demonstrating non pointer Connection.
func (d *Demo) OnMount(ctx app.Context) {
	Observe(ctx, &d.conn).OnChange(func() {
		if d.conn.ChangeReason() == Connect {
			// first time setup
		} else {
			// reconnect setup
		}
	})
}

type DemoTwo struct {
	app.Compo
	conn *Connection
}

// OnMount demonstrating pointer Connection.
func (d *DemoTwo) OnMount(ctx app.Context) {
	d.conn = &Connection{}
	Observe(ctx, d.conn)
}

func ExampleObserve() {
	// the two components above show two different methods
	// for calling Observe with a non nil pointer Connection
}
