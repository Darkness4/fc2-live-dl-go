//go:build contract

package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"testing"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/Darkness4/fc2-live-dl-go/testutils/ws"
	"github.com/Darkness4/fc2-live-dl-go/utils/try"
	"github.com/coder/websocket"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

type WebSocketTestSuite struct {
	suite.Suite
	ctx   context.Context
	proxy *ws.Server
	impl  *api.WebSocket
}

func (suite *WebSocketTestSuite) BeforeTest(suiteName, testName string) {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		panic(err)
	}
	hclient := http.Client{
		Jar: jar,
	}
	client := api.NewClient(&hclient)
	channelID, err := client.FindOnlineStream(context.Background())
	suite.Require().NoError(err)
	suite.ctx = context.Background()
	meta, err := client.GetMeta(suite.ctx, channelID)
	if err != nil {
		panic(err)
	}
	wsURL, _, err := client.GetWebSocketURL(suite.ctx, meta)
	if err != nil {
		panic(err)
	}
	suite.proxy = ws.NewServer(wsURL, &hclient)
	suite.impl = api.NewWebSocket(&hclient, suite.proxy.URL, 5*time.Second)
}

func (suite *WebSocketTestSuite) TestDial() {
	// Act
	conn, err := suite.impl.Dial(suite.ctx)

	// Assert
	suite.Require().NoError(err)
	conn.Close(websocket.StatusNormalClosure, "close")
}

func (suite *WebSocketTestSuite) TestListen() {
	// Arrange
	conn, err := suite.impl.Dial(suite.ctx)
	suite.Require().NoError(err)

	// Act
	msgChan := make(chan *api.WSResponse, 100)
	commentChan := make(chan *api.Comment, 100)
	done := make(chan error, 1)
	go func() {
		done <- suite.impl.Listen(suite.ctx, conn, msgChan, commentChan)
	}()
	time.Sleep(5 * time.Second)
	conn.Close(websocket.StatusNormalClosure, "close")
	err = <-done
	suite.Require().Error(err, io.EOF.Error())
}

func (suite *WebSocketTestSuite) TestHealthCheckLoop() {
	// Arrange
	conn, err := suite.impl.Dial(suite.ctx)
	suite.Require().NoError(err)

	ctx, cancel := context.WithCancel(suite.ctx)

	// Act
	msgChan := make(chan *api.WSResponse, 100)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if err := suite.impl.Listen(ctx, conn, msgChan, nil); err != nil &&
			!errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
			log.Fatal().Err(err).Msg("listen failed")
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		if err := suite.impl.HeartbeatLoop(ctx, conn, msgChan); err != nil &&
			!errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
			log.Fatal().Err(err).Msg("heartbeat failed")
		}
		wg.Done()
	}()
	time.Sleep(15 * time.Second)
	cancel()
	wg.Wait()
}

func (suite *WebSocketTestSuite) TestGetHLSInformation() {
	// Arrange
	conn, err := suite.impl.Dial(suite.ctx)
	suite.Require().NoError(err)

	// Act
	msgChan := make(chan *api.WSResponse, 100)
	commentChan := make(chan *api.Comment, 100)
	done := make(chan error, 1)
	go func() {
		done <- suite.impl.Listen(suite.ctx, conn, msgChan, commentChan)
	}()

	// Try multiple time as FC2 may not return HLS information immediately.
	msg, err := try.DoWithResult(5, time.Second, func(_ int) (api.HLSInformation, error) {
		return suite.impl.GetHLSInformation(suite.ctx, conn, msgChan)
	})
	suite.Require().NoError(err)
	suite.Require().Condition(func() (success bool) {
		return len(msg.Playlists) > 0
	})

	conn.Close(websocket.StatusNormalClosure, "close")
	err = <-done
	suite.Require().Error(err, io.EOF.Error())
}

func (suite *WebSocketTestSuite) TestFetchPlaylist() {
	// Arrange
	conn, err := suite.impl.Dial(suite.ctx)
	suite.Require().NoError(err)

	// Act
	msgChan := make(chan *api.WSResponse, 100)
	commentChan := make(chan *api.Comment, 100)
	done := make(chan error, 1)
	go func() {
		done <- suite.impl.Listen(suite.ctx, conn, msgChan, commentChan)
	}()

	ret, err := try.DoWithResult(5, time.Second, func(_ int) (struct {
		playlist   api.Playlist
		availables []api.Playlist
	}, error) {
		playlist, availables, err := suite.impl.FetchPlaylist(
			suite.ctx,
			conn,
			msgChan,
			32,
		)
		return struct {
			playlist   api.Playlist
			availables []api.Playlist
		}{playlist, availables}, err
	})
	suite.Require().NoError(err)
	suite.Require().NotEmpty(ret.playlist)
	suite.Require().NotEmpty(ret.availables)
	fmt.Println(ret.playlist)
	conn.Close(websocket.StatusNormalClosure, "close")
	err = <-done
	suite.Require().Error(err, io.EOF.Error())
}

func (suite *WebSocketTestSuite) TestControlDisconnection() {
	tt := []struct {
		code     int
		expected error
	}{
		{
			code:     4101,
			expected: api.ErrWebSocketPaidProgram,
		},
		{
			code:     4507,
			expected: api.ErrWebSocketLoginRequired,
		},
		{
			code:     4512,
			expected: api.ErrWebSocketLoginRequired,
		},
		{
			code:     9999,
			expected: api.ErrWebSocketServerDisconnection,
		},
	}

	for _, tc := range tt {
		suite.T().Run(fmt.Sprintf("code %d", tc.code), func(t *testing.T) {
			// Arrange
			conn, err := suite.impl.Dial(suite.ctx)
			suite.Require().NoError(err)

			// Act
			msgChan := make(chan *api.WSResponse, 100)
			done := make(chan error, 1)
			go func() {
				done <- suite.impl.Listen(suite.ctx, conn, msgChan, nil)
			}()

			time.Sleep(2 * time.Second)
			args := api.ControlDisconnectionArguments{
				Code: tc.code,
			}
			b, err := json.Marshal(args)
			suite.Require().NoError(err)
			suite.proxy.SendMessage(api.WSResponse{
				Name:      "control_disconnection",
				Arguments: b,
			})
			err = <-done
			suite.Require().Error(err, tc.expected)
		})
	}
}

func (suite *WebSocketTestSuite) AfterTest(suiteName, testName string) {
	suite.proxy.Close()
	// Sleep to avoid multiple connections on websocket
	time.Sleep(1 * time.Second)
}

func TestWebSocketTestSuite(t *testing.T) {
	log.Logger = log.Logger.With().Caller().Logger()
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.test")

	suite.Run(t, &WebSocketTestSuite{})
}
