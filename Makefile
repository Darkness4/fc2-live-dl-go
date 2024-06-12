GO_SRCS := $(shell find . -type f -name '*.go' -a -name '*.tpl' -a ! \( -name 'zz_generated*' -o -name '*_test.go' \))
GO_TESTS := $(shell find . -type f -name '*_test.go')
TAG_NAME = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
TAG_NAME_DEV = $(shell git describe --tags --abbrev=0 2>/dev/null)
VERSION_CORE = $(shell echo $(TAG_NAME) | sed 's/^\(v[0-9]\+\.[0-9]\+\.[0-9]\+\)\(+\([0-9]\+\)\)\?$$/\1/')
VERSION_CORE_DEV = $(shell echo $(TAG_NAME_DEV) | sed 's/^\(v[0-9]\+\.[0-9]\+\.[0-9]\+\)\(+\([0-9]\+\)\)\?$$/\1/')
GIT_COMMIT = $(shell git rev-parse --short=7 HEAD)
VERSION = $(or $(and $(TAG_NAME),$(VERSION_CORE)),$(and $(TAG_NAME_DEV),$(VERSION_CORE_DEV)-dev),$(GIT_COMMIT))
VERSION_NO_V = $(shell echo $(VERSION) | sed 's/^v\(.*\)$$/\1/')
golint := $(shell which golangci-lint)
ifeq ($(golint),)
golint := $(shell go env GOPATH)/bin/golangci-lint
endif

.PHONY: bin/fc2-live-dl-go
bin/fc2-live-dl-go: $(GO_SRCS)
	CGO_ENABLED=1 go build -trimpath -ldflags '-X main.version=${VERSION} -s -w' -o "$@" ./main.go

.PHONY: bin/fc2-live-dl-go-static
bin/fc2-live-dl-go-static: $(GO_SRCS)
	CGO_ENABLED=1 go build -trimpath -ldflags '-X main.version=${VERSION} -s -w -extldflags "-lswresample -static"' -o "$@" ./main.go

.PHONY: bin/fc2-live-dl-go-static.exe
bin/fc2-live-dl-go-static.exe: $(GO_SRCS)
	CGO_ENABLED=1 \
	GOOS=windows \
	GOARCH=amd64 \
	go build -trimpath -ldflags '-X main.version=${VERSION} -linkmode external -s -w -extldflags "-static"' -o "$@" ./main.go

.PHONY: bin/fc2-live-dl-go-darwin
bin/fc2-live-dl-go-darwin: $(GO_SRCS)
	CGO_ENABLED=1 \
	GOOS=darwin \
	go build -trimpath -ldflags '-X main.version=${VERSION} -linkmode external -s -w' -o "$@" ./main.go

.PHONY: all
all: $(addprefix bin/,$(bins))

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
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: lint
lint: $(golint)
	$(golint) run ./...

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: package
package: target/darwin \
	target/static \
	target/static-windows \
	target/checksums.txt \
	target/checksums.md

target/checksums.txt: target/darwin \
	target/static \
	target/static-windows
	sha256sum -b $(addsuffix /*,$^) | sed 's|target/.*/||' > $@

target/checksums.md: target/checksums.txt
	@echo "### SHA256 Checksums" > $@
	@echo >> $@
	@echo "\`\`\`" >> $@
	@cat $< >> $@
	@echo "\`\`\`" >> $@

target/static:
	podman manifest rm localhost/builder:static || true
	podman build \
		--manifest localhost/builder:static \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8,linux/riscv64 \
		--target busybox \
		-f Dockerfile.static .
	mkdir -p ./target/static
	podman run --rm \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		--entrypoint sh \
		localhost/builder:static -c "mv /fc2-live-dl-go /target/static/fc2-live-dl-go-linux-amd64"
	podman run --rm \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		--entrypoint sh \
		localhost/builder:static -c "mv /fc2-live-dl-go /target/static/fc2-live-dl-go-linux-arm64"
	podman run --rm \
		-v $(shell pwd)/target/:/target/ \
		--arch riscv64 \
		--entrypoint sh \
		localhost/builder:static -c "mv /fc2-live-dl-go /target/static/fc2-live-dl-go-linux-riscv64"
	./assert-arch.sh

target/static-windows:
	podman build \
		-t localhost/builder:static-windows \
		-f Dockerfile.static-windows .
	mkdir -p ./target/static-windows
	podman run --rm \
		-v $(shell pwd)/target/:/target/ \
		localhost/builder:static-windows mv /work/bin/fc2-live-dl-go-static.exe /target/static-windows/fc2-live-dl-go-windows-amd64.exe

target/darwin:
	podman manifest rm localhost/builder:darwin || true
	podman build \
		--manifest localhost/builder:darwin \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8 \
		--target busybox \
		-f Dockerfile.darwin .
	mkdir -p ./target/darwin
	podman run --rm \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		--entrypoint sh \
		localhost/builder:darwin -c "mv /fc2-live-dl-go /target/darwin/fc2-live-dl-go-darwin-amd64"
	podman run --rm \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		--entrypoint sh \
		localhost/builder:darwin -c "mv /fc2-live-dl-go /target/darwin/fc2-live-dl-go-darwin-arm64"

.PHONY: docker-static
docker-static:
	podman manifest rm ghcr.io/darkness4/fc2-live-dl-go:latest || true
	podman build \
		--manifest ghcr.io/darkness4/fc2-live-dl-go:latest \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8,linux/riscv64 \
		-f Dockerfile.static .
	podman manifest push --all ghcr.io/darkness4/fc2-live-dl-go:latest "docker://ghcr.io/darkness4/fc2-live-dl-go:latest"
	podman manifest push --all ghcr.io/darkness4/fc2-live-dl-go:latest "docker://ghcr.io/darkness4/fc2-live-dl-go:${VERSION_NO_V}"
	podman manifest push --all ghcr.io/darkness4/fc2-live-dl-go:latest "docker://ghcr.io/darkness4/fc2-live-dl-go:dev"

.PHONY: docker-static-base
docker-static-base:
	podman build \
		-t ghcr.io/darkness4/fc2-live-dl-go:latest-static-base \
		--platform=linux/amd64 \
		-f Dockerfile.static-base .
	podman push ghcr.io/darkness4/fc2-live-dl-go:latest-static-base

.PHONY: docker-static-windows-base
docker-static-windows-base:
	podman build \
		-t ghcr.io/darkness4/fc2-live-dl-go:latest-static-windows-base \
		-f Dockerfile.static-windows-base .
	podman push ghcr.io/darkness4/fc2-live-dl-go:latest-static-windows-base

.PHONY: docker-darwin-base
docker-darwin-base:
	podman build \
		-t ghcr.io/darkness4/fc2-live-dl-go:latest-darwin-base-amd64 \
		--build-arg TARGET_ARCH=x86_64 \
		-f Dockerfile.darwin-base .
	podman push ghcr.io/darkness4/fc2-live-dl-go:latest-darwin-base-amd64
	podman build \
		-t ghcr.io/darkness4/fc2-live-dl-go:latest-darwin-base-arm64 \
		--build-arg TARGET_ARCH=aarch64 \
		--build-arg OSX_VERSION_MIN=11.0 \
		-f Dockerfile.darwin-base .
	podman push ghcr.io/darkness4/fc2-live-dl-go:latest-darwin-base-arm64

.PHONY: version
version:
	echo version=$(VERSION)

.PHONY: memleaks
memleaks:
	cd video/probe && make valgrind
	cd video/concat && make valgrind
