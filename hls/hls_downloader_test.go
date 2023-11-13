package hls

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "embed"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

//go:embed fixtures/playlist.txt
var fixture1 []byte

var expectedURLs1 = []string{
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118606.ts?time=1699894101&hash=4670624c359019cd3a95c84fa0a6690c6c1a7862728e030e5555f49a9416b9d6",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118607.ts?time=1699894102&hash=d9007927f368b2ed45be06f712d06603d7acc55582c8c285db1e4efa9b904f7f",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118608.ts?time=1699894103&hash=38e9f49df48ff7dc1db51da8dc895cfca97c13ad7191100ac27f5914c1fa4cd5",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118609.ts?time=1699894104&hash=4c3d56df768ead84d0c0e9eb49c63266fa5bf81596a22f519dff7c907718894c",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118610.ts?time=1699894105&hash=469a25e33324559dacb316af2cebef2f37fd4e596c78dbfe202e00caf085fadd",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118611.ts?time=1699894106&hash=f92b582874046ce5023f748330fbf5035168c4bd6277799d1adb489bdf6edfe5",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118612.ts?time=1699894107&hash=502b6cb1829de3556c7eb6bf90e4a4bd49865d94db760e5bcc8e10576ba77617",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118613.ts?time=1699894108&hash=0a58f6ba15ef03ae79488cc53f572b7696d5a270dbbf0b2d5fae532091cdcdcb",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118614.ts?time=1699894109&hash=0f035f862395c9c9af31275cc838bd4e9b5ed7f6dc0b231a98055950ff4f2283",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118615.ts?time=1699894110&hash=a3c7b2159c1202833bf58dfdfec8dee78774a467c90d0258164fea85cae43157",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118616.ts?time=1699894111&hash=c3884ff7a03e2376d5f39233ac4960d43ed6a7d9779620e343bbd08c56e8a3bf",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118617.ts?time=1699894112&hash=0cb05a2db11cad553d740d58ea21c6244365e4a27f2974556810ab19c897f7eb",
}

//go:embed fixtures/playlist2.txt
var fixture2 []byte

var expectedURLs2 = []string{
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118607.ts?time=1699894102&hash=d9007927f368b2ed45be06f712d06603d7acc55582c8c285db1e4efa9b904f7f",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118608.ts?time=1699894103&hash=38e9f49df48ff7dc1db51da8dc895cfca97c13ad7191100ac27f5914c1fa4cd5",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118609.ts?time=1699894104&hash=4c3d56df768ead84d0c0e9eb49c63266fa5bf81596a22f519dff7c907718894c",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118610.ts?time=1699894105&hash=469a25e33324559dacb316af2cebef2f37fd4e596c78dbfe202e00caf085fadd",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118611.ts?time=1699894106&hash=f92b582874046ce5023f748330fbf5035168c4bd6277799d1adb489bdf6edfe5",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118612.ts?time=1699894107&hash=502b6cb1829de3556c7eb6bf90e4a4bd49865d94db760e5bcc8e10576ba77617",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118613.ts?time=1699894108&hash=0a58f6ba15ef03ae79488cc53f572b7696d5a270dbbf0b2d5fae532091cdcdcb",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118614.ts?time=1699894109&hash=0f035f862395c9c9af31275cc838bd4e9b5ed7f6dc0b231a98055950ff4f2283",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118615.ts?time=1699894110&hash=a3c7b2159c1202833bf58dfdfec8dee78774a467c90d0258164fea85cae43157",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118616.ts?time=1699894111&hash=c3884ff7a03e2376d5f39233ac4960d43ed6a7d9779620e343bbd08c56e8a3bf",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118617.ts?time=1699894112&hash=0cb05a2db11cad553d740d58ea21c6244365e4a27f2974556810ab19c897f7eb",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118618.ts?time=1699894113&hash=2ac6c20de3fd3060e9dec7895fbc5d074821490f43f1babd5fe3bbdc6bf5bbfa",
}

var combinedExpectedURLs = []string{
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118606.ts?time=1699894101&hash=4670624c359019cd3a95c84fa0a6690c6c1a7862728e030e5555f49a9416b9d6",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118607.ts?time=1699894102&hash=d9007927f368b2ed45be06f712d06603d7acc55582c8c285db1e4efa9b904f7f",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118608.ts?time=1699894103&hash=38e9f49df48ff7dc1db51da8dc895cfca97c13ad7191100ac27f5914c1fa4cd5",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118609.ts?time=1699894104&hash=4c3d56df768ead84d0c0e9eb49c63266fa5bf81596a22f519dff7c907718894c",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118610.ts?time=1699894105&hash=469a25e33324559dacb316af2cebef2f37fd4e596c78dbfe202e00caf085fadd",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118611.ts?time=1699894106&hash=f92b582874046ce5023f748330fbf5035168c4bd6277799d1adb489bdf6edfe5",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118612.ts?time=1699894107&hash=502b6cb1829de3556c7eb6bf90e4a4bd49865d94db760e5bcc8e10576ba77617",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118613.ts?time=1699894108&hash=0a58f6ba15ef03ae79488cc53f572b7696d5a270dbbf0b2d5fae532091cdcdcb",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118614.ts?time=1699894109&hash=0f035f862395c9c9af31275cc838bd4e9b5ed7f6dc0b231a98055950ff4f2283",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118615.ts?time=1699894110&hash=a3c7b2159c1202833bf58dfdfec8dee78774a467c90d0258164fea85cae43157",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118616.ts?time=1699894111&hash=c3884ff7a03e2376d5f39233ac4960d43ed6a7d9779620e343bbd08c56e8a3bf",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118617.ts?time=1699894112&hash=0cb05a2db11cad553d740d58ea21c6244365e4a27f2974556810ab19c897f7eb",
	"https://us-west-1-media-worker1075.live.fc2.com/a/stream/v3/48843568/32/data/118618.ts?time=1699894113&hash=2ac6c20de3fd3060e9dec7895fbc5d074821490f43f1babd5fe3bbdc6bf5bbfa",
}

type DownloaderTestSuite struct {
	suite.Suite
	counter int
	server  *httptest.Server
	impl    *Downloader
}

func (suite *DownloaderTestSuite) BeforeTest(suiteName, testName string) {
	suite.counter = 0
	suite.server = httptest.NewServer(
		http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			if suite.counter == 0 {
				_, _ = res.Write(fixture1)
				suite.counter = 1
			} else {
				_, _ = res.Write(fixture2)
			}
		}),
	)
	suite.impl = NewDownloader(suite.server.Client(), &log.Logger, 10, suite.server.URL)
}

func (suite *DownloaderTestSuite) TestGetFragmentURLs() {
	// Act 1
	urls1, err := suite.impl.GetFragmentURLs(context.Background())

	// Assert 1
	suite.Require().NoError(err)
	suite.Require().Equal(expectedURLs1, urls1)

	// Act 2
	urls2, err := suite.impl.GetFragmentURLs(context.Background())

	// Assert 2
	suite.Require().NoError(err)
	suite.Require().Equal(expectedURLs2, urls2)
}

func (suite *DownloaderTestSuite) TestFillQueue() {
	// Arrange
	urls := make([]string, 0, 11)
	urlsChan := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())

	// Act
	go func() {
		_ = suite.impl.fillQueue(ctx, urlsChan)
	}()

loop:
	for {
		select {
		case url := <-urlsChan:
			urls = append(urls, url)
		case <-time.After(5 * time.Second):
			cancel()
			break loop
		}
	}

	// Assert
	suite.Require().Equal(combinedExpectedURLs, urls)
}

func (suite *DownloaderTestSuite) AfterTest(suiteName, testName string) {
	suite.server.Close()
}

func TestDownloaderTestSuite(t *testing.T) {
	suite.Run(t, &DownloaderTestSuite{})
}
