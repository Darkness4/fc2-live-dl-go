// Package cookie provides a function to parse a Netscape cookie file and add the cookies to a cookie jar.
package cookie

import (
	"bufio"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// ParseFromFile parses a netscape cookie file and adds the cookies to the jar.
func ParseFromFile(jar http.CookieJar, cookieFile string) error {
	file, err := os.Open(cookieFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Ignore comment and empty line
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// Parse the line and extract the cookie fields.
		fields := strings.Fields(line)
		if len(fields) < 7 {
			log.Warn().Str("line", line).Msg("skipped (not enough fields)")
			continue
		}
		domain := fields[0]
		// field[1] is not handle (includeSubdomains, which is the inverse of hostOnly, which isn't used here)
		path := fields[2]
		isSecure, _ := strconv.ParseBool(fields[3])
		expiresUnix, _ := strconv.ParseInt(fields[4], 10, 64)
		name := fields[5]
		value := fields[6]

		// Convert the Unix timestamp to a time.Time object.
		expires := time.Unix(expiresUnix, 0)

		if expires.Before(time.Now()) && expires != time.Unix(0, 0) {
			log.Warn().Str("line", line).Msg("skipped (expired)")
			continue
		}

		// Create a new cookie object and add it to the jar.
		cookie := &http.Cookie{
			Name:     name,
			Value:    value,
			Domain:   domain,
			Path:     path,
			HttpOnly: false, // Isn't known in Netscape cookies
			Secure:   isSecure,
		}
		if expires != time.Unix(0, 0) {
			cookie.Expires = expires
		}
		jar.SetCookies(&url.URL{Scheme: "http", Host: domain}, []*http.Cookie{cookie})
	}
	return nil
}
