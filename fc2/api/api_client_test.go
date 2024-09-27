//go:build contract

package api_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/cookie"
	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
	impl      *api.Client
	channelID string
}

func (suite *ClientTestSuite) BeforeTest(suiteName, testName string) {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		panic(err)
	}
	_ = cookie.ParseFromFile(jar, "cookies.txt")
	suite.impl = api.NewClient(&http.Client{Jar: jar})
	channelID, err := suite.impl.FindUnrestrictedStream(context.Background())
	suite.Require().NoError(err)
	suite.channelID = channelID
}

func (suite *ClientTestSuite) TestGetMeta() {
	// Act
	_, err := suite.impl.GetMeta(context.Background(), suite.channelID)

	// Assert
	suite.Require().NoError(err)
}

func (suite *ClientTestSuite) TestGetWebSocketURL() {
	// Skip if cookies.txt is not present
	if err := cookie.ParseFromFile(nil, "cookies.txt"); err != nil {
		suite.T().Skip("cookies.txt not found")
	}

	// Arrange
	meta, err := suite.impl.GetMeta(context.Background(), suite.channelID)
	suite.Require().NoError(err)

	err = suite.impl.Login(context.Background())
	suite.Require().NoError(err)

	// Act
	actual, controlToken, err := suite.impl.GetWebSocketURL(context.Background(), meta)

	// Assert
	suite.Require().NoError(err)
	suite.Require().NotEmpty(controlToken.UserName)
	fmt.Printf("got %s\n", actual)
}

func (suite *ClientTestSuite) TestGetWebSocketURLNoLogin() {
	// Arrange
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		panic(err)
	}
	suite.impl = api.NewClient(&http.Client{Jar: jar})
	meta, err := suite.impl.GetMeta(context.Background(), suite.channelID)

	// Act
	actual, controlToken, err := suite.impl.GetWebSocketURL(context.Background(), meta)

	// Assert
	suite.Require().NoError(err)
	suite.Require().Empty(controlToken.UserName)
	fmt.Printf("got %s\n", actual)
}

func (suite *ClientTestSuite) TestLogin() {
	// Skip if cookies.txt is not present
	if err := cookie.ParseFromFile(nil, "cookies.txt"); err != nil {
		suite.T().Skip("cookies.txt not found")
	}

	// Act
	err := suite.impl.Login(context.Background())

	// Assert
	suite.Require().NoError(err)
}

func (suite *ClientTestSuite) TestFindOnlineStream() {
	// Act
	id, err := suite.impl.FindUnrestrictedStream(context.Background())

	// Assert
	suite.Require().NoError(err)
	fmt.Printf("got %s\n", id)
}

func TestClientTestSuite(t *testing.T) {
	log.Logger = log.Logger.With().Caller().Logger()
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.test")

	suite.Run(t, &ClientTestSuite{})
}
