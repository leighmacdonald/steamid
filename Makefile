GO_CMD=go
GO_BUILD=$(GO_CMD) build
GO_CLEAN=$(GO_CMD) clean
GO_TEST=$(GO_CMD) test
GO_MOD=$(GO_CMD) mod


all: test lin win mac dist

deps:
	${GO_CMD} tidy

lin:
	GOOS=linux GOARCH=amd64 $(GO_BUILD) -o build/linux64/steamid main.go

win:
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o build/win64/steamid.exe main.go

mac:
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o build/macos64/steamid main.go

dist:
	zip -j steamid-`git describe --abbrev=0`-win64.zip build/win64/steamid.exe LICENSE.md
	zip -j steamid-`git describe --abbrev=0`-linux64.zip build/linux64/steamid LICENSE.md
	zip -j steamid-`git describe --abbrev=0`-macos64.zip build/macos64/steamid LICENSE.md

run:
	$(GO_BUILD) -o $(BINARY_NAME) -v .
	./$(BINARY_NAME)

test:
	$(GO_TEST) -v ./...

fmt:
	#gci write . --skip-generated -s standard -s default
	gofumpt -l -w .

check: fmt lint_golangci static

lint_golangci:
	@golangci-lint run --timeout 3m

static:
	@staticcheck -go 1.20 ./...

check_deps:
	go install github.com/daixiang0/gci@v0.13.0
	go install mvdan.cc/gofumpt@v0.6.0
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.2
	go install honnef.co/go/tools/cmd/staticcheck@v0.4.7
	go install github.com/goreleaser/goreleaser@v1.24.0

dev_db:
	docker compose -f docker-compose-dev.yml up --force-recreate -V postgres

update:
	${GO_CMD} get -u ./...
