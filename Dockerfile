# syntax=docker/dockerfile:1
FROM registry-1.docker.io/library/alpine:edge as builder

RUN apk add --no-cache go ffmpeg-dev gcc musl-dev make git

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

ARG TARGETOS TARGETARCH
COPY . .

RUN CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH make bin/fc2-live-dl-go

# ---
FROM registry-1.docker.io/library/alpine:edge
RUN apk add --no-cache ca-certificates ffmpeg-libavformat ffmpeg-libavcodec ffmpeg-libavutil

WORKDIR /app

COPY --from=builder /build/bin/fc2-live-dl-go /fc2-live-dl-go

ENTRYPOINT [ "/fc2-live-dl-go" ]
