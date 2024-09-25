package api

import (
	"sort"
)

// ExtractAndMergePlaylists extracts and merges the playlists.
func ExtractAndMergePlaylists(hlsInfo HLSInformation) []Playlist {
	playlists := make(
		[]Playlist,
		0,
		len(
			hlsInfo.Playlists,
		)+len(
			hlsInfo.PlaylistsHighLatency,
		)+len(
			hlsInfo.PlaylistsMiddleLatency,
		),
	)
	playlists = append(playlists, hlsInfo.Playlists...)
	playlists = append(playlists, hlsInfo.PlaylistsHighLatency...)
	playlists = append(playlists, hlsInfo.PlaylistsMiddleLatency...)
	return playlists
}

// SortPlaylists sorts the playlists by mode.
func SortPlaylists(playlists []Playlist) []Playlist {
	sortedList := make([]Playlist, len(playlists))
	copy(sortedList, playlists)

	sort.Slice(sortedList, func(i, j int) bool {
		modeI := sortedList[i].Mode
		if modeI >= 90 {
			modeI -= 90
		}
		modeJ := sortedList[j].Mode
		if modeJ >= 90 {
			modeJ -= 90
		}
		return modeI > modeJ
	})

	return sortedList
}

// GetPlaylistOrBest returns the playlist that matches the mode or the best.
func GetPlaylistOrBest(sortedPlaylists []Playlist, expectMode int) (Playlist, error) {
	if len(sortedPlaylists) == 0 {
		return Playlist{}, ErrWebSocketEmptyPlaylist
	}

	var playlist Playlist
	for _, p := range sortedPlaylists {
		if p.Mode == expectMode {
			playlist = p
			break
		}
	}

	// If no playlist matches, ignore the quality and find the best
	// one matching the latency
	if playlist.URL == "" {
		for _, p := range sortedPlaylists {
			pl := LatencyFromMode(p.Mode)
			el := LatencyFromMode(expectMode)
			if pl == el {
				playlist = p
				break
			}
		}
	}

	// If no playlist matches, get first
	if playlist.URL == "" {
		playlist = sortedPlaylists[0]
	}

	return playlist, nil
}
