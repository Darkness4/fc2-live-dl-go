# Build from source

> ![WARNING]
>
> Other build methods like native builds are no more supported. Prefer using Podman for a better experience.
> You can still set up a development environment by simply installing Go, but won't be able to compile on Windows or MacOS.
>
> Docker buildx can also be used, but we won't help you with that.

## Linux (dynamically-linked binaries) or development environment setup

`fc2-live-dl-go` uses the shared libraries of ffmpeg (more precisely `libavformat`, `libavcodec` and `libavutil`).

1. Install the development packages `libavformat-dev libavcodec-dev libavutil-dev` or `ffmpeg-devel` depending on your OS distribution.

2. Install [Go](https://go.dev)

3. Run:

   ```shell
   go install github.com/Darkness4/fc2-live-dl-go@latest
   ```

   Or `git clone` this repository and run:

   ```shell
   make
   ```

   which basically runs:

   ```shell
   # make
   CGO_ENABLED=1 go build -o "bin/fc2-live-dl-go" ./main.go
   ```

## Linux (static binaries)

To build static binaries, we use Podman (Docker) with Gentoo Musl Linux containers.

You can run:

```shell
make target-static
```

If you wish to build a static executable. Note that it will build an arm64 version too. If you want to build just for your platform you can run:

```shell
podman build \
   --target export \
   --output=type=local,dest=./target \
   -f Dockerfile.static .
```

The binary will be in the `target` directory.

## Windows

It is recommended to use [Dockerfile.static-windows](Dockerfile.static-windows) instead of doing everything manually.

You can simply run:

```shell
make target-static-windows
```

Which will run:

```shell
podman build \
   --target export \
   --output=type=local,dest=./target \
   -f Dockerfile.static-windows .
```

The binary will be in the `target` directory.

## MacOS (>10.15)

It is recommended to use [Dockerfile.static-macos](Dockerfile.darwin) instead of doing everything manually.

You can simply run:

```shell
make target-darwin
```

Which will run:

```shell
podman build \
   --target export \
   --output=type=local,dest=./target \
   -f Dockerfile.darwin .
```

The binary will be in the `target` directory.
