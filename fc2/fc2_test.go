//go:build contract

package fc2_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/stretchr/testify/suite"
)

type FC2TestSuite struct {
	suite.Suite
	wsURL  string
	client *api.Client
	impl   *fc2.FC2
}

func (suite *FC2TestSuite) BeforeTest(suiteName, testName string) {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		panic(err)
	}
	hclient := http.Client{
		Jar: jar,
	}
	suite.client = api.NewClient(&hclient)
	channelID, err := suite.client.FindOnlineStream(context.Background())
	suite.Require().NoError(err)
	tmpDir := suite.T().TempDir()
	suite.impl = fc2.New(suite.client, fc2.Params{
		Quality:       api.Quality2MBps,
		Latency:       api.LatencyMid,
		PacketLossMax: 20,
		OutFormat: fmt.Sprintf(
			"%s/{{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}",
			tmpDir,
		),
		WriteChat:                  true,
		WriteInfoJSON:              true,
		WriteThumbnail:             true,
		WaitForLive:                true,
		WaitForQualityMaxTries:     15,
		AllowQualityUpgrade:        true,
		PollQualityUpgradeInterval: 10 * time.Second,
		WaitPollInterval:           5 * time.Second,
		Remux:                      true,
		Concat:                     true,
		KeepIntermediates:          true,
		ScanDirectory:              "",
		EligibleForCleaningAge:     48 * time.Hour,
		DeleteCorrupted:            true,
		ExtractAudio:               true,
	}, channelID)
}

func (suite *FC2TestSuite) TestWatch() {
	ctx, cancel := context.WithCancel(context.Background())

	// Act
	done := make(chan error, 1)
	go func() {
		err := suite.impl.Watch(ctx)
		done <- err
	}()

	// Assert
	time.Sleep(10 * time.Second)
	cancel()
	err := <-done
	suite.Require().NoError(err, context.Canceled.Error())
}

func (suite *FC2TestSuite) TestWatchRestrictedStream() {
	ctx, cancel := context.WithCancel(context.Background())

	// Arrange
	channelID, err := suite.client.FindRestrictedStream(ctx)
	suite.Require().NoError(err)
	impl := fc2.New(suite.client, suite.impl.Params, channelID)

	// Act
	done := make(chan error, 1)
	go func() {
		err := impl.Watch(ctx)
		done <- err
	}()

	// Assert
	time.Sleep(2 * time.Second)
	cancel()
	err = <-done
	suite.Require().NoError(err, api.ErrWebSocketLoginRequired.Error())
}

func TestFC2TestSuite(t *testing.T) {
	suite.Run(t, &FC2TestSuite{})
}
