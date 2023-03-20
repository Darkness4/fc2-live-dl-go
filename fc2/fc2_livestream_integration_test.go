// go:build integration

package fc2_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/Darkness4/fc2-live-dl-lite/fc2"
	"github.com/stretchr/testify/suite"
)

type LiveStreamIntegrationTestSuite struct {
	suite.Suite
	impl *fc2.LiveStream
}

func (suite *LiveStreamIntegrationTestSuite) BeforeTest(suiteName, testName string) {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		panic(err)
	}
	suite.impl = fc2.NewLiveStream(&http.Client{
		Jar: jar,
	}, "53539375")
}

func (suite *LiveStreamIntegrationTestSuite) TestWaitForIsOnline() {
	// Act
	suite.impl.WaitForOnline(context.Background(), time.Second)
}

func (suite *LiveStreamIntegrationTestSuite) TestIsOnline() {
	// Act
	actual := suite.impl.IsOnline(context.Background())

	// Assert
	suite.Require().Equal(true, actual)
}

func (suite *LiveStreamIntegrationTestSuite) TestGetMeta() {
	// Act
	_, err := suite.impl.GetMeta(context.Background())

	// Assert
	suite.Require().NoError(err)
}

func (suite *LiveStreamIntegrationTestSuite) TestWebSocketURL() {
	// Act
	actual, err := suite.impl.GetWebSocketURL(context.Background())

	// Assert
	suite.Require().NoError(err)
	fmt.Printf("got %s\n", actual)
}

func TestLiveStreamIntegrationTestSuite(t *testing.T) {
	suite.Run(t, &LiveStreamIntegrationTestSuite{})
}
