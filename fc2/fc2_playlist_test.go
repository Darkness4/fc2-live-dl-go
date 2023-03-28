//go:build unit

package fc2_test

import (
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/stretchr/testify/require"
)

func TestExtractAndMergePlaylists(t *testing.T) {
	tests := []struct {
		input    *fc2.HLSInformation
		expected []fc2.Playlist
		title    string
	}{
		{
			input: &fc2.HLSInformation{
				Playlists: []fc2.Playlist{
					{
						URL:  "a",
						Mode: 50,
					},
				},
				PlaylistsHighLatency: []fc2.Playlist{
					{
						URL:  "b",
						Mode: 51,
					},
				},
				PlaylistsMiddleLatency: []fc2.Playlist{
					{
						URL:  "c",
						Mode: 52,
					},
				},
			},
			expected: []fc2.Playlist{
				{
					URL:  "a",
					Mode: 50,
				},
				{
					URL:  "b",
					Mode: 51,
				},
				{
					URL:  "c",
					Mode: 52,
				},
			},
			title: "Positive test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			// Act
			actual := fc2.ExtractAndMergePlaylists(tt.input)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestSortPlaylists(t *testing.T) {
	tests := []struct {
		input    []fc2.Playlist
		expected []fc2.Playlist
		title    string
	}{
		{
			input: []fc2.Playlist{
				{
					URL:  "a",
					Mode: 92,
				},
				{
					URL:  "c",
					Mode: 32,
				},
				{
					URL:  "b",
					Mode: 52,
				},
				{
					URL:  "a",
					Mode: 91,
				},
			},
			expected: []fc2.Playlist{
				{
					URL:  "b",
					Mode: 52,
				},
				{
					URL:  "c",
					Mode: 32,
				},
				{
					URL:  "a",
					Mode: 92,
				},
				{
					URL:  "a",
					Mode: 91,
				},
			},
			title: "Positive test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			// Act
			actual := fc2.SortPlaylists(tt.input)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetPlaylistOrBest(t *testing.T) {
	sortedPlaylists := []fc2.Playlist{
		{
			URL:  "b",
			Mode: 52,
		},
		{
			URL:  "c",
			Mode: 32,
		},
		{
			URL:  "a",
			Mode: 92,
		},
		{
			URL:  "a",
			Mode: 91,
		},
	}
	tests := []struct {
		input struct {
			playlists  []fc2.Playlist
			expectMode int
		}
		expected fc2.Playlist
		title    string
	}{
		{
			input: struct {
				playlists  []fc2.Playlist
				expectMode int
			}{
				playlists:  sortedPlaylists,
				expectMode: 92,
			},
			expected: fc2.Playlist{
				URL:  "a",
				Mode: 92,
			},
			title: "Positive test: exact match",
		},
		{
			input: struct {
				playlists  []fc2.Playlist
				expectMode int
			}{
				playlists:  sortedPlaylists,
				expectMode: 72,
			},
			expected: fc2.Playlist{
				URL:  "b",
				Mode: 52,
			},
			title: "Positive test: no quality",
		},
		{
			input: struct {
				playlists  []fc2.Playlist
				expectMode int
			}{
				playlists:  sortedPlaylists,
				expectMode: 0,
			},
			expected: fc2.Playlist{
				URL:  "b",
				Mode: 52,
			},
			title: "Positive test: neither",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			// Act
			actual, err := fc2.GetPlaylistOrBest(tt.input.playlists, tt.input.expectMode)
			require.NoError(t, err)
			require.Equal(t, tt.expected, *actual)
		})
	}
}
