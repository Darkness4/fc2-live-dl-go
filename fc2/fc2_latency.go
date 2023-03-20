package fc2

import "errors"

type Latency int

const (
	LatencyUnknown Latency = 0
	LatencyLow     Latency = 1
	LatencyHigh    Latency = 2
	LatencyMid     Latency = 3
)

var ErrUnknownLatency = errors.New("unknown latency")

func LatencyParseString(value string) Latency {
	switch value {
	case "low":
		return LatencyLow
	case "high":
		return LatencyHigh
	case "mid":
		return LatencyMid
	default:
		return LatencyUnknown
	}
}

func (l *Latency) UnmarshalText(text []byte) error {
	*l = LatencyParseString(string(text))
	if *l == LatencyUnknown {
		return ErrUnknownLatency
	}
	return nil
}

func (l Latency) String() string {
	switch l {
	case LatencyLow:
		return "low"
	case LatencyHigh:
		return "high"
	case LatencyMid:
		return "mid"
	default:
		return "unknown"
	}
}

func LatencyFromMode(mode int) Latency {
	latency := mode%10 + 1
	switch {
	case latency <= int(LatencyUnknown) || latency > int(LatencyMid):
		return LatencyUnknown
	default:
		return Latency(latency)
	}
}
