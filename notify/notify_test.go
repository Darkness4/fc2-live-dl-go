//go:build integration

package notify_test

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/notify"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

type GoNotifierTestSuite struct {
	suite.Suite
	impl notify.Notifier
}

func (suite *GoNotifierTestSuite) TestNotify() {
	// Test the Notify method
	err := suite.impl.Notify(context.Background(), "Test Title", "Test Message", 1)

	// Assertions
	suite.NoError(err)
}

func TestGoNotifierTestSuite(t *testing.T) {
	err := godotenv.Load(".env.test")
	if err != nil {
		// Skip test if not defined
		log.Err(err).Msg("Error loading .env.test file")
	} else {
		suite.Run(t, &GoNotifierTestSuite{
			impl: notify.NewGoNotifier(http.DefaultClient, os.Getenv("NOTIFIER_ENDPOINT"), os.Getenv("NOTIFIER_TOKEN")),
		})
	}
}
