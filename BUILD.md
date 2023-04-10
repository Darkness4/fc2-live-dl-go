# Build from source

## Linux (dynamically-linked binaries)

`fc2-live-dl-go` uses the shared libraries of ffmpeg (more precisely `libavformat`, `libavcodec` and `libavutil`).

1. Install the development packages `libavformat-dev libavcodec-dev libavutil-dev` or `ffmpeg-devel` depending on your OS distribution.

2. Install [Go](https://go.dev)

3. Run:

   ```shell
   go install github.com/Darkness4/fc2-live-dl-go@latest
   ```

   Or `git clone` this repository and run `make` which basically runs:

   ```shell
   CGO_ENABLED=1 go build -o "$@" ./main.go
   ```

4. Then, you can remove the development packages and install the runtime packages. The runtime packages can be named `libavcodec` (fedora, debian) or `ffmpeg-libavcodec` (alpine). If you don't want to search, you can just install `ffmpeg`.

## Linux (static binaries)

### With Podman

To build static binaries, we use Podman (Docker) with Gentoo Musl Linux containers.

You can run:

```shell
make target/static
```

If you wish to build a static executable. Note that it will build an arm64 version too. If you want to build just for your platform you can run:

```shell
podman build \
   -f localhost/builder:static \
   --target builder \
   -f Dockerfile.static .
mkdir -p ./target/static
podman run --rm \
   -v $(pwd)/target/:/target/ \
   localhost/builder:static mv /work/bin/fc2-live-dl-go-static /target/static/fc2-live-dl-go-linux-amd64
```

### Manually

Sadly, we won't help you on this one.

The idea would be to:

- Use a musl linux compiler
- Compile ffmpeg and its dependencies statically
- Compile the application with:

```shell
CGO_ENABLED=1 go build -s -w -extldflags "-lswresample -static"' -o "$@" ./main.go
```

## Windows (static binaries)

### From Linux for Windows

We compile a static executable instead of a dynamically-linked executable for windows.

#### With Podman/Docker

It is recommended to use [Dockerfile.static-windows](Dockerfile.static-windows) instead of doing everything manually.

You can simply run:

```shell
make target/static-windows
```

Note that we use podman, but you can actually alias podman with docker.

#### Manually

We use [M cross environment (MXE)](https://mxe.cc).

1. Install MXE requirements by following [this guide](https://mxe.cc/#requirements). You should also install `mingw-w64`.

2. Step MXE by following these instructions:

   ```shell
   cd /opt
   git clone https://github.com/mxe/mxe mxe
   cd mxe
   echo "MXE_TARGETS := i686-w64-mingw32.static" >> settings.mk

   make gcc ffmpeg JOBS=$(nproc)

   export PATH=/opt/mxe/usr/bin:${PATH}
   export CC=x86_64-w64-mingw32.static-gcc
   export CXX=x86_64-w64-mingw32.static-g++
   export PKG_CONFIG=x86_64-w64-mingw32.static-pkg-config
   ```

3. Then you can cross-compile:

   ```shell
   make bin/fc2-live-dl-go-static.exe
   ```

### Native build

1. Install MSYS2 from [www.msys2.org](https://www.msys2.org/).

2. Start a MinGW-w64 shell with `mingw64.exe`.

3. Update MSYS2 to prevent errors during post-install:

   ```shell
   # Check for core updates. If instructed, close the shell window and reopen it
   # before continuing.
   pacman -Syu

   # Update everything else
   pacman -Su
   ```

4. Install the dependencies:

   ```shell
   pacman -S git make $MINGW_PACKAGE_PREFIX-{go, gcc, pkgconf, ffmpeg}

   export GOROOT=/mingw64/lib/go.exe
   export GOPATH=/mingw64
   export CC=/mingw64/bin/gcc.exe
   export CXX=/mingw64/bin/g++.exe
   export PKG_CONFIG=/mingw64/bin/pkg-config.exe
   ```

5. Then you can compile:

   ```shell
   make bin/fc2-live-dl-go-static.exe
   ```
