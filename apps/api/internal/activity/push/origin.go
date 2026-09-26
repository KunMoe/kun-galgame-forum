package push

import (
	"net/url"
	"strings"
)

func Origin(redirectURI string) string {
	u, err := url.Parse(redirectURI)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func originReady(origin string) bool {
	return strings.HasPrefix(origin, "https://")
}
