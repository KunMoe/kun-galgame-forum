package model

import (
	"strings"
	"time"
)

// GalgameRating is the forum's copy of one NextMoe rating. NextMoe owns every
// field here; galgame_rating also keeps the forum's own view, like_count,
// comment_count and galgame_type, which the mirror never writes. An unrated
// aspect is 0.
type GalgameRating struct {
	ID                     int
	WorkID                 int
	UserID                 int
	Overall                int
	Art                    int
	Story                  int
	Music                  int
	Character              int
	Route                  int
	System                 int
	Voice                  int
	ReplayValue            int
	Recommend              string
	SpoilerLevel           string
	ShortSummary           string
	ShortSummaryHTML       string
	ShortSummaryText       string
	ShortSummaryHasSpoiler bool
	AuthorState            *string
	AuthorCompletion       *string
	ClientID               string
	Site                   string
	WrittenHere            bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// SummaryShown says whether a review's text may appear outside its own page,
// the rule every NextMoe site applies: a review without spoilers shows, one that
// declares partial spoilers shows only when it masks them itself, and one that
// declares serious spoilers never does.
func SummaryShown(spoilerLevel string, hasSpoilerMask bool) bool {
	return spoilerLevel == "none" || (spoilerLevel == "portion" && hasSpoilerMask)
}

// SummaryExcerpt is NextMoe's short_summary_text as one line per block: the
// text separates blocks with a blank line, which a line-clamped excerpt would
// spend a visible line on.
func SummaryExcerpt(text string) string {
	lines := strings.Split(text, "\n")
	kept := lines[:0]
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}
