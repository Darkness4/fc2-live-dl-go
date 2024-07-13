# fc2-live-dl-go

Automatically download FC2 livestream. Written in Go.

## Motivation

Although [HoloArchivists/fc2-live-dl](https://github.com/HoloArchivists/fc2-live-dl) did most of the work, I wanted something lightweight that could run on a Raspberry Pi. While, I could have built a Docker image for arm64 based on the [HoloArchivists/fc2-live-dl](https://github.com/HoloArchivists/fc2-live-dl) source code, I also wanted to be light in terms of size, RAM and CPU usage. So I rewrote everything in Go. It was also a good way of training myself in the use of FFI.

## Differences and similarities between HoloArchivists/fc2-live-dl and this version

Differences:

- Rewritten Go, which provides a better error handling.
- No priority queue for download, no multithreaded download. Sequential download as per standard with m3u8 downloader.
- Low CPU usage at runtime, minimal IO block.
- Uses libav C API rather than running CLI commands on FFmpeg.
- Offering static binaries with no dependencies needed on the host.
- Can concatenate previous recordings if the recordings was splitted due to crashes.
- Can automatically upgrade quality to 3Mbps during download.
- Very light in size even with static binaries.
- Minor fixes like graceful exit and crash recovery.
- Session cookies auto-refresh.
- YAML/JSON config file.
- Notification via [shoutrrr](https://github.com/containrrr/shoutrrr) which supports multiple notification services.

Similarities:

- Business logic. The code follows a similar order with a similar configuration. This means that updates and fixes can be transferred from one project to another.

## Installation

### Static binaries (amd64, arm64) (~20MB)

Prebuilt binaries using FFmpeg static libraries are [available](https://github.com/Darkness4/fc2-live-dl-go/releases/latest) on the GitHub Releases tab.

**Linux**

Static binaries are generated using the file [Dockerfile.static-base](Dockerfile.static-base) and [Dockerfile.static](Dockerfile.static).

You can customize FFmpeg by editing [Dockerfile.static-base](Dockerfile.static-base).

**Darwin**

Partial static binaries are generated using the file [Dockerfile.darwin-base](Dockerfile.darwin-base) and [Dockerfile.darwin](Dockerfile.darwin).

You can customize FFmpeg by editing [Dockerfile.darwin-base](Dockerfile.darwin-base).

Do note that the Darwin binaries are also linked to `libSystem`, which adds a requirement on the OS version.

The requirements are:

- For x86_64, the OS X version must be greater or equal than 10.5.
- For ARM64v8, the OS X version must be greater or equal than 11.0.

### Docker (amd64, arm64, riscv64) (~22 MB)

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

### Install from source (~13M)

See [BUILD.md](BUILD.md).

### Deployments (Kubernetes/Docker-Compose)

Examples of deployments manifests are stored in the [`./deployments`](./deployments) directory.

## Usage

### Download a single live fc2 stream

```shell
fc2-live-dl-go [global options] download [command options] channelID
```

```shell
OPTIONS:
   Cleaning Routine:

   --eligible-for-cleaning-age value, --cleaning-age value  Minimum age of .combined files to be eligible for cleaning. (default: 48h0m0s)
   --scan-directory value                                   Directory to be scanned for .ts files to be deleted after concatenation.

   Polling:

   --loop                 Continue to download streams indefinitely. (default: false)
   --max-tries value      On failure, keep retrying (cancellation and end of stream will still force abort). (default: 10)
   --no-wait              Don't wait until the broadcast goes live, then start recording. (default: false)
   --poll-interval value  How many seconds between checks to see if broadcast is live. (default: 5s)

   Post-Processing:

   --concat             Concatenate and remux with previous recordings after it is finished.  (default: false)
   --extract-audio, -x  Generate an audio-only copy of the stream. (default: false)
   --format value       Golang templating format. Available fields: ChannelID, ChannelName, Date, Time, Title, Ext, Labels.Key.
Available format options:
  ChannelID: ID of the broadcast
  ChannelName: broadcaster's profile name
  Date: local date YYYY-MM-DD
  Time: local time HHMMSS
  Ext: file extension
  Title: title of the live broadcast
  Labels.Key: custom labels
 (default: "{{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}")
   --keep-intermediates, -k  Keep the raw .ts recordings after it has been remuxed. (default: false)
   --max-packet-loss value   Allow a maximum of packet loss before aborting stream download. (default: 20)
   --no-delete-corrupted     Delete corrupted .ts recordings. (default: false)
   --no-remux                Do not remux recordings into mp4/m4a after it is finished. (default: false)
   --remux-format value      Remux format of the video. (default: "mp4")

   Streaming:

   --allow-quality-upgrade  If the requested quality is not available, allow upgrading to a better quality. (default: false)
   --cookies-file value     Path to a cookies file. Format is a netscape cookies file.
   --latency value          Stream latency. Select a higher latency if experiencing stability issues.
Available latency options: low, high, mid. (default: "mid")
   --poll-quality-upgrade-interval value  How many seconds between checks to see if a better quality is available. (default: 10s)
   --quality value                        Quality of the stream to download.
Available latency options: 150Kbps, 400Kbps, 1.2Mbps, 2Mbps, 3Mbps, sound. (default: "3Mbps")
   --wait-for-quality-max-tries value  If the requested quality is not available, keep retrying before falling back to the next best quality. (default: 60)
   --write-chat                        Save live chat into a json file. (default: false)
   --write-info-json                   Dump output stream information into a json file. (default: false)
   --write-thumbnail                   Download thumbnail into a file. (default: false)

GLOBAL OPTIONS:
   --debug        (default: false) [$DEBUG]
   --log-json     (default: false) [$LOG_JSON]
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
   --pprof.listen-address value  (default: ":3000")
   --help, -h                show help

GLOBAL OPTIONS:
   --debug        (default: false) [$DEBUG]
   --log-json     (default: false) [$LOG_JSON]
   --help, -h     show help
   --version, -v  print the version
```

When running the watcher, the program opens the port `3000/tcp` for debugging. You can access the pprof dashboard by accessing at `http://<host>:3000/debug/pprof/` or by using `go tool pprof http://host:port/debug/pprof/profile`.

**A status page is also accessible at `http://<host>:3000/`.**

To configure the watcher, you must provide a configuration file. The configuration file is in YAML format. See the [config.yaml](config.yaml) file for an example.

<details>

<summary>Configuration Example</summary>

```yaml
---
defaultParams:
  ## Quality of the stream to download.
  ##
  ## Available latency options: 150Kbps, 400Kbps, 1.2Mbps, 2Mbps, 3Mbps, sound. (default: "3Mbps")
  quality: 3Mbps
  ## Stream latency. Select a higher latency if experiencing stability issues.
  ##
  ## Available latency options: low, high, mid. (default: "mid")
  latency: mid
  ## Output format. Uses Golang templating format.
  ##
  ## Available fields: ChannelID, ChannelName, Date, Time, Title, Ext, Labels.Key.
  ## Available format options:
  ##   ChannelID: sanitized ID of the broadcast
  ##   ChannelName: sanitized broadcaster's profile name
  ##   Date: local date YYYY-MM-DD
  ##   Time: local time HHMMSS
  ##   Ext: file extension
  ##   Title: sanitized title of the live broadcast
  ##   Metadata (object): the full FC2 metadata (see fc2/fc2_api_objects.go for the available field)
  ##   Labels.Key: custom labels
  ## (default: "{{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}")
  outFormat: '{{ .ChannelName }} {{ .Labels.EnglishName }}/{{ .Date }} {{ .Title }}.{{ .Ext }}'
  ## Allow a maximum of packet loss before aborting stream download. (default: 20)
  packetLossMax: 20
  ## Save live chat into a json file. (default: false)
  writeChat: false
  ## Dump output stream information into a json file. (default: false)
  writeInfoJson: false
  ## Download thumbnail into a file. (default: false)
  writeThumbnail: false
  ## Wait until the broadcast goes live, then start recording. (default: true)
  waitForLive: true
  ## If the requested quality is not available, keep retrying before falling
  ## back to the next best quality. (default: 60)
  ##
  ## There is a 1 second delay between each retry. The value must be big enough
  ## so that the best quality (3Mbps) is available. If your streamer takes more
  ## than expected to prepare, you should increase this value.
  waitForQualityMaxTries: 60
  ## EXPERIMENTAL: Allow quality upgrade during download if the requested
  ## quality is not "yet" available. (default: false)
  ##
  ## If the requested quality is not available, the downloader will fallback to
  ## the best quality available. However, it is possible that the streamer will
  ## upgrade the quality during the stream. FC2 often "waits" for the stream to
  ## be stable before upgrading the quality.
  ##
  ## If this option is enabled, the downloader will periodically check if the
  ## quality has been upgraded. If the quality has been upgraded, the downloader
  ## will switch to the new quality. **A cut off will be present in the recording.**
  ##
  ## If this option is enabled, it is recommended to:
  ##
  ## - Reduce waitForQualityMaxTries to 10s.
  ## - Enable Remux or Concat to fix mpegts discontinuities.
  allowQualityUpgrade: false
  ## How many seconds between checks to see if the quality can be upgraded. (default: 10s)
  ##
  ## allowQualityUpgrade needs to be enabled for this to work.
  pollQualityUpgradeInterval: '10s'
  ## How many seconds between checks to see if broadcast is live. (default: 5s)
  waitPollInterval: '5s'
  ## Path to a cookies file. Format is a netscape cookies file.
  cookiesFile: ''
  ## Refresh cookies by trying to re-login to FC2. "Keep me logged in" must be
  ## enabled and id.fc2.com cookies must be present.
  cookiesRefreshDuration: '24h'
  ## Remux recordings into mp4/m4a after it is finished. (default: true)
  remux: true
  ## Remux format (default: mp4)
  remuxFormat: 'mp4'
  ## Concatenate and remux with previous recordings after it is finished. (default: false)
  ##
  ## WARNING: We recommend to DISABLE remux since concat also remux.
  ##
  ## Input files must be named <name>.<n>.<ts/mp4/mkv...>. If n=0, n is optional.
  ## Output will be named: "<name>.combined.<remuxFormat>".
  ##
  ## n is only used to determine the order. If there are missing fragments,
  ## the concatenation will still be executed.
  ##
  ## The extensions do not matter. A name.1.ts and a name.2.mp4 will still be concatenated together.
  ## TS files will be prioritized over anything else.
  ##
  ## If remux is enabled, remux will be executed first, then the concatenation
  ## will be executed.
  ##
  ## If extractAudio is true, the m4a will be concatenated separatly.
  ##
  ## TL;DR: This is to concatenate if there is a crash.
  concat: false
  ## Keep the raw .ts recordings after it has been remuxed. (default: false)
  ##
  ## If this option is set to false and concat is true, before every "waiting
  ## for stream to be online", a scan will be executed to detect *.combined.*
  ## files.
  ## The scan will be done on the directory of `scanDirectory`.
  ## If a non-corrupted .combined. file is detected, it will remove .ts older
  ## than `eligibleForCleaningAge`.
  ## After the cleaning, the .combined files will be renamed without the
  ## ".combined" part (if a file already exists due to remux, it won't be renamed).
  keepIntermediates: false
  ## Directory to be scanned for .ts files to be deleted after concatenation. (default: '')
  ##
  ## Scan is recursive.
  ##
  ## Empty value means no scanning.
  scanDirectory: ''
  ## Minimum age of .combined files to be eligible for cleaning. (default: 48h)
  ##
  ## The minimum should be the expected duration of a stream to avoid any race condition.
  eligibleForCleaningAge: '48h'
  ## Delete corrupted .ts recordings. (default: true)
  deleteCorrupted: true
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

## Notify about the state of the watcher.
##
## See: https://containrrr.dev/shoutrrr/latest
notifier:
  enabled: false
  includeTitleInMessage: false
  ## Disable priorities if the transport does not support one.
  noPriority: false
  urls:
    - 'gotify://gotify.example.com/token'

  ## The notification formats can be customized.
  ## Title are automatically prefixed with "fc2-live-dl-go: "
  ## If the message is empty, the message will be the title.
  ## Priorities are following those of android:
  ## Minimum: 0
  ## Low: 1-3
  ## Default: 4-7
  ## High: 8-10
  notificationFormats:
    ## ConfigReloaded is sent when the config is reloaded, i.e. the service restarted.
    configReloaded:
      enabled: true
      # title: "config reloaded"
      # message: <empty>
      # priority: 10

    ## LoginFailed happens when the cookies refresh failed.
    ## Available fields:
    ##   - Error
    loginFailed:
      enabled: true
      # title: "login failed"
      # message: "{{ .Error }}"
      # priority: 10

    ## Panicked is sent when a critical error happens.
    ## When this happens, it is recommended to contact the developer and open an issue.
    ## Available fields:
    ##   - Capture
    panicked:
      enabled: true
      # title: "panicked"
      # message: "{{ .Capture }}"
      # priority: 10

    ## Idle is the initial state.
    ## Available fields:
    ##   - ChannelID
    ##   - Labels
    idle:
      enabled: false
      title: 'watching {{.Labels.EnglishName }}'
      # title: "watching {{ .ChannelID }}"
      # message: <empty>
      # priority: 0

    ## Preparing files happens when the stream is online, but not downloading.
    ## Available fields:
    ##   - ChannelID
    ##   - MetaData
    ##   - Labels
    preparingFiles:
      enabled: false
      title: 'preparing files for {{ .Labels.EnglishName }}'
      # title: 'preparing files for {{ .MetaData.ProfileData.Name }}'
      # message: ''
      # priority: 0

    ## Downloading happens when the stream is online and has emitted a video stream.
    ## Available fields:
    ##   - ChannelID
    ##   - MetaData
    ##   - Labels
    downloading:
      enabled: true
      title: '{{ .Labels.EnglishName }} is streaming'
      # title: "{{ .MetaData.ProfileData.Name }} is streaming"
      # message: "{{ .MetaData.ChannelData.Title }}"
      # priority: 7

    ## Post-processing happens when the stream has finished streaming.
    ## Available fields:
    ##   - ChannelID
    ##   - MetaData
    ##   - Labels
    postProcessing:
      enabled: false
      title: 'post-processing {{ .Labels.EnglishName }}'
      # title: "post-processing {{ .MetaData.ProfileData.Name }}"
      # message: "{{ .MetaData.ChannelData.Title }}"
      # priority: 7

    ## Finished happens when the stream has finished streaming and post-processing is done.
    ## Available fields:
    ##   - ChannelID
    ##   - MetaData
    ##   - Labels
    finished:
      enabled: true
      title: '{{ .Labels.EnglishName }} stream ended'
      # title: "{{ .MetaData.ProfileData.Name }} stream ended"
      # message: "{{ .MetaData.ChannelData.Title }}"
      # priority: 7

    ## Error happens when something bad happens with the downloading of the stream.
    ## Error like this can be user or developper related.
    ## Available fields:
    ##   - ChannelID
    ##   - Error
    ##   - Labels
    error:
      enabled: true
      title: 'stream download of {{ .Labels.EnglishName }} failed'
      # title: 'stream download of {{ .ChannelID }} failed'
      # message: '{{ .Error }}'
      # priority: 10

    ## Canceled happens when a stream download is canceled.
    ## Available fields:
    ##   - ChannelID
    ##   - Labels
    canceled:
      enabled: true
      title: 'stream download of {{ .Labels.EnglishName }} canceled'
      # title: "stream download of {{ .ChannelID }} canceled"
      # message: <empty>
      # priority: 7

    ## UpdateAvailable happens when a new version is available.
    ## Available fields:
    ##   - Version
    updateAvailable:
      enabled: true
      # title: "update available ({{ .Version }})"
      # message: "A new version ({{ .Version }}) of fc2-live-dl is available. Please update."
      # priority: 7
```

</details>

## Details

The [config.yaml](config.yaml) file already includes a documentation for each field. This section will explain some of the fields in more detail.

### About the concatenation and the cleaning routine

First issue: **When a download is interrupted and is reconnected, two files are created. If the stream is interrupted multiple times, the directory will be badly polluted.**

The solution: To avoid having multiple files, the program will concatenate the files into a single file.

The implementation: After each download, the program will check if there are files that can be concatenated using pattern matching. If there are files that can be concatenated, the program will concatenate them.

Example:

```text
- name.ts
- name.1.ts

After concatenation:

- name.ts
- name.1.ts
- name.combined.mp4
```

The concatenation is done by concatenating packets, which automatically includes a remuxing step (i.e. the container is changing, but the packets aren't touched). The concatenation is controlled using these parameters (with their recommended values):

```yaml
remux: false # No need to remux since the concatenation will do it.
remuxFormat: mp4 # The format of the remuxed file.
concat: true
extractAudio: false # Your preference.
deleteCorrupted: true # Recommended as corrupted files will also be skipped anyway.
```

Second issue: **If the concatenation is done, the raw files are not deleted.** This is because deleting the files too early can lead to missing parts in the combined file. There is also the issue of a race condition: concatenating while downloading is an undefined behavior.

The solution: To avoid having too many files, the program will clean the files after a certain amount of time.

The implementation: Scans will be executed frequently: Before every "waiting for stream to be online" and periodically. If the .combined file is old enough, the program will delete the raw files and rename the .combined file into the final file.

Example:

```text
- name.1.ts
- name.2.ts
- name.combined.mp4 (older than 48h)

After cleaning:

- name.mp4
```

The cleaning is controlled using these parameters (with their recommended values):

```yaml
concat: true
keepIntermediates: false # Clean the raw files after concatenation.
scanDirectory: '/path/to/directory' # Will scan recursively, so beware.
eligibleForCleaningAge: 48h
```

### About quality upgrade

The issue: **Streams can be downloaded at higher quality only after a certain amount of time.** More precisely, FC2 only exposes the 3Mbps quality after a certain amount of time. It can be 5 minutes, 10 minutes, 30 minutes, 1 hour, etc. `waitForQualityMaxTries` can lead to missing the beginning of the stream.

The solution: **Check periodically if the quality can be upgraded, and download it when it is possible.**

The implementation: The program will check periodically if the quality can be upgraded. If the quality can be upgraded, the program will switch to the new quality. **A cut off will be present in the recording, but in general, it will simply repeat a section (1 second of video/audio) and not miss anything.**

More precisely, when the quality can be upgraded, the program will stop the current download and download the new quality (as seamlessly as possible, with no "downtime"). Packets are appended to the same file, which will cause a discontinuity in the file. The program will try to fix the discontinuity by remuxing the file. **Remux or Concat must be enabled to fix the discontinuity.**

The quality upgrade is controlled using these parameters (with their recommended values):

```yaml
allowQualityUpgrade: true
pollQualityUpgradeInterval: 10s
waitForQualityMaxTries: 10 # Lower value to avoid missing the beginning of the stream. Prefer quality upgrade.

remux: false
remuxFormat: mp4
concat: true
extractAudio: false # Your preference.
deleteCorrupted: true

concat: true # Must be enabled to fix the discontinuity.
keepIntermediates: false # Clean the raw files after concatenation.
scanDirectory: '/path/to/directory' # Will scan recursively, so beware.
eligibleForCleaningAge: 48h
```

### About cookies refresh

This used to try to keep alive the FC2 login session. The program will try to re-login to FC2 every 24 hours. We cannot login directly to FC2 because of a captcha, so we must use the cookies to re-login.

From your browser, you must extract the cookies from the FC2 website. Login to FC2 with the "Keep me logged in" option enabled and extract the cookies.

Cookies can be extracted using the Chrome extension [Get cookies.txt LOCALLY](https://chrome.google.com/webstore/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc) or the Firefox extension [cookies.txt](https://addons.mozilla.org/en-US/firefox/addon/cookies-txt/). You must extract **all cookies** and filter them so that they only contain FC2-related cookies. `id.fc2.com` and `secure.id.fc2.com` are the most important ones.

> [!CAUTION]
> Cookies are sensitive data. Do not share them with anyone. They can be used to impersonate you.

It should look like that:

```shell
# Domain	IncludeSubdomains	Path	IsSecure	ExpiresUnix	Name	Value
.id.fc2.com	TRUE	/	TRUE	0	FCSID	<value>
id.fc2.com	FALSE	/	FALSE	0	AWSELB	<value>
id.fc2.com	FALSE	/	TRUE	0	AWSELBCORS	<value>
secure.id.fc2.com	FALSE	/	FALSE	0	AWSELB	<value>
secure.id.fc2.com	FALSE	/	TRUE	0	AWSELBCORS	<value>
.id.fc2.com	TRUE	/	TRUE	1702402071	login_status	<value>
.id.fc2.com	TRUE	/	TRUE	0	secure_check_fc2	<value>
.fc2.com	TRUE	/	FALSE	1699810057	language	<value>
.fc2.com	TRUE	/	FALSE	1731259657	fclo	<value>
.fc2.com	TRUE	/	FALSE	0	fcu	<value>
.fc2.com	TRUE	/	TRUE	0	fcus	<value>
.fc2.com	TRUE	/	FALSE	1715794071	FC2_GDPR	<value>
.fc2.com	TRUE	/	FALSE	1702402071	glgd_val	<value>
.fc2.com	TRUE	/	TRUE	1702315671	__fc2id_rct	<value>
.live.fc2.com	TRUE	/	FALSE	0	lang	<value>
.live.fc2.com	TRUE	/	FALSE	0	PHPSESSID	<value>
live.fc2.com	FALSE	/	FALSE	1705080472	ab_test_logined_flg	<value>
```

Don't worry if it doesn't look exactly like that. The most important cookies are from `id.fc2.com` and `secure.id.fc2.com`.

### About metrics, traces and continuous profiling

#### Metrics and Traces

##### Prometheus (Pull-based, metrics only)

The program exposes metrics on the `/metrics` endpoint. The metrics are in Prometheus format.

#### OTLP (Push-based)

The program can push metrics to an OTLP receiver. The OTLP client is configurable using standard environment variables.

- [OpenTelemetry Protocol Exporter](https://opentelemetry.io/docs/specs/otel/protocol/exporter/)

Example with Grafana Alloy:

```shell
OTEL_EXPORTER_OTLP_TRACES_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT="https://alloy.example.com:4317"
# (Recommended) CA Verification
OTEL_EXPORTER_OTLP_CERTIFICATE="/certs/ca.crt" # Or /etc/ssl/certs/ca-certificates.crt
# (Optional) mTLS
OTEL_EXPORTER_OTLP_CLIENT_KEY="/certs/tls.key"
OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE="/certs/tls.crt"
```

With the typical Grafana Alloy configuration:

```hcl
otelcol.receiver.otlp "otlp_receiver" {
  grpc {
    tls {
      ca_file = "/etc/alloy/certs/ca.crt"
      cert_file = "/etc/alloy/certs/tls.crt"
      key_file = "/etc/alloy/certs/tls.key"
      # (optional) mTLS
      client_ca_file = "/etc/alloy/certs/ca.crt"
    }
  }
  http {
    tls {
      ca_file = "/etc/alloy/certs/ca.crt"
      cert_file = "/etc/alloy/certs/tls.crt"
      key_file = "/etc/alloy/certs/tls.key"
      # (optional) mTLS
      client_ca_file = "/etc/alloy/certs/ca.crt"
    }
  }

  output {
    metrics = [otelcol.processor.resourcedetection.default.input]
    logs    = [otelcol.processor.resourcedetection.default.input]
    traces  = [otelcol.processor.resourcedetection.default.input]
  }
}

# [...]
# See https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/setup/collector/grafana-alloy/

# Feel free to export to Grafana Tempo, Mimir, Prometheus, etc.
# See: https://grafana.com/docs/alloy/latest/reference/components/
```

If you are using mTLS, make sure to have the usages of the certificates correctly set up:

- `client auth` for the client certificate.
- `server auth` for the server certificate.

#### Continuous Profiling (pull-based)

The program can be profiled using the `pprof` package. The pprof package is enabled by default and can be accessed at `http://<host>:3000/debug/pprof/`.

In addition to that, `godeltaprof` has been added for Pyroscope.

You can continuously profile the program using Grafana Alloy with the following configuration:

```hcl
pyroscope.write "write_grafana_pyroscope" {
  endpoint {
    url = env("GRAFANA_CLOUD_PYROSCOPE_ENDPOINT")

    basic_auth {
      username = env("GRAFANA_CLOUD_INSTANCE_ID")
      password = env("GRAFANA_CLOUD_API_KEY")
    }
  }
}


pyroscope.scrape "scrape_fc2_pprof" {
  targets    = [{"__address__" = "<host>:3000", "service_name" = "fc2"}]
  forward_to = [pyroscope.write.write_grafana_pyroscope.receiver]

  profiling_config {
    profile.process_cpu {
      enabled = true
    }

    profile.godeltaprof_memory {
      enabled = true
    }

    profile.memory { // disable memory, use godeltaprof_memory instead
      enabled = false
    }

    profile.godeltaprof_mutex {
      enabled = true
    }

    profile.mutex { // disable mutex, use godeltaprof_mutex instead
      enabled = false
    }

    profile.godeltaprof_block {
      enabled = true
    }

    profile.block { // disable block, use godeltaprof_block instead
      enabled = false
    }

    profile.goroutine {
      enabled = true
    }
  }
}
```

See [Grafana documentation - Set up Go profiling in pull mode](https://grafana.com/docs/pyroscope/latest/configure-client/grafana-agent/go_pull/) for more information.

### About proxies

Since we are using `net/http` and `nhooyr.io/websocket`, proxies are supported by passing `HTTP_PROXY` and `HTTPS_PROXY` as environment variables. The format should be either a complete URL or a "host[:port]", in which case the "HTTP" scheme is assumed.

## License

This project is under [MIT License](LICENSE).

## Credits

Many thanks to [hizkifw](https://github.com/hizkifw) and contributors to the [HoloArchivists/fc2-live-dl](https://github.com/HoloArchivists/fc2-live-dl) project for their excellent source code.

The executable links to libavformat, libavutil and libavcodec, which are licensed under the Lesser GPL v2.1 (LGPLv2.1). The source code for the libavformat, libavutil and libavcodec libraries is available on the [FFmpeg website](https://www.ffmpeg.org/).
