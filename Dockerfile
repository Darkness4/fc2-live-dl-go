# syntax=docker/dockerfile:1
FROM registry-1.docker.io/library/alpine:edge as builder

RUN apk add --no-cache go ffmpeg-dev gcc musl-dev

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

ARG TARGETOS TARGETARCH VERSION=dev
COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg \
  CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -ldflags "-X main.version=${VERSION}" -o /out/app ./main.go

# ---
FROM registry-1.docker.io/library/alpine:edge
RUN apk add --no-cache ca-certificates ffmpeg-libavformat ffmpeg-libavcodec ffmpeg-libavutil

WORKDIR /app

COPY --from=builder /out/app .

ENTRYPOINT [ "/app/app" ]
