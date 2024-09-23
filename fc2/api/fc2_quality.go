package api

import "errors"

// Quality represents the quality of the live stream.
type Quality int

const (
	// QualityUnknown represents an unknown quality.
	QualityUnknown Quality = 0
	// Quality150KBps represents a 150Kbps bitrate.
	Quality150KBps Quality = 10
	// Quality400KBps represents a 400Kbps bitrate.
	Quality400KBps Quality = 20
	// Quality1_2MBps represents a 1.2Mbps bitrate.
	Quality1_2MBps Quality = 30
	// Quality2MBps represents a 2Mbps bitrate.
	Quality2MBps Quality = 40
	// Quality3MBps represents a 3Mbps bitrate.
	Quality3MBps Quality = 50
	// QualitySound represents a sound only stream.
	QualitySound Quality = 90
)

// ErrUnknownQuality is returned when the quality is unknown.
var ErrUnknownQuality = errors.New("unknown quality")

// QualityParseString parses a string into a Quality.
func QualityParseString(value string) Quality {
	switch value {
	case "150Kbps":
		return Quality150KBps
	case "400Kbps":
		return Quality400KBps
	case "1.2Mbps":
		return Quality1_2MBps
	case "2Mbps":
		return Quality2MBps
	case "3Mbps":
		return Quality3MBps
	case "sound":
		return QualitySound
	default:
		return QualityUnknown
	}
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (q *Quality) UnmarshalText(text []byte) error {
	*q = QualityParseString(string(text))
	if *q == QualityUnknown {
		return ErrUnknownQuality
	}
	return nil
}

// String implements the fmt.Stringer interface.
func (q Quality) String() string {
	switch q {
	case Quality150KBps:
		return "150Kbps"
	case Quality400KBps:
		return "400Kbps"
	case Quality1_2MBps:
		return "1.2Mbps"
	case Quality2MBps:
		return "2Mbps"
	case Quality3MBps:
		return "3Mbps"
	case QualitySound:
		return "sound"
	default:
		return "unknown"
	}
}

// QualityFromMode returns a Quality from a live stream mode.
func QualityFromMode(mode int) Quality {
	quality := (mode / 10) * 10
	switch {
	case quality <= int(Quality150KBps) || quality > int(QualitySound):
		return QualityUnknown
	default:
		return Quality(quality)
	}
}
