package fc2

import "errors"

type Quality int

const (
	QualityUnknown Quality = 0
	Quality150KBps Quality = 10
	Quality400KBps Quality = 20
	Quality1_2MBps Quality = 30
	Quality2MBps   Quality = 40
	Quality3MBps   Quality = 50
	QualitySound   Quality = 90
)

var ErrUnknownQuality = errors.New("unknown quality")

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

func (q *Quality) UnmarshalText(text []byte) error {
	*q = QualityParseString(string(text))
	if *q == QualityUnknown {
		return ErrUnknownQuality
	}
	return nil
}

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

func QualityFromMode(mode int) Quality {
	quality := (mode / 10) * 10
	switch {
	case quality <= int(Quality150KBps) || quality > int(QualitySound):
		return QualityUnknown
	default:
		return Quality(quality)
	}
}
