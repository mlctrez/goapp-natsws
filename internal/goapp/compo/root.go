package compo

import (
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/mlctrez/goapp-natsws/internal/goapp/compo/demo"
	"github.com/mlctrez/goapp-natsws/natsws"
)

var _ app.AppUpdater = (*Root)(nil)

type Root struct {
	app.Compo
}

func (r *Root) Render() app.UI {
	return app.Div().Body(
		&natsws.Nats{},
		&demo.Demo{},
	)
}

func (r *Root) OnAppUpdate(ctx app.Context) {
	if app.Getenv("DEV") != "" && ctx.AppUpdateAvailable() {
		ctx.Reload()
	}
}
