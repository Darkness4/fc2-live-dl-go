package fc2

import "errors"

// Latency represents the latency of the live stream.
type Latency int

const (
	// LatencyUnknown represents an unknown latency.
	LatencyUnknown Latency = 0
	// LatencyLow represents a low latency.
	LatencyLow Latency = 1
	// LatencyHigh represents a high latency.
	LatencyHigh Latency = 2
	// LatencyMid represents a mid latency.
	LatencyMid Latency = 3
)

// ErrUnknownLatency is returned when the latency is unknown.
var ErrUnknownLatency = errors.New("unknown latency")

// LatencyParseString parses a string into a Latency.
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

// UnmarshalText unmarshals a string into a Latency.
func (l *Latency) UnmarshalText(text []byte) error {
	*l = LatencyParseString(string(text))
	if *l == LatencyUnknown {
		return ErrUnknownLatency
	}
	return nil
}

// String returns the string representation of a Latency.
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

// LatencyFromMode returns a Latency from a mode.
func LatencyFromMode(mode int) Latency {
	latency := mode%10 + 1
	switch {
	case latency <= int(LatencyUnknown) || latency > int(LatencyMid):
		return LatencyUnknown
	default:
		return Latency(latency)
	}
}
