//go:build unit

package fc2_test

import (
	"testing"

	"github.com/Darkness4/fc2-live-dl-lite/fc2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type LatencyYamlTest struct {
	Test fc2.Latency `yaml:"test"`
}

func TestLatencyUnmarshalText(t *testing.T) {
	tests := []struct {
		input    []byte
		isError  error
		expected LatencyYamlTest
		title    string
	}{
		{
			input:    []byte("test: \"high\""),
			expected: LatencyYamlTest{fc2.LatencyHigh},
			title:    "Positive test",
		},
		{
			input:   []byte("test: \"unknown unknown\""),
			title:   "Negative test",
			isError: fc2.ErrUnknownLatency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			// Act
			actual := LatencyYamlTest{}
			err := yaml.Unmarshal(tt.input, &actual)

			// Assert
			if tt.isError != nil {
				assert.Error(t, err)
				require.Equal(t, tt.isError, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, actual)
			}
		})
	}
}

func TestLatencyFromMode(t *testing.T) {
	tests := []struct {
		input    int
		expected fc2.Latency
		title    string
	}{
		{
			input:    92,
			expected: fc2.LatencyMid,
			title:    "LatencyMid",
		},
		{
			input:    107,
			title:    "LatencyUnknown",
			expected: fc2.LatencyUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			// Act
			actual := fc2.LatencyFromMode(tt.input)

			// Assert
			require.Equal(t, tt.expected, actual)
		})
	}
}
