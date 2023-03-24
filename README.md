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

Similarities:

- Business logic. It follow a similar order with a similar configuration. This means that updates can be passed from one project to the other.

## Installation

```shell
go install github.com/Darkness4/fc2-live-dl-go@latest
```

## Usage

### Download a single live fc2 stream

```shell
fc2-live-dl-go download [command options] channelID
```

```shell
OPTIONS:
   --quality value  Quality of the stream to download.
      Available latency options: 150Kbps, 400Kbps, 1.2Mbps, 2Mbps, 3Mbps, sound. (default: "3Mbps")
   --latency value  Stream latency. Select a higher latency if experiencing stability issues.
      Available latency options: low, high, mid. (default: "mid")
   --format value  Golang templating format. Available fields: ChannelID, ChannelName, Date, Time, Title, Ext, Labels[key].
      Available format options:
        ChannelID: ID of the broadcast
        ChannelName: broadcaster's profile name
        Date: local date YYYY-MM-DD
        Time: local time HHMMSS
        Ext: file extension
        Title: title of the live broadcast
        Labels[key]: custom labels
       (default: "{{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}")
   --no-remux                          Do not remux recordings into mp4/m4a after it is finished. (default: false)
   --keep-intermediates, -k            Keep the raw .ts recordings after it has been remuxed. (default: false)
   --extract-audio, -x                 Generate an audio-only copy of the stream. (default: false)
   --cookies value                     Path to a cookies file.
   --write-chat                        Save live chat into a json file. (default: false)
   --write-info-json                   Dump output stream information into a json file. (default: false)
   --write-thumbnail                   Download thumbnail into a file. (default: false)
   --wait                              Wait until the broadcast goes live, then start recording. (default: false)
   --wait-for-quality-max-tries value  If the requested quality is not available, keep retrying before falling back to the next best quality. (default: 10)
   --poll-interval value               How many seconds between checks to see if broadcast is live. (default: 5s)
   --max-tries value                   On failure, keep retrying. (cancellation and end of stream will be ignored) (default: 10)
   --help, -h                          show help
```

### Download a multiple live fc2 stream

## Credits

Many thanks to https://github.com/hizkifw and contributors to the [HoloArchivists/fc2-live-dl](https://github.com/HoloArchivists/fc2-live-dl) project for their excellent source code.
