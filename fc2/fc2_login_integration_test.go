package fc2_test

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/cookie"
	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/stretchr/testify/require"
)

// TestLogin requires a cookies.txt file in the package.
func TestLogin(t *testing.T) {
	// Arrange
	jar, err := cookiejar.New(&cookiejar.Options{})
	require.NoError(t, err)
	err = cookie.ParseFromFile(jar, "cookies.txt")
	require.NoError(t, err)
	client := &http.Client{Jar: jar, Timeout: time.Minute}

	// Act
	err = fc2.Login(context.Background(), fc2.WithHTTPClient(client))

	// Assert
	require.NoError(t, err)
}
