BINARY  := bingo
LDFLAGS := -s -w
GOBUILD := CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)"

.PHONY: build web run test linux clean

## build : compile le front puis le binaire (front embarqué)
build: web
	$(GOBUILD) -o $(BINARY) .

## web : compile le front Vue dans web/dist
web: web/node_modules
	cd web && npm run build

web/node_modules: web/package.json web/package-lock.json
	cd web && npm ci
	@touch $@

## run : compile tout et lance le serveur sur http://localhost:8080
run: build
	./$(BINARY)

## test : tests Go + vérification des types du front
test: web/node_modules
	@mkdir -p web/dist && touch web/dist/.gitkeep
	go vet . ./internal/... ./web
	go test ./internal/...
	cd web && npm run typecheck

## linux : binaires Linux (amd64 et arm64) dans build/, prêts à déployer
linux: web
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o build/$(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o build/$(BINARY)-linux-arm64 .

clean:
	rm -rf $(BINARY) build web/dist
