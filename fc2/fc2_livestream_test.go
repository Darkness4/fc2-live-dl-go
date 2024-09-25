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
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"

	"github.com/joho/godotenv"
)

func TestDownloadLiveStream(t *testing.T) {
	log.Logger = log.Logger.With().Caller().Logger()
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.test")

	// Arrange
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		panic(err)
	}
	hclient := &http.Client{
		Jar: jar,
	}
	client := api.NewClient(hclient)
	ctx := context.Background()
	ctx = log.Logger.WithContext(ctx)
	channelID, err := client.FindOnlineStream(ctx)
	require.NoError(t, err)
	meta, err := client.GetMeta(ctx, channelID)
	require.NoError(t, err)
	wsURL, _, err := client.GetWebSocketURL(ctx, meta)
	require.NoError(t, err)
	tmpDir := t.TempDir()

	// Act
	ctx, cancel := context.WithCancel(ctx)

	done := make(chan error, 1)

	go func() {
		err := fc2.DownloadLiveStream(
			ctx,
			hclient,
			fc2.LiveStream{
				WebsocketURL:   wsURL,
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
			},
		)
		done <- err
	}()
	time.Sleep(10 * time.Second)
	cancel()
	err = <-done
	require.Error(t, err, context.Canceled.Error())
}
