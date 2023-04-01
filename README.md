# fc2-live-dl-go

Inspired by [HoloArchivists/fc2-live-dl](https://github.com/HoloArchivists/fc2-live-dl), incredible work.

## Motivation

Needed to be multiplatform, run on raspberry pi and be light.

Also written for training.

Business code maintenance may not planned for the long term. However, PRs are accepted and will be reviewed quickly.

Bugs will be fixed when seen.

## Differences and similarities between HoloArchivists/fc2-live-dl and this version

Differences:

- Re-written Go.
- Proper error handling.
- No priority queue for downloading, no multithread download. I tried a thread safe priority queue, but it was way too slow.
- Low CPU usage at runtime.
- Uses libavformat over executing CLI commands on FFmpeg.

Similarities:

- Business logic. The code follows a similar order with a similar configuration. This means that updates and fixes can be passed from one project to the other.

## Installation

### Static binaries (amd64, arm64) (~80MB)

Prebuilt binaries using ffmpeg static libraries are [available](https://github.com/Darkness4/fc2-live-dl-go/releases/latest) on the GitHub Releases tab.

### Linked binaries (Debian, Ubuntu, EL) (~7MB)

Prebuilt binaries using ffmpeg shared libraries are [available](https://github.com/Darkness4/fc2-live-dl-go/releases/latest) on the GitHub Releases tab.

**Debian/Ubuntu**

Download the package for the corresponding distribution (you can find you distribution bu running `cat /etc/os-release`), and install it:

```shell
dpkg -i fc2-live-dl-go_*.deb
```

**Enterprise Linux (RHEL, RockyLinux, AlmaLinux)/Fedora**

Download the package for the corresponding distribution (you can find you distribution bu running `cat /etc/os-release`), and install it:

```shell
rpm -Uvh fc2-live-dl-go_*.rpm
```

### Docker (amd64, arm64, s390x, ppc64le, riscv64) (~109 MB)

The container has been fine-tuned, so it is recommended to use it.

```shell
docker pull ghcr.io/darkness4/fc2-live-dl-go:latest
```

Usage:

```shell
docker run -it --rm ghcr.io/darkness4/fc2-live-dl-go:latest [global options] [command] [command options]
```

```shell
mkdir -p $(pwd)/out
docker run -it --rm \
  -v $(pwd)/out:/out \
  ghcr.io/darkness4/fc2-live-dl-go:latest download \
    --keep-intermediates \
    --extract-audio \
    --format "/out/{{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}" 91544481
```

Make sure to change the UID if you run docker as root:

```shell
mkdir -p $(pwd)/out
chown 1000:1000 $(pwd)/out
docker run -it --rm \
  -u 1000:1000 \
  -v $(pwd)/out:/out \
  ghcr.io/darkness4/fc2-live-dl-go:latest download \
    --keep-intermediates \
    --extract-audio \
    --format "/out/{{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}" 91544481
```

### Deployments (Kubernetes/Docker-Compose)

Examples of deployments manifests are stored in the [`./deployments`](./deployments) directory.

### Build from source

#### Linux

`fc2-live-dl-go` uses the shared libraries of ffmpeg (more precisely `libavformat`, `libavcodec`, `libavutil`).

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

## Usage

### Download a single live fc2 stream

```shell
fc2-live-dl-go [global options] download [command options] channelID
```

```shell
OPTIONS:
   --quality value  Quality of the stream to download.
      Available latency options: 150Kbps, 400Kbps, 1.2Mbps, 2Mbps, 3Mbps, sound. (default: "1.2Mbps")
   --latency value  Stream latency. Select a higher latency if experiencing stability issues.
      Available latency options: low, high, mid. (default: "mid")
   --format value  Golang templating format. Available fields: ChannelID, ChannelName, Date, Time, Title, Ext, Labels.Key.
      Available format options:
        ChannelID: ID of the broadcast
        ChannelName: broadcaster's profile name
        Date: local date YYYY-MM-DD
        Time: local time HHMMSS
        Ext: file extension
        Title: title of the live broadcast
        Labels.Key: custom labels
       (default: "{{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}")
   --max-packet-loss value             Allow a maximum of packet loss before aborting stream download. (default: 200)
   --no-remux                          Do not remux recordings into mp4/m4a after it is finished. (default: false)
   --keep-intermediates, -k            Keep the raw .ts recordings after it has been remuxed. (default: false)
   --extract-audio, -x                 Generate an audio-only copy of the stream. (default: false)
   --cookies value                     Path to a cookies file.
   --write-chat                        Save live chat into a json file. (default: false)
   --write-info-json                   Dump output stream information into a json file. (default: false)
   --write-thumbnail                   Download thumbnail into a file. (default: false)
   --no-wait                           Don't wait until the broadcast goes live, then start recording. (default: false)
   --wait-for-quality-max-tries value  If the requested quality is not available, keep retrying before falling back to the next best quality. (default: 10)
   --poll-interval value               How many seconds between checks to see if broadcast is live. (default: 5s)
   --max-tries value                   On failure, keep retrying (cancellation and end of stream will still force abort). (default: 10)
   --loop                              Continue to download streams indefinitely. (default: false)
   --help, -h                          show help

GLOBAL OPTIONS:
   --debug        (default: false) [$DEBUG]
   --help, -h     show help
   --version, -v  print the version
```

### Download multiple live fc2 streams

```shell
fc2-live-dl-go [global options] watch [command options]
```

```shell
OPTIONS:
   --config value, -c value  Config file path. (required)
   --help, -h                show help

GLOBAL OPTIONS:
   --debug        (default: false) [$DEBUG]
   --help, -h     show help
   --version, -v  print the version
```

Configuration Example:

```yaml
---
defaultParams:
  ## Quality of the stream to download.
  ##
  ## Available latency options: 150Kbps, 400Kbps, 1.2Mbps, 2Mbps, 3Mbps, sound. (default: "1.2Mbps")
  quality: 1.2Mbps
  ## Stream latency. Select a higher latency if experiencing stability issues.
  ##
  ## Available latency options: low, high, mid. (default: "mid")
  latency: mid
  ## Output format. Uses Golang templating format.
  ##
  ## Available fields: ChannelID, ChannelName, Date, Time, Title, Ext, Labels.Key.
  ## Available format options:
  ##   ChannelID: ID of the broadcast
  ##   ChannelName: broadcaster's profile name
  ##   Date: local date YYYY-MM-DD
  ##   Time: local time HHMMSS
  ##   Ext: file extension
  ##   Title: title of the live broadcast
  ##   Labels.Key: custom labels
  ## (default: "{{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}")
  outFormat: '{{ .ChannelName }} {{ .Labels.EnglishName }}/{{ .Date }} {{ .Title }}.{{ .Ext }}'
  ## Allow a maximum of packet loss before aborting stream download. (default: 200)
  packetLossMax: 200
  ## Save live chat into a json file. (default: false)
  writeChat: false
  ## Dump output stream information into a json file. (default: false)
  writeInfoJson: false
  ## Download thumbnail into a file. (default: false)
  writeThumbnail: false
  ## Wait until the broadcast goes live, then start recording. (default: true)
  waitForLive: true
  ## If the requested quality is not available, keep retrying before falling back to the next best quality. (default: 10)
  waitForQualityMaxTries: 10
  ## How many seconds between checks to see if broadcast is live. (default: 5s)
  waitPollInterval: '5s'
  ## Path to a cookies file. Format is a netscape cookies file.
  cookiesFile: ''
  ## Remux recordings into mp4/m4a after it is finished. (default: true)
  remux: true
  ## Keep the raw .ts recordings after it has been remuxed. (default: false)
  keepIntermediates: false
  ## Generate an audio-only copy of the stream. (default: false)
  extractAudio: true
  ## Map of key/value strings.
  ##
  ## The value of the label can be invoked in the go template by using {{ .Labels.Key }}.
  labels: {}

## A list of channels.
##
## The keys are the channel IDs.
channels:
  '40740626':
    labels:
      EnglishName: Komae Nadeshiko
  '72364867':
    labels:
      EnglishName: Uno Sakura
  '81840800':
    labels:
      EnglishName: Ronomiya Hinagiku
  '91544481':
    labels:
      EnglishName: Necoma Karin
```

## Credits

Many thanks to https://github.com/hizkifw and contributors to the [HoloArchivists/fc2-live-dl](https://github.com/HoloArchivists/fc2-live-dl) project for their excellent source code.
