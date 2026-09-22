package utils

import (
	"encoding/json"
	"net/url"
	"strconv"

	"kun-galgame-api/pkg/content"

	"github.com/gofiber/fiber/v3"
)

// The keys are the frontend's persisted-settings store, which rides in the
// KUNGalgameSettings cookie rather than an API field — camelCase here is the
// store's own naming, not this API's.
type kunSettings struct {
	ShowKUNGalgameContentLimit       string `json:"showKUNGalgameContentLimit"`
	ShowKUNGalgamePreferOriginalName bool   `json:"showKUNGalgamePreferOriginalName"`
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

const NSFWHeader = "X-Kungal-Nsfw"

// Three lanes in this order: a signed-in session is decided by the account
// stance alone (the cookie it still writes is ignored), the App's Bearer lane
// still reads its header, and an anonymous reader still gets the cookie. The
// cookie branch is not dead code — it is the whole of the logged-out lane.
func IsSFW(c fiber.Ctx) bool {
	if stance, ok := content.FromCtx(c); ok {
		return !stance.AllowsNSFW()
	}
	if nsfw, err := strconv.ParseBool(c.Get(NSFWHeader)); err == nil {
		return !nsfw
	}
	limit := readSettings(c).ShowKUNGalgameContentLimit
	return limit != "nsfw" && limit != "all"
}

// PrefersOriginalName reports whether this reader asked to see each record's
// own name (原名) ahead of its Chinese one.
func PrefersOriginalName(c fiber.Ctx) bool {
	return readSettings(c).ShowKUNGalgamePreferOriginalName
}
