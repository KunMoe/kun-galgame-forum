package utils

import (
	"encoding/json"
	"net/url"

	"github.com/gofiber/fiber/v3"
)

// The keys are the frontend's persisted-settings store, which rides in the
// KUNGalgameSettings cookie rather than an API field — camelCase here is the
// store's own naming, not this API's.
type kunSettings struct {
	ShowKUNGalgamePreferOriginalName bool `json:"showKUNGalgamePreferOriginalName"`
}

func readSettings(c fiber.Ctx) kunSettings {
	var settings kunSettings

	raw := c.Cookies("KUNGalgameSettings", "")
	if raw == "" {
		return settings
	}

	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		decoded = raw
	}
	if err := json.Unmarshal([]byte(decoded), &settings); err != nil {
		return kunSettings{}
	}
	return settings
}

// PrefersOriginalName reports whether this reader asked to see each record's
// own name (原名) ahead of its Chinese one.
func PrefersOriginalName(c fiber.Ctx) bool {
	return readSettings(c).ShowKUNGalgamePreferOriginalName
}
