//go:build integration

package hls_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"testing"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/hls"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
	"nhooyr.io/websocket"
)

type DownloaderIntegrationTestSuite struct {
	suite.Suite
	ctx     context.Context
	client  *http.Client
	impl    *hls.Downloader
	msgChan chan *fc2.WSResponse
	conn    *websocket.Conn
	ws      *fc2.WebSocket
}

func (suite *DownloaderIntegrationTestSuite) fetchPlaylist() *fc2.Playlist {
	hlsInfo, err := suite.ws.GetHLSInformation(suite.ctx, suite.conn, suite.msgChan)
	suite.Require().NoError(err)

	playlist, err := fc2.GetPlaylistOrBest(
		fc2.SortPlaylists(fc2.ExtractAndMergePlaylists(hlsInfo)),
		50,
	)
	suite.Require().NoError(err)

	return playlist
}

func (suite *DownloaderIntegrationTestSuite) BeforeTest(suiteName, testName string) {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		panic(err)
	}
	suite.client = &http.Client{
		Jar: jar,
	}
	suite.ctx = context.Background()

	// Check livestream
	ls := fc2.NewLiveStream(suite.client, "8829230")

	// Get WS and listen to it
	wsURL, err := ls.GetWebSocketURL(suite.ctx)
	if err != nil {
		panic(err)
	}
	suite.msgChan = make(chan *fc2.WSResponse)
	suite.ws = fc2.NewWebSocket(suite.client, wsURL, 30*time.Second)
	suite.conn, err = suite.ws.Dial(suite.ctx)
	suite.Require().NoError(err)
	go func() {
		err := suite.ws.Listen(suite.ctx, suite.conn, suite.msgChan, nil)
		suite.Require().NoError(err)
	}()

	// Fetch playlist
	playlist := suite.fetchPlaylist()

	// Prepare implementation
	suite.impl = hls.NewDownloader(suite.client, &log.Logger, 8, playlist.URL)
}

func (suite *DownloaderIntegrationTestSuite) TestGetFragmentURLs() {
	urls, err := suite.impl.GetFragmentURLs(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(urls)
	fmt.Println(urls)
}

func (suite *DownloaderIntegrationTestSuite) TestRead() {
	ctx, cancel := context.WithTimeout(suite.ctx, 20*time.Second)
	defer cancel()
	out := make(chan []byte)
	go func() {
		_ = suite.impl.Read(ctx, out)
	}()

	go func(out <-chan []byte) {
		f, err := os.Create("output.ts")
		if err != nil {
			suite.Require().NoError(err)
			return
		}
		defer f.Close()

		for data := range out {
			_, err := f.Write(data)
			if err != nil {
				return
			}
		}
	}(out)

	select {
	case <-ctx.Done():
		fmt.Println("Done")
	}
}

func (suite *DownloaderIntegrationTestSuite) AfterTest(suiteName, testName string) {
	// Clean up
	if suite.conn != nil {
		suite.conn.Close(websocket.StatusNormalClosure, "ended connection")
	}
	if suite.msgChan != nil {
		close(suite.msgChan)
	}
}

func TestDownloaderIntegrationTestSuite(t *testing.T) {
	suite.Run(t, &DownloaderIntegrationTestSuite{})
}
