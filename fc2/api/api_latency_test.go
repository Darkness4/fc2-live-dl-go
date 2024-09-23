//go:build unit

package api_test

import (
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type LatencyYamlTest struct {
	Test api.Latency `yaml:"test"`
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
			expected: LatencyYamlTest{api.LatencyHigh},
			title:    "Positive test",
		},
		{
			input:   []byte("test: \"unknown unknown\""),
			title:   "Negative test",
			isError: api.ErrUnknownLatency,
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
		expected api.Latency
		title    string
	}{
		{
			input:    92,
			expected: api.LatencyMid,
			title:    "LatencyMid",
		},
		{
			input:    107,
			title:    "LatencyUnknown",
			expected: api.LatencyUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			// Act
			actual := api.LatencyFromMode(tt.input)

			// Assert
			require.Equal(t, tt.expected, actual)
		})
	}
}
