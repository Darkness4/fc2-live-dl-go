GO_SRCS := $(shell find . -type f -name '*.go' -a -name '*.tpl' -a ! \( -name 'zz_generated*' -o -name '*_test.go' \))
GO_TESTS := $(shell find . -type f -name '*_test.go')
TAG_NAME = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
TAG_NAME_DEV = $(shell git describe --tags --abbrev=0 2>/dev/null)
GIT_COMMIT = $(shell git rev-parse --short=7 HEAD)
VERSION = $(or $(TAG_NAME),$(and $(TAG_NAME_DEV),$(TAG_NAME_DEV)-dev),$(GIT_COMMIT))
ifeq ($(golint),)
golint := $(shell go env GOPATH)/bin/golangci-lint
endif
bins := fc2-live-dl-go-linux-amd64 fc2-live-dl-go-linux-arm64 fc2-live-dl-go-linux-ppc64le fc2-live-dl-go-linux-s390x fc2-live-dl-go-linux-riscv64

bin/fc2-live-dl-go: $(GO_SRCS)
	CGO_ENABLED=1 go build -ldflags '-X main.version=${VERSION}' -o "$@" ./main.go

.PHONY: all
all: $(addprefix bin/,$(bins))

bin/fc2-live-dl-go-linux-amd64: $(GO_SRCS)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags '-X main.version=${VERSION}' -o "$@" ./main.go

bin/fc2-live-dl-go-linux-arm64: $(GO_SRCS)
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -ldflags '-X main.version=${VERSION}' -o "$@" ./main.go

bin/fc2-live-dl-go-linux-ppc64le: $(GO_SRCS)
	CGO_ENABLED=1 GOOS=linux GOARCH=ppc64le go build -ldflags '-X main.version=${VERSION}' -o "$@" ./main.go

bin/fc2-live-dl-go-linux-s390x: $(GO_SRCS)
	CGO_ENABLED=1 GOOS=linux GOARCH=s390x go build -ldflags '-X main.version=${VERSION}' -o "$@" ./main.go

bin/fc2-live-dl-go-linux-riscv64: $(GO_SRCS)
	CGO_ENABLED=1 GOOS=linux GOARCH=riscv64 go build -ldflags '-X main.version=${VERSION}' -o "$@" ./main.go

bin/fc2-live-dl-go-windows-amd64: $(GO_SRCS)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -ldflags '-X main.version=${VERSION}' -o "$@".exe ./main.go

bin/checksums.txt: $(addprefix bin/,$(bins))
	sha256sum -b $(addprefix bin/,$(bins)) | sed 's/bin\///' > $@

bin/checksums.md: bin/checksums.txt
	@echo "### SHA256 Checksums" > $@
	@echo >> $@
	@echo "\`\`\`" >> $@
	@cat $< >> $@
	@echo "\`\`\`" >> $@

.PHONY: build-all
build-all: $(addprefix bin/,$(bins)) bin/checksums.md

.PHONY: unit
unit:
	go test -race -covermode=atomic -tags=unit -timeout=30s ./...

.PHONY: coverage
coverage: $(GO_TESTS)
	go test -race -covermode=atomic -tags=unit -timeout=30s -coverprofile=coverage.out ./...
	go tool cover -html coverage.out -o coverage.html

.PHONY: integration
integration:
	go test -race -covermode=atomic -tags=integration -timeout=300s ./...

$(golint):
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: lint
lint: $(golint)
	$(golint) run ./...

.PHONY: clean
clean:
	rm -rf bin/
