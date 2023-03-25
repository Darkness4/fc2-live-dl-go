# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM registry-1.docker.io/library/golang:1.20.2 as builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

ARG TARGETOS TARGETARCH VERSION=dev
RUN --mount=target=/build/ \
  --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg \
  CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -ldflags "-X main.version=${VERSION}" -o /out/app ./main.go

# ---
FROM registry-1.docker.io/library/alpine:latest
RUN apk add --no-cache ffmpeg ca-certificates

ARG TARGETOS TARGETARCH
ENV TINI_VERSION v0.19.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static-$TARGETARCH /tini
RUN chmod +x /tini

RUN mkdir /app
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app

COPY --from=builder /out/app .

RUN chown -R app:app .
USER app

ENTRYPOINT [ "/tini", "--", "./app" ]
