# goapp-natsws

[go-app](https://go-app.dev/) component and proxy to allow [nats client](https://github.com/nats-io/nats.go)
connections from go-app applications to interact with [nats-server](https://github.com/nats-io/nats-server) via
websockets.

This repository contains an example go-app application at [goapp](internal/goapp) to demonstrate how to use the
component and the proxy.

See [demo.go](internal/goapp/compo/demo/demo.go) for component usage and [service.go](internal/goapp/service/service.go)
for configuration of the proxy.

The demo application uses a custom certificate authority if the listen address ends with 443.  

[![Go Report Card](https://goreportcard.com/badge/github.com/mlctrez/goapp-natsws)](https://goreportcard.com/report/github.com/mlctrez/goapp-natsws)

created by [tigwen](https://github.com/mlctrez/tigwen)
