# natsws

A [go-app](https://go-app.dev/) component and proxy to allow [nats client](https://github.com/nats-io/nats.go)
connections from a go-app application.

This was written for a very specific use case to connect to the same host:port as the go-app backend.

Don't use this if you need:

* Fail-over requirements provided by the default client.
* Support for jwt tokens or other security in the default client.

This repository contains an example go-app application at [goapp](internal/goapp) to demonstrate how to use the
component and the proxy.

See [root.go](internal/goapp/compo/root.go) for component usage,
[demo.go](internal/goapp/compo/demo/demo.go) for interacting with nats,
and [service.go](internal/goapp/service/service.go#L176)
for configuration of the proxy.

> Note: The demo app will fail if the listen address ends with 443. It will try to connect to a custom Certificate
> Authority which you won't have.

[![Go Report Card](https://goreportcard.com/badge/github.com/mlctrez/goapp-natsws)](https://goreportcard.com/report/github.com/mlctrez/goapp-natsws)

created by [tigwen](https://github.com/mlctrez/tigwen)
