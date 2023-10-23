GO_SRCS := $(shell find . -type f -name '*.go' -a -name '*.tpl' -a ! \( -name 'zz_generated*' -o -name '*_test.go' \))
GO_TESTS := $(shell find . -type f -name '*_test.go')
TAG_NAME = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
TAG_NAME_DEV = $(shell git describe --tags --abbrev=0 2>/dev/null)
BUILD = $(shell echo $(TAG_NAME) | sed 's/^v[0-9]\+\.[0-9]\+\.[0-9]\+\(+\([0-9]\+\)\)\?$$/\2/')
BUILD_DEV = $(shell echo $(TAG_NAME_DEV) | sed 's/^v[0-9]\+\.[0-9]\+\.[0-9]\+\(+\([0-9]\+\)\)\?$$/\2/')
VERSION_CORE = $(shell echo $(TAG_NAME) | sed 's/^\(v[0-9]\+\.[0-9]\+\.[0-9]\+\)\(+\([0-9]\+\)\)\?$$/\1/')
VERSION_CORE_DEV = $(shell echo $(TAG_NAME_DEV) | sed 's/^\(v[0-9]\+\.[0-9]\+\.[0-9]\+\)\(+\([0-9]\+\)\)\?$$/\1/')
GIT_COMMIT = $(shell git rev-parse --short=7 HEAD)
VERSION = $(or $(and $(TAG_NAME),$(VERSION_CORE)),$(and $(TAG_NAME_DEV),$(VERSION_CORE_DEV)-dev),$(GIT_COMMIT))
VERSION_NO_V = $(shell echo $(VERSION) | sed 's/^v\(.*\)$$/\1/')
RELEASE = $(or $(and $(TAG_NAME),$(BUILD)),$(and $(TAG_NAME_DEV),$(BUILD_DEV)),0)
ifeq ($(golint),)
golint := $(shell go env GOPATH)/bin/golangci-lint
endif

.PHONY: bin/fc2-live-dl-go
bin/fc2-live-dl-go: $(GO_SRCS)
	CGO_ENABLED=1 go build -trimpath -ldflags '-X main.version=${VERSION}+${RELEASE} -s -w' -o "$@" ./main.go

.PHONY: bin/fc2-live-dl-go-static
bin/fc2-live-dl-go-static: $(GO_SRCS)
	CGO_ENABLED=1 go build -trimpath -ldflags '-X main.version=${VERSION}+${RELEASE} -s -w -extldflags "-lswresample -static"' -o "$@" ./main.go

.PHONY: bin/fc2-live-dl-go-static.exe
bin/fc2-live-dl-go-static.exe: $(GO_SRCS)
	CGO_ENABLED=1 \
	GOOS=windows \
	GOARCH=amd64 \
	go build -trimpath -ldflags '-X main.version=${VERSION}+${RELEASE} -linkmode external -s -w -extldflags "-static"' -o "$@" ./main.go

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
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: lint
lint: $(golint)
	$(golint) run ./...

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: package
package: target/alpine-edge \
	target/el8 \
	target/el9 \
	target/fc37 \
	target/fc38 \
	target/fc39 \
	target/deb10 \
	target/deb11 \
	target/deb12 \
	target/ubuntu20 \
	target/ubuntu22 \
	target/static \
	target/static-windows \
	target/checksums.txt \
	target/checksums.md

target/checksums.txt: target/alpine-edge \
	target/el8 \
	target/el9 \
	target/fc37 \
	target/fc38 \
	target/fc39 \
	target/deb10 \
	target/deb11 \
	target/deb12 \
	target/ubuntu20 \
	target/ubuntu22 \
	target/static \
	target/static-windows
	sha256sum -b $(addsuffix /*,$^) | sed 's|target/.*/||' > $@

target/checksums.md: target/checksums.txt
	@echo "### SHA256 Checksums" > $@
	@echo >> $@
	@echo "\`\`\`" >> $@
	@cat $< >> $@
	@echo "\`\`\`" >> $@

target/alpine-edge:
	podman manifest rm localhost/builder:alpine-edge || true
	podman build \
		--manifest localhost/builder:alpine-edge \
		--build-arg VERSION=${VERSION} \
		--build-arg RELEASE=r${RELEASE} \
		--build-arg IMAGE=docker.io/library/alpine:edge \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8 \
		-f Dockerfile.apk .
	mkdir -p ./target/alpine-edge
	podman run --rm \
		-e DEPENDS_LIBAVCODEC=ffmpeg-libavcodec \
		-e DEPENDS_LIBAVFORMAT=ffmpeg-libavformat \
		-e DEPENDS_LIBAVUTIL=ffmpeg-libavutil \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		localhost/builder:alpine-edge package \
		--config /work/nfpm.yaml \
		--target /target/alpine-edge/ \
		--packager apk
	podman run --rm \
		-e DEPENDS_LIBAVCODEC=ffmpeg-libavcodec \
		-e DEPENDS_LIBAVFORMAT=ffmpeg-libavformat \
		-e DEPENDS_LIBAVUTIL=ffmpeg-libavutil \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		localhost/builder:alpine-edge package \
		--config /work/nfpm.yaml \
		--target /target/alpine-edge/ \
		--packager apk

target/el8:
	podman manifest rm localhost/builder:el8 || true
	podman build \
		--manifest localhost/builder:el8 \
		--build-arg VERSION=${VERSION} \
		--build-arg RELEASE=${RELEASE}.el8 \
		--build-arg IMAGE=docker.io/library/rockylinux:8 \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8 \
		-f Dockerfile.rpm .
	mkdir -p ./target/el8
	podman run --rm \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		localhost/builder:el8 package \
		--config /work/nfpm.yaml \
		--target /target/el8/ \
		--packager rpm
	podman run --rm \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		localhost/builder:el8 package \
		--config /work/nfpm.yaml \
		--target /target/el8/ \
		--packager rpm

target/el9:
	podman manifest rm localhost/builder:el9 || true
	podman build \
		--manifest localhost/builder:el9 \
		--build-arg VERSION=${VERSION} \
		--build-arg RELEASE=${RELEASE}.el9 \
		--build-arg IMAGE=docker.io/library/rockylinux:9 \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8 \
		-f Dockerfile.rpm .
	mkdir -p ./target/el9
	podman run --rm \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		localhost/builder:el9 package \
		--config /work/nfpm.yaml \
		--target /target/el9/ \
		--packager rpm
	podman run --rm \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		localhost/builder:el9 package \
		--config /work/nfpm.yaml \
		--target /target/el9/ \
		--packager rpm

target/fc37:
	podman manifest rm localhost/builder:fc37 || true
	podman build \
		--manifest localhost/builder:fc37 \
		--build-arg VERSION=${VERSION} \
		--build-arg RELEASE=${RELEASE}.fc37 \
		--build-arg IMAGE=docker.io/library/fedora:37 \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8 \
		-f Dockerfile.fedora .
	mkdir -p ./target/fc37
	podman run --rm \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		localhost/builder:fc37 package \
		--config /work/nfpm.yaml \
		--target /target/fc37/ \
		--packager rpm
	podman run --rm \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		localhost/builder:fc37 package \
		--config /work/nfpm.yaml \
		--target /target/fc37/ \
		--packager rpm

target/fc38:
	podman manifest rm localhost/builder:fc38 || true
	podman build \
		--manifest localhost/builder:fc38 \
		--build-arg VERSION=${VERSION} \
		--build-arg RELEASE=${RELEASE}.fc38 \
		--build-arg IMAGE=docker.io/library/fedora:38 \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8 \
		-f Dockerfile.fedora .
	mkdir -p ./target/fc38
	podman run --rm \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		localhost/builder:fc38 package \
		--config /work/nfpm.yaml \
		--target /target/fc38/ \
		--packager rpm
	podman run --rm \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		localhost/builder:fc38 package \
		--config /work/nfpm.yaml \
		--target /target/fc38/ \
		--packager rpm

target/fc39:
	podman manifest rm localhost/builder:fc39 || true
	podman build \
		--manifest localhost/builder:fc39 \
		--build-arg VERSION=${VERSION} \
		--build-arg RELEASE=${RELEASE}.fc39 \
		--build-arg IMAGE=docker.io/library/fedora:39 \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8 \
		-f Dockerfile.fedora .
	mkdir -p ./target/fc39
	podman run --rm \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		localhost/builder:fc39 package \
		--config /work/nfpm.yaml \
		--target /target/fc39/ \
		--packager rpm
	podman run --rm \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		localhost/builder:fc39 package \
		--config /work/nfpm.yaml \
		--target /target/fc39/ \
		--packager rpm

target/deb10:
	podman manifest rm localhost/builder:deb10 || true
	podman build \
		--manifest localhost/builder:deb10 \
		--build-arg VERSION=${VERSION} \
		--build-arg RELEASE=${RELEASE}+deb10u1 \
		--build-arg IMAGE=docker.io/library/debian:10 \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8 \
		-f Dockerfile.deb .
	mkdir -p ./target/deb10
	podman run --rm \
		-e DEPENDS_LIBAVCODEC=libavcodec58 \
		-e DEPENDS_LIBAVFORMAT=libavformat58 \
		-e DEPENDS_LIBAVUTIL=libavutil56 \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		localhost/builder:deb10 package \
		--config /work/nfpm.yaml \
		--target /target/deb10/ \
		--packager deb
	podman run --rm \
		-e DEPENDS_LIBAVCODEC=libavcodec58 \
		-e DEPENDS_LIBAVFORMAT=libavformat58 \
		-e DEPENDS_LIBAVUTIL=libavutil56 \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		localhost/builder:deb10 package \
		--config /work/nfpm.yaml \
		--target /target/deb10/ \
		--packager deb

target/deb11:
	podman manifest rm localhost/builder:deb11 || true
	podman build \
		--manifest localhost/builder:deb11 \
		--build-arg VERSION=${VERSION} \
		--build-arg RELEASE=${RELEASE}+deb11u1 \
		--build-arg IMAGE=docker.io/library/debian:11 \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8 \
		-f Dockerfile.deb .
	mkdir -p ./target/deb11
	podman run --rm \
		-e DEPENDS_LIBAVCODEC=libavcodec58 \
		-e DEPENDS_LIBAVFORMAT=libavformat58 \
		-e DEPENDS_LIBAVUTIL=libavutil56 \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		localhost/builder:deb11 package \
		--config /work/nfpm.yaml \
		--target /target/deb11/ \
		--packager deb
	podman run --rm \
		-e DEPENDS_LIBAVCODEC=libavcodec58 \
		-e DEPENDS_LIBAVFORMAT=libavformat58 \
		-e DEPENDS_LIBAVUTIL=libavutil56 \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		localhost/builder:deb11 package \
		--config /work/nfpm.yaml \
		--target /target/deb11/ \
		--packager deb

target/deb12:
	podman manifest rm localhost/builder:deb12 || true
	podman build \
		--manifest localhost/builder:deb12 \
		--build-arg VERSION=${VERSION} \
		--build-arg RELEASE=${RELEASE}+deb12u1 \
		--build-arg IMAGE=docker.io/library/debian:12 \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8 \
		-f Dockerfile.deb .
	mkdir -p ./target/deb12
	podman run --rm \
		-e DEPENDS_LIBAVCODEC=libavcodec59 \
		-e DEPENDS_LIBAVFORMAT=libavformat59 \
		-e DEPENDS_LIBAVUTIL=libavutil57 \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		localhost/builder:deb12 package \
		--config /work/nfpm.yaml \
		--target /target/deb12/ \
		--packager deb
	podman run --rm \
		-e DEPENDS_LIBAVCODEC=libavcodec59 \
		-e DEPENDS_LIBAVFORMAT=libavformat59 \
		-e DEPENDS_LIBAVUTIL=libavutil57 \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		localhost/builder:deb12 package \
		--config /work/nfpm.yaml \
		--target /target/deb12/ \
		--packager deb

target/ubuntu20:
	podman manifest rm localhost/builder:ubuntu20 || true
	podman build \
		--manifest localhost/builder:ubuntu20 \
		--build-arg VERSION=${VERSION} \
		--build-arg RELEASE=${RELEASE}ubuntu20.04 \
		--build-arg IMAGE=docker.io/library/ubuntu:20.04 \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8 \
		-f Dockerfile.deb .
	mkdir -p ./target/ubuntu20
	podman run --rm \
		-e DEPENDS_LIBAVCODEC=libavcodec58 \
		-e DEPENDS_LIBAVFORMAT=libavformat58 \
		-e DEPENDS_LIBAVUTIL=libavutil56 \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		localhost/builder:ubuntu20 package \
		--config /work/nfpm.yaml \
		--target /target/ubuntu20/ \
		--packager deb
	podman run --rm \
		-e DEPENDS_LIBAVCODEC=libavcodec58 \
		-e DEPENDS_LIBAVFORMAT=libavformat58 \
		-e DEPENDS_LIBAVUTIL=libavutil56 \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		localhost/builder:ubuntu20 package \
		--config /work/nfpm.yaml \
		--target /target/ubuntu20/ \
		--packager deb

target/ubuntu22:
	podman manifest rm localhost/builder:ubuntu22 || true
	podman build \
		--manifest localhost/builder:ubuntu22 \
		--build-arg VERSION=${VERSION} \
		--build-arg RELEASE=${RELEASE}ubuntu22.04 \
		--build-arg IMAGE=docker.io/library/ubuntu:22.04 \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8 \
		-f Dockerfile.deb .
	mkdir -p ./target/ubuntu22
	podman run --rm \
		-e DEPENDS_LIBAVCODEC=libavcodec59 \
		-e DEPENDS_LIBAVFORMAT=libavformat59 \
		-e DEPENDS_LIBAVUTIL=libavutil57 \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		localhost/builder:ubuntu22 package \
		--config /work/nfpm.yaml \
		--target /target/ubuntu22/ \
		--packager deb
	podman run --rm \
		-e DEPENDS_LIBAVCODEC=libavcodec59 \
		-e DEPENDS_LIBAVFORMAT=libavformat59 \
		-e DEPENDS_LIBAVUTIL=libavutil57 \
		-v $(shell pwd)/nfpm.yaml:/work/nfpm.yaml \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		localhost/builder:ubuntu22 package \
		--config /work/nfpm.yaml \
		--target /target/ubuntu22/ \
		--packager deb

target/static:
	podman manifest rm localhost/builder:static || true
	podman build \
		--manifest localhost/builder:static \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8,linux/riscv64 \
		--target builder \
		-f Dockerfile.static .
	mkdir -p ./target/static
	podman run --rm \
		-v $(shell pwd)/target/:/target/ \
		--arch amd64 \
		localhost/builder:static mv /work/bin/fc2-live-dl-go-static /target/static/fc2-live-dl-go-linux-amd64
	podman run --rm \
		-v $(shell pwd)/target/:/target/ \
		--arch arm64 \
		--variant v8 \
		localhost/builder:static mv /work/bin/fc2-live-dl-go-static /target/static/fc2-live-dl-go-linux-arm64
	podman run --rm \
		-v $(shell pwd)/target/:/target/ \
		--arch riscv64 \
		--variant v8 \
		localhost/builder:static mv /work/bin/fc2-live-dl-go-static /target/static/fc2-live-dl-go-linux-riscv64
	./assert-arch.sh

target/static-windows:
	podman build \
		-t localhost/builder:static-windows \
		-f Dockerfile.static-windows .
	mkdir -p ./target/static-windows
	podman run --rm \
		-v $(shell pwd)/target/:/target/ \
		localhost/builder:static-windows mv /work/bin/fc2-live-dl-go-static.exe /target/static-windows/fc2-live-dl-go-windows-amd64.exe

.PHONY: docker-static
docker-static:
	podman manifest rm ghcr.io/darkness4/fc2-live-dl-go:latest || true
	podman build \
		--manifest ghcr.io/darkness4/fc2-live-dl-go:latest \
		--jobs=2 --platform=linux/amd64,linux/arm64/v8 \
		-f Dockerfile.static .
	podman manifest push --all ghcr.io/darkness4/fc2-live-dl-go:latest "docker://ghcr.io/darkness4/fc2-live-dl-go:latest"
	podman manifest push --all ghcr.io/darkness4/fc2-live-dl-go:latest "docker://ghcr.io/darkness4/fc2-live-dl-go:${VERSION_NO_V}-${RELEASE}"
	podman manifest push --all ghcr.io/darkness4/fc2-live-dl-go:latest "docker://ghcr.io/darkness4/fc2-live-dl-go:dev"

.PHONY: docker-static-base
docker-static-base:
	podman manifest rm ghcr.io/darkness4/fc2-live-dl-go:latest-static-base || true
	podman build \
		--manifest ghcr.io/darkness4/fc2-live-dl-go:latest-static-base \
		--platform=linux/amd64,linux/arm64/v8,linux/riscv64 \
		-f Dockerfile.static-base .
	podman manifest push --all --rm ghcr.io/darkness4/fc2-live-dl-go:latest-static-base "docker://ghcr.io/darkness4/fc2-live-dl-go:latest-static-base"

.PHONY: docker-static-windows-base
docker-static-windows-base:
	podman build \
		-t ghcr.io/darkness4/fc2-live-dl-go:latest-static-windows-base \
		-f Dockerfile.static-windows-base .
	podman push ghcr.io/darkness4/fc2-live-dl-go:latest-static-windows-base

.PHONY: version
version:
	echo version=$(VERSION) release=$(RELEASE)
