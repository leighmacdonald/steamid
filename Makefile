GO_CMD=go
GO_BUILD=$(GO_CMD) build
GO_CLEAN=$(GO_CMD) clean
GO_TEST=$(GO_CMD) test
GO_MOD=$(GO_CMD) mod
BINARY_NAME=steamid_cli

GO_FLAGS = -ldflags "-X 'github.com/leighmacdonald/steamid/v2/steamid.BuildVersion=`git describe --abbrev=0`'"

all: test lin win mac dist

deps:
	${GO_CMD} tidy

lin:
	GOOS=linux GOARCH=amd64 $(GO_BUILD) $(GO_FLAGS) -o build/linux64/steamid main.go

win:
	GOOS=windows GOARCH=amd64 $(GO_BUILD) $(GO_FLAGS) -o build/win64/steamid.exe main.go

mac:
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) $(GO_FLAGS) -o build/macos64/steamid main.go

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
	gofmt -s -w .

check: lint_golangci lint_imports lint_cyclo lint_golint static

lint_golangci:
	@golangci-lint run --timeout 3m

lint_vet:
	@go vet -tags ci ./...

lint_imports:
	@test -z $(goimports -e -d . | tee /dev/stderr)

lint_cyclo:
	@gocyclo -over 45 .

lint_golint:
	@golint -set_exit_status $(go list -tags ci ./...)

static:
	@staticcheck -go 1.20 ./...

check_deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.3
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install golang.org/x/lint/golint@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest

dev_db:
	docker compose -f docker-compose-dev.yml up --force-recreate -V postgres

update:
	${GO_CMD} get -u ./...
