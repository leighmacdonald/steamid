GO_CMD=go
GO_BUILD=$(GO_CMD) build
GO_CLEAN=$(GO_CMD) clean
GO_TEST=$(GO_CMD) test
GO_MOD=$(GO_CMD) mod

test:
	$(GO_TEST) -v ./...

fmt:
	go tool gci write . --skip-generated -s standard -s default
	go tool gofumpt -l -w .

check: fmt lint_golangci static

lint_golangci:
	go tool golangci-lint run --timeout 3m

static:
	go tool staticcheck -go 1.24 ./...

update:
	${GO_CMD} get -u ./...
