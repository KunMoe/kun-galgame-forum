package apiv1

import (
	"encoding/json"
	"net/url"

	"kun-galgame-api/pkg/perm"
)

const (
	permCreate = perm.WebsiteCreate
	permEdit   = perm.WebsiteEdit
	permDelete = perm.WebsiteDelete
)

func validURL(raw string) bool {
	if len(raw) > 100 {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func jsonArray(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	raw, err := json.Marshal(values)
	return string(raw), err
}
