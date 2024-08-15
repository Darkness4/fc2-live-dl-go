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
	"github.com/coder/websocket"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

func init() {
	log.Logger = log.Logger.With().Caller().Logger()
}

type DownloaderIntegrationTestSuite struct {
	suite.Suite
	ctx       context.Context
	ctxCancel context.CancelFunc
	client    *http.Client
	impl      *hls.Downloader
	msgChan   chan *fc2.WSResponse
	conn      *websocket.Conn
	ws        *fc2.WebSocket
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
	suite.ctx, suite.ctxCancel = context.WithCancel(context.Background())

	// Check livestream
	ls := fc2.NewLiveStream(suite.client, "48863711")

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
		suite.Require().Error(err, context.Canceled.Error())
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
	ctx, cancel := context.WithTimeout(suite.ctx, 10*time.Second)
	defer cancel()
	f, err := os.Create("output.ts")
	if err != nil {
		suite.Require().NoError(err)
		return
	}
	defer f.Close()

	errChan := make(chan error, 1)

	go func() {
		_, err := suite.impl.Read(ctx, f, hls.DefaultCheckpoint())
		if err != nil {
			errChan <- err
		}
	}()

	for {
		select {
		case err := <-errChan:
			suite.Require().Error(err, context.Canceled.Error())
			return
		}
	}
}

func (suite *DownloaderIntegrationTestSuite) AfterTest(suiteName, testName string) {
	suite.ctxCancel()

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
