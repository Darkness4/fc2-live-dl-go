package fc2

import (
	"encoding/json"
	"time"
)

// Params represents the parameters for the download.
type Params struct {
	Quality                    Quality           `yaml:"quality,omitempty"`
	Latency                    Latency           `yaml:"latency,omitempty"`
	PacketLossMax              int               `yaml:"packetLossMax,omitempty"`
	OutFormat                  string            `yaml:"outFormat,omitempty"`
	WriteChat                  bool              `yaml:"writeChat,omitempty"`
	WriteInfoJSON              bool              `yaml:"writeInfoJson,omitempty"`
	WriteThumbnail             bool              `yaml:"writeThumbnail,omitempty"`
	WaitForLive                bool              `yaml:"waitForLive,omitempty"`
	WaitForQualityMaxTries     int               `yaml:"waitForQualityMaxTries,omitempty"`
	AllowQualityUpgrade        bool              `yaml:"allowQualityUpgrade,omitempty"`
	PollQualityUpgradeInterval time.Duration     `yaml:"pollQualityUpgradeInterval,omitempty"`
	WaitPollInterval           time.Duration     `yaml:"waitPollInterval,omitempty"`
	CookiesFile                string            `yaml:"cookiesFile,omitempty"`
	CookiesRefreshDuration     time.Duration     `yaml:"cookiesRefreshDuration,omitempty"`
	Remux                      bool              `yaml:"remux,omitempty"`
	RemuxFormat                string            `yaml:"remuxFormat,omitempty"`
	Concat                     bool              `yaml:"concat,omitempty"`
	KeepIntermediates          bool              `yaml:"keepIntermediates,omitempty"`
	ScanDirectory              string            `yaml:"scanDirectory,omitempty"`
	EligibleForCleaningAge     time.Duration     `yaml:"eligibleForCleaningAge,omitempty"`
	DeleteCorrupted            bool              `yaml:"deleteCorrupted,omitempty"`
	ExtractAudio               bool              `yaml:"extractAudio,omitempty"`
	Labels                     map[string]string `yaml:"labels,omitempty"`
}

func (p *Params) String() string {
	out, _ := json.MarshalIndent(p, "", "  ")
	return string(out)
}

// OptionalParams represents the optional parameters for the download.
type OptionalParams struct {
	Quality                    *Quality          `yaml:"quality,omitempty"`
	Latency                    *Latency          `yaml:"latency,omitempty"`
	PacketLossMax              *int              `yaml:"packetLossMax,omitempty"`
	OutFormat                  *string           `yaml:"outFormat,omitempty"`
	WriteChat                  *bool             `yaml:"writeChat,omitempty"`
	WriteInfoJSON              *bool             `yaml:"writeInfoJson,omitempty"`
	WriteThumbnail             *bool             `yaml:"writeThumbnail,omitempty"`
	WaitForLive                *bool             `yaml:"waitForLive,omitempty"`
	WaitForQualityMaxTries     *int              `yaml:"waitForQualityMaxTries,omitempty"`
	AllowQualityUpgrade        *bool             `yaml:"allowQualityUpgrade,omitempty"`
	PollQualityUpgradeInterval *time.Duration    `yaml:"pollQualityUpgradeInterval,omitempty"`
	WaitPollInterval           *time.Duration    `yaml:"waitPollInterval,omitempty"`
	CookiesFile                *string           `yaml:"cookiesFile,omitempty"`
	CookiesRefreshDuration     *time.Duration    `yaml:"cookiesRefreshDuration,omitempty"`
	Remux                      *bool             `yaml:"remux,omitempty"`
	RemuxFormat                *string           `yaml:"remuxFormat,omitempty"`
	Concat                     *bool             `yaml:"concat,omitempty"`
	KeepIntermediates          *bool             `yaml:"keepIntermediates,omitempty"`
	ScanDirectory              *string           `yaml:"scanDirectory,omitempty"`
	EligibleForCleaningAge     *time.Duration    `yaml:"eligibleForCleaningAge,omitempty"`
	DeleteCorrupted            *bool             `yaml:"deleteCorrupted,omitempty"`
	ExtractAudio               *bool             `yaml:"extractAudio,omitempty"`
	Labels                     map[string]string `yaml:"labels,omitempty"`
}

// DefaultParams is the default set of parameters.
var DefaultParams = Params{
	Quality:                    Quality3MBps,
	Latency:                    LatencyMid,
	PacketLossMax:              20,
	OutFormat:                  "{{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}",
	WriteChat:                  false,
	WriteInfoJSON:              false,
	WriteThumbnail:             false,
	WaitForLive:                true,
	WaitForQualityMaxTries:     60,
	AllowQualityUpgrade:        false,
	PollQualityUpgradeInterval: 10 * time.Second,
	WaitPollInterval:           5 * time.Second,
	CookiesFile:                "",
	CookiesRefreshDuration:     24 * time.Hour,
	Remux:                      true,
	RemuxFormat:                "mp4",
	Concat:                     true,
	KeepIntermediates:          false,
	ScanDirectory:              "",
	EligibleForCleaningAge:     48 * time.Hour,
	DeleteCorrupted:            true,
	ExtractAudio:               false,
	Labels:                     nil,
}

// Override applies the values from the OptionalParams to the Params.
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
	if override.CookiesRefreshDuration != nil {
		params.CookiesRefreshDuration = *override.CookiesRefreshDuration
	}
	if override.WaitForQualityMaxTries != nil {
		params.WaitForQualityMaxTries = *override.WaitForQualityMaxTries
	}
	if override.AllowQualityUpgrade != nil {
		params.AllowQualityUpgrade = *override.AllowQualityUpgrade
	}
	if override.PollQualityUpgradeInterval != nil {
		params.PollQualityUpgradeInterval = *override.PollQualityUpgradeInterval
	}
	if override.WaitPollInterval != nil {
		params.WaitPollInterval = *override.WaitPollInterval
	}
	if override.Remux != nil {
		params.Remux = *override.Remux
	}
	if override.RemuxFormat != nil {
		params.RemuxFormat = *override.RemuxFormat
	}
	if override.Concat != nil {
		params.Concat = *override.Concat
	}
	if override.KeepIntermediates != nil {
		params.KeepIntermediates = *override.KeepIntermediates
	}
	if override.ScanDirectory != nil {
		params.ScanDirectory = *override.ScanDirectory
	}
	if override.EligibleForCleaningAge != nil {
		params.EligibleForCleaningAge = *override.EligibleForCleaningAge
	}
	if override.DeleteCorrupted != nil {
		params.DeleteCorrupted = *override.DeleteCorrupted
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

// Clone creates a deep copy of the Params struct.
func (p *Params) Clone() *Params {
	// Create a new Params struct with the same field values as the original
	clone := Params{
		Quality:                    p.Quality,
		Latency:                    p.Latency,
		PacketLossMax:              p.PacketLossMax,
		OutFormat:                  p.OutFormat,
		WriteChat:                  p.WriteChat,
		WriteInfoJSON:              p.WriteInfoJSON,
		WriteThumbnail:             p.WriteThumbnail,
		WaitForLive:                p.WaitForLive,
		WaitForQualityMaxTries:     p.WaitForQualityMaxTries,
		AllowQualityUpgrade:        p.AllowQualityUpgrade,
		PollQualityUpgradeInterval: p.PollQualityUpgradeInterval,
		WaitPollInterval:           p.WaitPollInterval,
		CookiesFile:                p.CookiesFile,
		CookiesRefreshDuration:     p.CookiesRefreshDuration,
		Remux:                      p.Remux,
		RemuxFormat:                p.RemuxFormat,
		Concat:                     p.Concat,
		KeepIntermediates:          p.KeepIntermediates,
		ScanDirectory:              p.ScanDirectory,
		EligibleForCleaningAge:     p.EligibleForCleaningAge,
		DeleteCorrupted:            p.DeleteCorrupted,
		ExtractAudio:               p.ExtractAudio,
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
