package dto

type GalgameArtMeta struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Thumbhash string `json:"thumbhash,omitempty"`
}

type GalgameRatingBucket struct {
	Score int `json:"score"`
	Count int `json:"count"`
}

type GalgameRatingStats struct {
	Average *float64 `json:"average,omitempty"`
	Stdev   *float64 `json:"stdev,omitempty"`
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
}

type GalgameExternalRating struct {
	Source       string                `json:"source"`
	Score        float64               `json:"score"`
	VoteCount    int                   `json:"vote_count"`
	Rank         *int                  `json:"rank,omitempty"`
	Distribution []GalgameRatingBucket `json:"distribution,omitempty"`
	Stats        *GalgameRatingStats   `json:"stats,omitempty"`
}

type GalgamePlaytime struct {
	Source    string `json:"source"`
	Minutes   int    `json:"minutes"`
	VoteCount int    `json:"vote_count"`
}

// GalgameIntro is one language's introduction. The language list is whatever
// catalog actually carries rather than a fixed set: the four product slots this
// replaces always shipped an empty 繁體中文, because catalog has never held a
// zh-Hant intro row, and the reader got a tab that could only say 暂无对应翻译.
type GalgameIntro struct {
	Lang    string `json:"lang"`
	Intro   string `json:"intro"`
	Machine bool   `json:"machine"`
}
