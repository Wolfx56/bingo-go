BINARY  := bingo
LDFLAGS := -s -w
GOBUILD := CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)"

.PHONY: build web run test linux clean

## build: build the front end, then the binary (front end embedded)
build: web
	$(GOBUILD) -o $(BINARY) .

## web: build the Vue front end into web/dist
web: web/node_modules
	cd web && npm run build

web/node_modules: web/package.json web/package-lock.json
	cd web && npm ci
	@touch $@

## run: build everything and start the server on http://localhost:8080
run: build
	./$(BINARY)

## test: Go tests + front-end type check
test: web/node_modules
	@mkdir -p web/dist && touch web/dist/.gitkeep
	go vet . ./internal/... ./web
	go test ./internal/...
	cd web && npm run typecheck

## linux: Linux binaries (amd64 and arm64) in build/, ready to deploy
linux: web
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o build/$(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o build/$(BINARY)-linux-arm64 .

clean:
	rm -rf $(BINARY) build web/dist
