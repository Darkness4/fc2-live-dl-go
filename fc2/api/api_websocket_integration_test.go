//go:build integration

package api_test

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/coder/websocket"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

type WebSocketIntegrationTestSuite struct {
	suite.Suite
	ctx  context.Context
	impl *api.WebSocket
}

func (suite *WebSocketIntegrationTestSuite) BeforeTest(suiteName, testName string) {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		panic(err)
	}
	client := http.Client{
		Jar: jar,
	}
	suite.ctx = context.Background()
	ls := fc2.NewLiveStream(&client, "8829230")
	wsURL, err := ls.GetWebSocketURL(suite.ctx)
	if err != nil {
		panic(err)
	}
	suite.impl = api.NewWebSocket(&client, wsURL, 30*time.Second)
}

func (suite *WebSocketIntegrationTestSuite) TestDial() {
	// Act
	conn, err := suite.impl.Dial(suite.ctx)

	// Assert
	suite.Require().NoError(err)
	conn.Close(websocket.StatusNormalClosure, "close")
}

func (suite *WebSocketIntegrationTestSuite) TestListen() {
	// Arrange
	conn, err := suite.impl.Dial(suite.ctx)
	suite.Require().NoError(err)
	defer conn.Close(websocket.StatusNormalClosure, "close")

	// Act
	msgChan := make(chan *api.WSResponse, 100)
	commentChan := make(chan *api.Comment, 100)
	defer close(msgChan)
	go func() {
		if err := suite.impl.Listen(suite.ctx, conn, msgChan, commentChan); err != nil {
			log.Fatal().Err(err).Msg("listen failed")
		}
	}()
	time.Sleep(5 * time.Second)
}

func (suite *WebSocketIntegrationTestSuite) TestHealthCheckLoop() {
	// Arrange
	conn, err := suite.impl.Dial(suite.ctx)
	suite.Require().NoError(err)
	defer conn.Close(websocket.StatusNormalClosure, "close")

	// Act
	msgChan := make(chan *api.WSResponse, 100)
	go func() {
		if err := suite.impl.HeartbeatLoop(suite.ctx, conn, msgChan); err != nil {
			log.Fatal().Err(err).Msg("heartbeat failed")
		}
	}()
	time.Sleep(5 * time.Second)
}

func (suite *WebSocketIntegrationTestSuite) TestGetHLSInformation() {
	// Arrange
	conn, err := suite.impl.Dial(suite.ctx)
	suite.Require().NoError(err)
	defer conn.Close(websocket.StatusNormalClosure, "close")

	// Act
	msgChan := make(chan *api.WSResponse, 100)
	commentChan := make(chan *api.Comment, 100)
	defer close(msgChan)
	go func() {
		if err := suite.impl.Listen(suite.ctx, conn, msgChan, commentChan); err != nil {
			log.Fatal().Err(err).Msg("listen failed")
		}
	}()

	msg, err := suite.impl.GetHLSInformation(suite.ctx, conn, msgChan)
	suite.Require().NoError(err)
	suite.Require().Condition(func() (success bool) {
		return len(msg.Playlists) > 0
	})
}

func TestWebSocketIntegrationTestSuite(t *testing.T) {
	suite.Run(t, &WebSocketIntegrationTestSuite{})
}
