package service

import (
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/mlctrez/goapp-natsws/internal/goapp/compo"
)

func Entry() {
	compo.Routes()
	app.RunWhenOnBrowser()
}
