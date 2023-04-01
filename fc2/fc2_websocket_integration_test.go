//go:build integration

package fc2_test

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/logger"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

type WebSocketIntegrationTestSuite struct {
	suite.Suite
	ctx  context.Context
	impl *fc2.WebSocket
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
	suite.impl = fc2.NewWebSocket(&client, wsURL, 30*time.Second)
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
	msgChan := make(chan *fc2.WSResponse, 100)
	commentChan := make(chan *fc2.Comment, 100)
	defer close(msgChan)
	go func() {
		if err := suite.impl.Listen(suite.ctx, conn, msgChan, commentChan); err != nil {
			logger.I.Fatal("listen failed", zap.Error(err))
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
	go func() {
		if err := suite.impl.HeartbeatLoop(suite.ctx, conn); err != nil {
			logger.I.Fatal("listen failed", zap.Error(err))
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
	msgChan := make(chan *fc2.WSResponse, 100)
	commentChan := make(chan *fc2.Comment, 100)
	defer close(msgChan)
	go func() {
		if err := suite.impl.Listen(suite.ctx, conn, msgChan, commentChan); err != nil {
			logger.I.Fatal("listen failed", zap.Error(err))
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
