// go:build integration

package fc2_test

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/Darkness4/fc2-live-dl-lite/fc2"
	"github.com/stretchr/testify/suite"
	"nhooyr.io/websocket"
)

type FC2IntegrationTestSuite struct {
	suite.Suite
	wsURL  string
	ctx    context.Context
	client *http.Client
	impl   *fc2.FC2
}

func (suite *FC2IntegrationTestSuite) BeforeTest(suiteName, testName string) {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		panic(err)
	}
	suite.client = &http.Client{
		Jar: jar,
	}
	ls := fc2.NewLiveStream(suite.client, "53539375")
	suite.ctx = context.Background()
	wsURL, err := ls.GetWebSocketURL(suite.ctx)
	if err != nil {
		panic(err)
	}
	suite.wsURL = wsURL
	suite.impl = fc2.New(suite.client, fc2.FC2Params{
		WaitForQualityMaxTries: 15,
	})
}

func (suite *FC2IntegrationTestSuite) TestFetchPlaylist() {
	// Arrange
	msgChan := make(chan *fc2.WSResponse)
	defer close(msgChan)
	ws := fc2.NewWebSocket(suite.client, suite.wsURL, 30*time.Second)
	conn, err := ws.Dial(suite.ctx)
	suite.Require().NoError(err)
	defer conn.Close(websocket.StatusNormalClosure, "ended connection")

	go func() {
		err := ws.Listen(suite.ctx, conn, msgChan, nil)
		suite.Require().NoError(err)
	}()

	playlist, err := suite.impl.FetchPlaylist(
		suite.ctx,
		ws,
		conn,
		msgChan,
	)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(playlist)
}

func TestFC2IntegrationTestSuite(t *testing.T) {
	suite.Run(t, &FC2IntegrationTestSuite{})
}
