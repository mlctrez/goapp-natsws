//go:build !wasm

package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	brotli "github.com/anargu/gin-brotli"
	"github.com/gin-gonic/gin"
	"github.com/kardianos/service"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/mlctrez/goapp-natsws"
	"github.com/mlctrez/goapp-natsws/internal/goapp"
	"github.com/mlctrez/goapp-natsws/internal/goapp/compo"
	"github.com/mlctrez/goapp-natsws/internal/gocert"
	"github.com/mlctrez/servicego"
	"github.com/nats-io/nats-server/v2/logger"
	"github.com/nats-io/nats-server/v2/server"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

func Entry() {
	compo.Routes()
	servicego.Run(&Service{})
}

var _ servicego.Service = (*Service)(nil)

var DevEnv = os.Getenv("DEV")
var IsDev = DevEnv != ""

type Service struct {
	servicego.Defaults
	serverShutdown func(ctx context.Context) error
	listenInfo     *ListenInfo
	natsServer     *server.Server
}

func (s *Service) startNats() (err error) {

	host := s.listenInfo.host
	port := s.listenInfo.portInt

	o := &server.Options{
		Host: host, Port: port + 10, NoSigs: true,
		Websocket: server.WebsocketOpts{Host: host, Port: port + 20},
	}

	if s.listenInfo.tlsConfig != nil {
		o.Websocket.TLSConfig = s.listenInfo.tlsConfig
	} else {
		o.Websocket.NoTLS = true
	}

	var svr *server.Server
	if svr, err = server.NewServer(o); err != nil {
		return
	}

	svr.SetLogger(logger.NewTestLogger("nats", false), true, false)
	go svr.Start()

	if !svr.ReadyForConnections(4 * time.Second) {
		return fmt.Errorf("nats failed to start, see log above")
	}
	svr.SetLogger(nil, false, false)

	return nil
}

type ListenInfo struct {
	listener  net.Listener
	tlsConfig *tls.Config
	host      string
	port      string
	portInt   int
}

func (i *ListenInfo) useDialer() string {
	if os.Getenv(natsws.UseDialer) == "" {
		return ""
	}
	scheme := "ws"
	if i.tlsConfig != nil {
		scheme = "wss"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, i.host, i.portInt)
}

func (i *ListenInfo) wsScheme() string {
	if i.tlsConfig != nil {
		return "wss"
	}
	return "ws"
}

func (i *ListenInfo) scheme() string {
	if i.tlsConfig != nil {
		return "https"
	}
	return "http"
}

func (i *ListenInfo) backends() string {
	if os.Getenv(natsws.UseDialer) == "" {
		return fmt.Sprintf("%s://%s:%d", i.wsScheme(), i.host, i.portInt+20)
	} else {
		return fmt.Sprintf("%s://%s:%d", i.scheme(), i.host, i.portInt+20)
	}
}

func (s *Service) Start(_ service.Service) (err error) {

	if err = s.listen(); err != nil {
		return
	}

	if err = s.startNats(); err != nil {
		_ = s.listenInfo.listener.Close()
		return
	}

	engine := buildGinEngine()
	if err = setupStaticHandlers(engine); err != nil {
		return
	}
	if err = s.setupApiEndpoints(engine); err != nil {
		return
	}
	if err = s.setupGoAppHandler(engine); err != nil {
		return
	}

	server := &http.Server{Handler: engine}
	s.serverShutdown = server.Shutdown

	go func() {
		var serveErr error
		if s.listenInfo.tlsConfig != nil {
			server.TLSConfig = s.listenInfo.tlsConfig
			serveErr = server.ServeTLS(s.listenInfo.listener, "", "")
		} else {
			serveErr = server.Serve(s.listenInfo.listener)
		}
		if serveErr != nil && serveErr != http.ErrServerClosed {
			fmt.Println("server existing on error", serveErr)
			_ = s.Log().Error(serveErr)
		}
	}()

	return nil
}

func (s *Service) Stop(_ service.Service) (err error) {

	if s.natsServer != nil {
		s.natsServer.Shutdown()
	}

	if s.serverShutdown != nil {

		stopContext, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()

		err = s.serverShutdown(stopContext)
		if errors.Is(err, context.Canceled) {
			os.Exit(-1)
		}
	}
	_ = s.Log().Info("http.Server.Shutdown success")
	return
}

func listenAddress() string {
	if address := os.Getenv("ADDRESS"); address != "" {
		return address
	}
	if port := os.Getenv("PORT"); port == "" {
		return "localhost:8080"
	} else {
		return "localhost:" + port
	}

}

func (s *Service) listen() (err error) {

	address := listenAddress()
	var listener net.Listener
	if listener, err = net.Listen("tcp4", address); err != nil {
		return
	}

	info := &ListenInfo{listener: listener}
	if info.host, info.port, err = net.SplitHostPort(address); err != nil {
		_ = listener.Close()
		return
	}

	scheme := "http"
	if os.Getenv("GOAPP_USE_TLS") != "" {
		scheme = "https"
		if info.tlsConfig, err = gocert.DevTlsConfig(info.host); err != nil {
			_ = listener.Close()
			return
		}
	}
	fmt.Printf("listening on %s://%s\n", scheme, address)

	if info.portInt, err = parseInt(info.port); err != nil {
		_ = listener.Close()
		return
	}
	s.listenInfo = info

	return nil
}

func buildGinEngine() (engine *gin.Engine) {

	if !IsDev {
		gin.SetMode(gin.ReleaseMode)
	}

	engine = gin.New()

	// required for go-app to work correctly
	engine.RedirectTrailingSlash = false

	if IsDev {
		// omit some common paths to reduce startup logging noise
		skipPaths := []string{
			"/app.css", "/app.js", "/app-worker.js", "/manifest.webmanifest", "/wasm_exec.js",
			"/web/logo-192.png", "/web/logo-512.png", "/web/app.wasm"}
		engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: skipPaths}))
	}
	engine.Use(gin.Recovery(), brotli.Brotli(brotli.DefaultCompression))

	return
}

func setupStaticHandlers(engine *gin.Engine) (err error) {

	var wasmFile fs.File
	if wasmFile, err = goapp.WebFs.Open("web/app.wasm"); err != nil {
		return
	}
	defer func() { _ = wasmFile.Close() }()

	var stat fs.FileInfo
	if stat, err = wasmFile.Stat(); err != nil {
		return
	}
	wasmSize := stat.Size()

	engine.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Wasm-Content-Length", fmt.Sprintf("%d", wasmSize))
		c.Next()
	})

	staticHandler := http.FileServer(http.FS(goapp.WebFs))
	engine.GET("/web/:path", gin.WrapH(staticHandler))

	if _, err = fs.Stat(goapp.WebFs, "web/app.css"); err == nil {
		//  use provided web/app.css instead of app.css provided by go-app
		engine.GET("/app.css", func(c *gin.Context) {
			c.Redirect(http.StatusTemporaryRedirect, "/web/app.css")
		})
	} else {
		err = nil
	}

	return
}

func (s *Service) setupApiEndpoints(engine *gin.Engine) error {

	proxy := &natsws.Proxy{
		Manager: natsws.StaticManager(os.Getenv("DEV") != "", s.listenInfo.backends()),
	}

	engine.GET("/natsws/:clientId", gin.WrapH(proxy))

	return nil
}

func (s *Service) setupGoAppHandler(engine *gin.Engine) (err error) {

	var handler *app.Handler

	// if dynamic customization of other app.Handler fields is required,
	// just build programmatically and skip the goAppHandlerFromJson() call
	if handler, err = goAppHandlerFromJson(); err != nil {
		return
	}

	handler.WasmContentLengthHeader = "Wasm-Content-Length"
	handler.Env["DEV"] = os.Getenv("DEV")
	handler.Env[natsws.UseDialer] = s.listenInfo.useDialer()

	if IsDev {
		handler.AutoUpdateInterval = time.Second * 3
		handler.Version = ""
	} else {
		handler.AutoUpdateInterval = time.Hour
		handler.Version = fmt.Sprintf("%s@%s", goapp.Version, goapp.Commit)
	}

	engine.NoRoute(gin.WrapH(handler))
	return nil
}

func goAppHandlerFromJson() (handler *app.Handler, err error) {

	var file fs.File
	if file, err = goapp.WebFs.Open("web/handler.json"); err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	handler = &app.Handler{}
	if err = json.NewDecoder(file).Decode(handler); err != nil {
		return
	}

	return
}

func parseInt(in string) (int, error) {
	i, err := strconv.ParseInt(in, 10, 16)
	if err != nil {
		return 0, err
	}
	return int(i), nil
}
