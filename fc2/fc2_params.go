package fc2

import (
	"time"
)

type Params struct {
	Quality                Quality           `yaml:"quality"`
	Latency                Latency           `yaml:"latency"`
	PacketLossMax          int               `yaml:"packetLossMax"`
	OutFormat              string            `yaml:"outFormat"`
	WriteChat              bool              `yaml:"writeChat"`
	WriteInfoJSON          bool              `yaml:"writeInfoJson"`
	WriteThumbnail         bool              `yaml:"writeThumbnail"`
	WaitForLive            bool              `yaml:"waitForLive"`
	WaitForQualityMaxTries int               `yaml:"waitForQualityMaxTries"`
	WaitPollInterval       time.Duration     `yaml:"waitPollInterval"`
	CookiesFile            string            `yaml:"cookiesFile"`
	Remux                  bool              `yaml:"remux"`
	KeepIntermediates      bool              `yaml:"keepIntermediates"`
	ExtractAudio           bool              `yaml:"extractAudio"`
	Labels                 map[string]string `yaml:"labels"`
}

type OptionalParams struct {
	Quality                *Quality          `yaml:"quality"`
	Latency                *Latency          `yaml:"latency"`
	PacketLossMax          *int              `yaml:"packetLossMax"`
	OutFormat              *string           `yaml:"outFormat"`
	WriteChat              *bool             `yaml:"writeChat"`
	WriteInfoJSON          *bool             `yaml:"writeInfoJson"`
	WriteThumbnail         *bool             `yaml:"writeThumbnail"`
	WaitForLive            *bool             `yaml:"waitForLive"`
	WaitForQualityMaxTries *int              `yaml:"waitForQualityMaxTries"`
	WaitPollInterval       *time.Duration    `yaml:"waitPollInterval"`
	CookiesFile            *string           `yaml:"cookiesFile"`
	Remux                  *bool             `yaml:"remux"`
	KeepIntermediates      *bool             `yaml:"keepIntermediates"`
	ExtractAudio           *bool             `yaml:"extractAudio"`
	Labels                 map[string]string `yaml:"labels"`
}

var DefaultParams Params = Params{
	Quality:                Quality1_2MBps,
	Latency:                LatencyMid,
	PacketLossMax:          200,
	OutFormat:              "{{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}",
	WriteChat:              false,
	WriteInfoJSON:          false,
	WriteThumbnail:         false,
	WaitForLive:            true,
	WaitForQualityMaxTries: 10,
	WaitPollInterval:       5 * time.Second,
	CookiesFile:            "",
	Remux:                  true,
	KeepIntermediates:      false,
	ExtractAudio:           false,
	Labels:                 nil,
}

func (override *OptionalParams) Override(params *Params) {
	if override.Quality != nil {
		params.Quality = *override.Quality
	}
	if override.Latency != nil {
		params.Latency = *override.Latency
	}
	if override.PacketLossMax != nil {
		params.PacketLossMax = *override.PacketLossMax
	}
	if override.OutFormat != nil {
		params.OutFormat = *override.OutFormat
	}
	if override.WriteChat != nil {
		params.WriteChat = *override.WriteChat
	}
	if override.WriteInfoJSON != nil {
		params.WriteInfoJSON = *override.WriteInfoJSON
	}
	if override.WriteThumbnail != nil {
		params.WriteThumbnail = *override.WriteThumbnail
	}
	if override.WaitForLive != nil {
		params.WaitForLive = *override.WaitForLive
	}
	if override.CookiesFile != nil {
		params.CookiesFile = *override.CookiesFile
	}
	if override.WaitForQualityMaxTries != nil {
		params.WaitForQualityMaxTries = *override.WaitForQualityMaxTries
	}
	if override.WaitPollInterval != nil {
		params.WaitPollInterval = *override.WaitPollInterval
	}
	if override.Remux != nil {
		params.Remux = *override.Remux
	}
	if override.KeepIntermediates != nil {
		params.KeepIntermediates = *override.KeepIntermediates
	}
	if override.ExtractAudio != nil {
		params.ExtractAudio = *override.ExtractAudio
	}
	if override.Labels != nil {
		if params.Labels == nil {
			params.Labels = make(map[string]string)
		}
		for k, v := range override.Labels {
			params.Labels[k] = v
		}
	}
}

func (p *Params) Clone() *Params {
	// Create a new Params struct with the same field values as the original
	clone := Params{
		Quality:                p.Quality,
		Latency:                p.Latency,
		PacketLossMax:          p.PacketLossMax,
		OutFormat:              p.OutFormat,
		WriteChat:              p.WriteChat,
		WriteInfoJSON:          p.WriteInfoJSON,
		WriteThumbnail:         p.WriteThumbnail,
		WaitForLive:            p.WaitForLive,
		WaitForQualityMaxTries: p.WaitForQualityMaxTries,
		WaitPollInterval:       p.WaitPollInterval,
		CookiesFile:            p.CookiesFile,
		Remux:                  p.Remux,
		KeepIntermediates:      p.KeepIntermediates,
		ExtractAudio:           p.ExtractAudio,
	}

	// Clone the labels map if it exists
	if p.Labels != nil {
		clone.Labels = make(map[string]string)
		for k, v := range p.Labels {
			clone.Labels[k] = v
		}
	}

	return &clone
}
