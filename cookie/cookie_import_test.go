package cookie_test

import (
	"fmt"
	"net/http/cookiejar"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/cookie"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFromFile(t *testing.T) {
	now := time.Now().Add(24 * time.Hour)
	file, err := os.CreateTemp("", "cookies.txt")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	_, err = file.WriteString(
		fmt.Sprintf("example.com\tFALSE\t/\tFALSE\t%d\tcookiename\tcookievalue\n", now.Unix()),
	)
	require.NoError(t, err)

	// Act
	jar, err := cookiejar.New(&cookiejar.Options{})
	require.NoError(t, err)
	err = cookie.ParseFromFile(jar, file.Name())
	require.NoError(t, err)

	// Assert
	url, err := url.Parse("http://example.com/")
	require.NoError(t, err)
	cookies := jar.Cookies(url)
	require.Len(t, cookies, 1)
	cookie := cookies[0]
	assert.Equal(t, "cookiename", cookie.Name)
	assert.Equal(t, "cookievalue", cookie.Value)
}
