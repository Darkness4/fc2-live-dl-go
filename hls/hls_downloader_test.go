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
	"https://us-west-1-media-worker1001.live.fc2.com/a/stream/v3/48843568/32/data/30469.ts?time=1699805959&hash=6b8c4a08700ee16480cb471c4d1c6542101ebd43b5d3358dcddb325ea3206db7",
	"https://us-west-1-media-worker1001.live.fc2.com/a/stream/v3/48843568/32/data/30470.ts?time=1699805960&hash=cfaa7e9ff7eb6f658d3d536be2bb5c10ac27d14ce0017d20ecf07e437c05fa03",
	"https://us-west-1-media-worker1001.live.fc2.com/a/stream/v3/48843568/32/data/30471.ts?time=1699805961&hash=582801e533841d1f3d50940f7329706f4c5e8472e90fb2d85dc87c974990dd50",
	"https://us-west-1-media-worker1001.live.fc2.com/a/stream/v3/48843568/32/data/30472.ts?time=1699805962&hash=1d9073e134c55b1bc33fa7a668db44408ee8c80ed7b031864a9b2d8eedddd7a6",
	"https://us-west-1-media-worker1001.live.fc2.com/a/stream/v3/48843568/32/data/30473.ts?time=1699805963&hash=88501e5af0770da45bbe72483ea3be251273d7d26d015a54c493e54de418fbf7",
	"https://us-west-1-media-worker1001.live.fc2.com/a/stream/v3/48843568/32/data/30474.ts?time=1699805964&hash=76ba8c4cd652682971b1716877cc1d34560bbb85c8d423ccec30fc22f0329679",
	"https://us-west-1-media-worker1001.live.fc2.com/a/stream/v3/48843568/32/data/30475.ts?time=1699805965&hash=cb405c2ddcb4dc056ac7779af36421c104cd28d9d7852781af39258c0534cc32",
	"https://us-west-1-media-worker1001.live.fc2.com/a/stream/v3/48843568/32/data/30476.ts?time=1699805966&hash=dfb57aa8b8aa31586f544acdd7e82f0bf129c6dcc0d13bd10303db5df6e88a04",
	"https://us-west-1-media-worker1001.live.fc2.com/a/stream/v3/48843568/32/data/30477.ts?time=1699805967&hash=aad83c6cbf1d692ee8655718c3778ab86319ae3f518e3a17cc0bc8a7b029c390",
	"https://us-west-1-media-worker1001.live.fc2.com/a/stream/v3/48843568/32/data/30478.ts?time=1699805968&hash=7e643522a9e7d50146eeb475e7ccaed565a8a9ac4f773456d9d0a46ec353109d",
	"https://us-west-1-media-worker1001.live.fc2.com/a/stream/v3/48843568/32/data/30479.ts?time=1699805969&hash=ec28ea8ed8f8890047c790fe3777eef94e17408442189b47d57ee990c8a3cb20",
	"https://us-west-1-media-worker1001.live.fc2.com/a/stream/v3/48843568/32/data/30480.ts?time=1699805970&hash=e8099b9be074cad734d52d1aa2b54b877cd7ddddb58d681882418d0553ede8e2",
}

type DownloaderTestSuite struct {
	suite.Suite
	server *httptest.Server
	impl   *Downloader
}

func (suite *DownloaderTestSuite) BeforeTest(suiteName, testName string) {
	suite.server = httptest.NewServer(
		http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.Write(fixture1)
		}),
	)
	suite.impl = NewDownloader(suite.server.Client(), &log.Logger, 10, suite.server.URL)
}

func (suite *DownloaderTestSuite) TestGetFragmentURLs() {
	// Act
	urls, err := suite.impl.GetFragmentURLs(context.Background())

	// Assert
	suite.Require().NoError(err)
	suite.Require().Equal(expectedURLs1, urls)
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
	suite.Require().Equal(expectedURLs1, urls)
}

func (suite *DownloaderTestSuite) AfterTest(suiteName, testName string) {
	suite.server.Close()
}

func TestDownloaderTestSuite(t *testing.T) {
	suite.Run(t, &DownloaderTestSuite{})
}
