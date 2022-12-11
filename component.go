package natsws

import (
	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

var _ app.Mounter = (*Component)(nil)

type Component struct {
	app.Compo
	connection *Connection
}

func (n *Component) Render() app.UI {
	div := app.Div().Style("display", "none").DataSet("name", "natsws-component")
	if n.connection != nil {
		div.DataSet("client-name", n.connection.ClientName())
	}
	return div
}

func (n *Component) OnMount(ctx app.Context) {
	n.connection = &Connection{appContext: ctx}
	ctx.Async(n.connection.run)
	Observe(ctx, n.connection).OnChange(n.Update)
}
