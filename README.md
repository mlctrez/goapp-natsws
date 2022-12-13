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

To run the demo application, change the working directory to `goapp-natsws/internal` and issue a `make` command to
build and run the demo application. By default, the application will run on port 8080 which can be changed in the Makefile.

The first release used nats.InProcessServer to make the connection to the websocket proxy which is still the default. 

If the environment [UseDialer](connection.go#L27) is set, a nats.CustomDialer will be used instead.  This environment
must also be present in the app.Handler environment for the client to pick it up.

If the backend urls returned from [Manager](manager.go#L14) begin with http or https, then the [Proxy](proxy.go#L69) 
will use httputil.ReverseProxy and the websocket handshake will occur in the nats codebase.

[![Go Report Card](https://goreportcard.com/badge/github.com/mlctrez/goapp-natsws)](https://goreportcard.com/report/github.com/mlctrez/goapp-natsws)

created by [tigwen](https://github.com/mlctrez/tigwen)
