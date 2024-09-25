//go:build unit

package api_test

import (
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type QualityYamlTest struct {
	Test api.Quality `yaml:"test"`
}

func TestQualityUnmarshalText(t *testing.T) {
	tests := []struct {
		input    []byte
		isError  error
		expected QualityYamlTest
		title    string
	}{
		{
			input:    []byte("test: \"150Kbps\""),
			expected: QualityYamlTest{api.Quality150KBps},
			title:    "Positive test",
		},
		{
			input:   []byte("test: \"unknown unknown\""),
			title:   "Negative test",
			isError: api.ErrUnknownQuality,
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			// Act
			actual := QualityYamlTest{}
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

func TestQualityFromMode(t *testing.T) {
	tests := []struct {
		input    int
		expected api.Quality
		title    string
	}{
		{
			input:    92,
			expected: api.QualitySound,
			title:    "QualitySound",
		},
		{
			input:    52,
			expected: api.Quality3MBps,
			title:    "Quality3MBps",
		},
		{
			input:    101,
			title:    "QualityUnknown",
			expected: api.QualityUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			// Act
			actual := api.QualityFromMode(tt.input)

			// Assert
			require.Equal(t, tt.expected, actual)
		})
	}
}
