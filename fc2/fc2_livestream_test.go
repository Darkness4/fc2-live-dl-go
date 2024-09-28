//go:build contract

package fc2_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/Darkness4/fc2-live-dl-go/testutils/ws"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"

	"github.com/joho/godotenv"
)

type DownloadLiveStreamTestSuite struct {
	suite.Suite
	ctx     context.Context
	hclient *http.Client
	proxy   *ws.Server
	ls      fc2.LiveStream
}

func (suite *DownloadLiveStreamTestSuite) BeforeTest(suiteName, testName string) {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		panic(err)
	}
	suite.hclient = &http.Client{
		Jar: jar,
	}
	client := api.NewClient(suite.hclient)
	channelID, err := client.FindUnrestrictedStream(context.Background())
	suite.Require().NoError(err)
	suite.ctx = log.Logger.WithContext(context.Background())
	meta, err := client.GetMeta(suite.ctx, channelID)
	if err != nil {
		panic(err)
	}
	wsURL, _, err := client.GetWebSocketURL(suite.ctx, meta)
	if err != nil {
		panic(err)
	}
	suite.proxy = ws.NewServer(wsURL, suite.hclient)
	tmpDir := suite.T().TempDir()
	suite.ls = fc2.LiveStream{
		WebsocketURL:   suite.proxy.URL,
		Meta:           meta,
		OutputFileName: tmpDir + "/test.ts",
		ChatFileName:   tmpDir + "/test_chat.json",
		Params: fc2.Params{
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
		},
	}
}

func (suite *DownloadLiveStreamTestSuite) TestDownloadLiveStream() {
	// Act
	ctx, cancel := context.WithCancel(suite.ctx)
	done := make(chan error, 1)
	go func() {
		done <- fc2.DownloadLiveStream(
			ctx,
			suite.hclient,
			suite.ls,
		)
	}()
	time.Sleep(10 * time.Second)
	cancel()
	err := <-done
	suite.Require().NoError(err)
}

func (suite *DownloadLiveStreamTestSuite) TestDownloadLiveStreamPaidProgram() {
	// Act
	done := make(chan error, 1)
	go func() {
		done <- fc2.DownloadLiveStream(
			suite.ctx,
			suite.hclient,
			suite.ls,
		)
	}()
	time.Sleep(10 * time.Second)
	args := api.ControlDisconnectionArguments{
		Code: 4101,
	}
	b, err := json.Marshal(args)
	suite.Require().NoError(err)
	suite.proxy.SendMessage(api.WSResponse{
		Name:      "control_disconnection",
		Arguments: b,
	})
	err = <-done
	suite.Require().Error(err, api.ErrWebSocketPaidProgram.Error())
}

func TestDownloadLiveStreamTestSuite(t *testing.T) {
	log.Logger = log.Logger.With().Caller().Logger()
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.test")

	suite.Run(t, &DownloadLiveStreamTestSuite{})
}
