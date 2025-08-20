package cookie_test

import (
	"fmt"
	"net/http/cookiejar"
	"net/url"
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/cookie"
	"github.com/stretchr/testify/require"
)

func TestParseFromFile(t *testing.T) {
	// Act
	jar, err := cookiejar.New(&cookiejar.Options{})
	require.NoError(t, err)
	err = cookie.ParseFromFile(jar, "fixtures/fixture.txt")
	require.NoError(t, err)

	// Assert (https://id.fc2.com/)
	url, err := url.Parse("https://id.fc2.com/")
	require.NoError(t, err)
	cookies := jar.Cookies(url)
	expected := []string{
		"FCSID",
		"fcu",
		"fcus",
		"login_status",
		"secure_check_fc2",
	}

expectLoop:
	for _, test := range expected {
		for _, cookie := range cookies {
			if cookie.Name == test {
				continue expectLoop
			}
		}
		t.Errorf("cookie %s not found", test)
	}

	// Assert (https://live.fc2.com)
	url, err = url.Parse("https://live.fc2.com")
	require.NoError(t, err)
	cookies = jar.Cookies(url)
	fmt.Println(cookies)
	expected = []string{
		"PHPSESSID",
	}

expectLoop2:
	for _, test := range expected {
		for _, cookie := range cookies {
			if cookie.Name == test {
				continue expectLoop2
			}
		}
		t.Errorf("cookie %s not found", test)
	}
}
