package catalogclient

import (
	"context"
	"net/url"
	"sync"
	"time"
)

type VocabularyValue struct {
	Value       string `json:"value"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
}

type Vocabulary struct {
	Name   string            `json:"name"`
	Closed bool              `json:"closed"`
	Values []VocabularyValue `json:"values"`
}

// The catalog's closed vocabularies change only when catalog ships, so one hour
// is a cache and not a staleness risk. It is held in process rather than in
// redis because the whole answer is ~6KB and every edit page needs it.
const vocabularyTTL = time.Hour

type vocabularyCache struct {
	mu      sync.Mutex
	byName  map[string]Vocabulary
	fetched time.Time
}

// Vocabularies answers the enum token sets the edit schema's `vocabulary` names.
// Without it every enum field reaches the browser with no options and the form
// renders it read-only.
func (c *Client) Vocabularies(ctx context.Context) (map[string]Vocabulary, error) {
	c.vocabularies.mu.Lock()
	defer c.vocabularies.mu.Unlock()
	if c.vocabularies.byName != nil && time.Since(c.vocabularies.fetched) < vocabularyTTL {
		return c.vocabularies.byName, nil
	}
	var page struct {
		Items []Vocabulary `json:"items"`
	}
	if err := c.appV2JSON(ctx, "/v2/vocabularies", url.Values{}, &page); err != nil {
		return nil, err
	}
	byName := make(map[string]Vocabulary, len(page.Items))
	for _, v := range page.Items {
		byName[v.Name] = v
	}
	c.vocabularies.byName = byName
	c.vocabularies.fetched = time.Now()
	return byName, nil
}
