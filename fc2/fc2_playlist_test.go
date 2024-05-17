//go:build unit

package fc2_test

import (
	"encoding/json"
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/stretchr/testify/require"
)

const fixtureStoredPlaylists = `[
  {
    "mode": 52,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/52/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 51,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/51/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 50,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/50/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 42,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/42/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 41,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/41/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 40,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/40/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 32,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/32/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 31,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/31/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 30,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/30/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 22,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/22/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 21,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/21/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 20,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/20/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 12,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/12/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 11,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/11/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 10,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/10/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 2,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/2/master_playlist?targets=10,20,30,40,50,90&c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 92,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/92/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 91,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/91/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 1,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/1/master_playlist?targets=10,20,30,40,50,90&c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 90,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/90/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  },
  {
    "mode": 0,
    "status": 0,
    "url": "https://us-west-1-media.live.fc2.com/a/stream/92991170/0/master_playlist?targets=10,20,30,40,50,90&c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK"
  }
]`

func fixturePlaylists() []fc2.Playlist {
	var playlists []fc2.Playlist
	if err := json.Unmarshal([]byte(fixtureStoredPlaylists), &playlists); err != nil {
		panic(err)
	}
	return playlists
}

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
		{
			input: struct {
				playlists  []fc2.Playlist
				expectMode int
			}{
				playlists:  fixturePlaylists(),
				expectMode: 52,
			},
			expected: fc2.Playlist{
				URL:  "https://us-west-1-media.live.fc2.com/a/stream/92991170/52/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK",
				Mode: 52,
			},
			title: "Positive test: fixture",
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
