
APP_NAME=goapp

VERSION=$(shell git describe --abbrev=0 --tags 2>/dev/null || echo "v0.0.0")
COMMIT=$(shell git rev-parse --short HEAD || echo "HEAD")
MODULE=$(shell grep ^module go.mod | awk '{print $$2;}')
LD_FLAGS="-w -X $(MODULE)/goapp.Version=$(VERSION) -X $(MODULE)/goapp.Commit=$(COMMIT)"
MAIN="internal/goapp/service/main/main.go"

run: binary
	@DEV=1 ADDRESS=:9021 ./temp/$(APP_NAME)

binary: wasm
	@mkdir -p temp
	@go build -o temp/$(APP_NAME) -ldflags $(LD_FLAGS) $(MAIN)

wasm:
	@rm -f internal/goapp/web/app.wasm
	@GOARCH=wasm GOOS=js go build -o internal/goapp/web/app.wasm -ldflags $(LD_FLAGS) $(MAIN)

clean:
	@rm -rf temp
	@rm -f internal/goapp/web/app.wasm
