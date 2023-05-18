package fc2

import "sort"

func ExtractAndMergePlaylists(hlsInfo *HLSInformation) []Playlist {
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

func GetPlaylistOrBest(sortedPlaylists []Playlist, expectMode int) (*Playlist, error) {
	if len(sortedPlaylists) == 0 {
		return nil, ErrWebSocketEmptyPlaylist
	}

	var playlist *Playlist
	for _, p := range sortedPlaylists {
		if p.Mode == expectMode {
			playlist = &p
			break
		}
	}

	// If no playlist matches, ignore the quality and find the best
	// one matching the latency
	if playlist == nil {
		for _, p := range sortedPlaylists {
			pl := LatencyFromMode(p.Mode)
			el := LatencyFromMode(expectMode)
			if pl == el {
				playlist = &p
				break
			}
		}
	}

	// If no playlist matches, get first
	if playlist == nil {
		playlist = &sortedPlaylists[0]
	}

	return playlist, nil
}
