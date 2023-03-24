//go:build unit

package fc2_test

import (
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type QualityYamlTest struct {
	Test fc2.Quality `yaml:"test"`
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
			expected: QualityYamlTest{fc2.Quality150KBps},
			title:    "Positive test",
		},
		{
			input:   []byte("test: \"unknown unknown\""),
			title:   "Negative test",
			isError: fc2.ErrUnknownQuality,
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
		expected fc2.Quality
		title    string
	}{
		{
			input:    92,
			expected: fc2.QualitySound,
			title:    "QualitySound",
		},
		{
			input:    101,
			title:    "QualityUnknown",
			expected: fc2.QualityUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			// Act
			actual := fc2.QualityFromMode(tt.input)

			// Assert
			require.Equal(t, tt.expected, actual)
		})
	}
}
