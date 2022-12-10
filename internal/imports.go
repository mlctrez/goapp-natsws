package internal

// prevents removal from go.mod by keeping a reference
import (
	_ "github.com/dave/jennifer/jen"
	_ "github.com/maxence-charriere/go-app/v9/pkg/app"
	_ "github.com/nats-io/nats-server/v2/server"
	_ "github.com/nats-io/nats.go"
	_ "nhooyr.io/websocket"
)
