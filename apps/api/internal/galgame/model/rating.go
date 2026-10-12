package model

import "time"

// GalgameRating is the forum's copy of one NextMoe rating. NextMoe owns every
// field here; galgame_rating also keeps the forum's own view, like_count,
// comment_count and galgame_type, which the mirror never writes. An unrated
// aspect is 0.
type GalgameRating struct {
	ID               int
	WorkID           int
	UserID           int
	Overall          int
	Art              int
	Story            int
	Music            int
	Character        int
	Route            int
	System           int
	Voice            int
	ReplayValue      int
	Recommend        string
	SpoilerLevel     string
	ShortSummary     string
	AuthorState      *string
	AuthorCompletion *string
	ClientID         string
	Site             string
	WrittenHere      bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
