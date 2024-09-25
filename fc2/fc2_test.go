//go:build contract

package fc2_test

import (
	"context"
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
	wsURL   string
	hclient *http.Client
	client  *api.Client
	impl    *fc2.FC2
}

func (suite *FC2TestSuite) BeforeTest(suiteName, testName string) {
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
	suite.impl = fc2.New(client, fc2.Params{
		Quality:                    api.Quality2MBps,
		Latency:                    api.LatencyMid,
		PacketLossMax:              20,
		OutFormat:                  "{{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}",
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

func TestFC2TestSuite(t *testing.T) {
	suite.Run(t, &FC2TestSuite{})
}
