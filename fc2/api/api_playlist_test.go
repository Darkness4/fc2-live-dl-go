//go:build unit

package api_test

import (
	"encoding/json"
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
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

func fixturePlaylists() []api.Playlist {
	var playlists []api.Playlist
	if err := json.Unmarshal([]byte(fixtureStoredPlaylists), &playlists); err != nil {
		panic(err)
	}
	return playlists
}

func TestExtractAndMergePlaylists(t *testing.T) {
	tests := []struct {
		input    api.HLSInformation
		expected []api.Playlist
		title    string
	}{
		{
			input: api.HLSInformation{
				Playlists: []api.Playlist{
					{
						URL:  "a",
						Mode: 50,
					},
				},
				PlaylistsHighLatency: []api.Playlist{
					{
						URL:  "b",
						Mode: 51,
					},
				},
				PlaylistsMiddleLatency: []api.Playlist{
					{
						URL:  "c",
						Mode: 52,
					},
				},
			},
			expected: []api.Playlist{
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
			actual := api.ExtractAndMergePlaylists(tt.input)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestSortPlaylists(t *testing.T) {
	tests := []struct {
		input    []api.Playlist
		expected []api.Playlist
		title    string
	}{
		{
			input: []api.Playlist{
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
			expected: []api.Playlist{
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
			actual := api.SortPlaylists(tt.input)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetPlaylistOrBest(t *testing.T) {
	sortedPlaylists := []api.Playlist{
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
			playlists  []api.Playlist
			expectMode int
		}
		expected api.Playlist
		title    string
	}{
		{
			input: struct {
				playlists  []api.Playlist
				expectMode int
			}{
				playlists:  sortedPlaylists,
				expectMode: 92,
			},
			expected: api.Playlist{
				URL:  "a",
				Mode: 92,
			},
			title: "Positive test: exact match",
		},
		{
			input: struct {
				playlists  []api.Playlist
				expectMode int
			}{
				playlists:  sortedPlaylists,
				expectMode: 72,
			},
			expected: api.Playlist{
				URL:  "b",
				Mode: 52,
			},
			title: "Positive test: no quality",
		},
		{
			input: struct {
				playlists  []api.Playlist
				expectMode int
			}{
				playlists:  sortedPlaylists,
				expectMode: 0,
			},
			expected: api.Playlist{
				URL:  "b",
				Mode: 52,
			},
			title: "Positive test: neither",
		},
		{
			input: struct {
				playlists  []api.Playlist
				expectMode int
			}{
				playlists:  fixturePlaylists(),
				expectMode: 52,
			},
			expected: api.Playlist{
				URL:    "https://us-west-1-media.live.fc2.com/a/stream/92991170/52/playlist?c=UGI9ZdHe2rDzgPTfUzC3Z&d=-0f9-vFxVvzpFQ8vaMPiDBlJnpaeHlW6AzU9Gmwg_4gqx0SocrzfH4xZmzbil_TK",
				Status: json.Number("0"),
				Mode:   52,
			},
			title: "Positive test: fixture",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			// Act
			actual, err := api.GetPlaylistOrBest(tt.input.playlists, tt.input.expectMode)
			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})
	}
}
