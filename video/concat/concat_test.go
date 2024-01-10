package concat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterFiles(t *testing.T) {
	tests := []struct {
		names    []string
		base     string
		path     string
		options  []Option
		isError  error
		expected []string
		title    string
	}{
		{
			names: []string{
				"name.mp4",
				"name.1.ts",
				"name.1.mp4",
				"name.2.mp4",
				"trash.ts",
			},
			base: "name",
			path: ".",
			options: []Option{
				IgnoreExtension(),
			},
			expected: []string{
				"name.mp4",
				"name.1.ts",
				"name.2.mp4",
			},
			title: "Positive test",
		},
		{
			names: []string{
				"2024-01-10 _.1.m4a",
				"2024-01-10 _.1.mp4",
				"2024-01-10 _.combined.mp4",
				"2024-01-10 _.m4a",
				"2024-01-10 _.mp4",
			},
			base: "2024-01-10 _",
			path: ".",
			options: []Option{
				IgnoreExtension(),
			},
			expected: []string{
				"2024-01-10 _.mp4",
				"2024-01-10 _.1.mp4",
			},
			title: "Positive test 2",
		},
		{
			names: []string{
				"2024-01-10 _.1.m4a",
				"2024-01-10 _.1.mp4",
				"2024-01-10 _.combined.mp4",
				"2024-01-10 _.m4a",
				"2024-01-10 _.mp4",
			},
			base: "2024-01-10 _",
			path: ".",
			options: []Option{
				IgnoreExtension(),
				WithAudioOnly(),
			},
			expected: []string{
				"2024-01-10 _.mp4",
				"2024-01-10 _.1.mp4",
			},
			title: "Positive test 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			// Act
			actual, err := filterFiles(tt.names, tt.base, tt.path, applyOptions(tt.options))

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
