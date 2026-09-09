package model

type GalgameListFilter struct {
	Type                 string
	Language             string
	Platform             string
	GameType             string
	SortField            string
	SortOrder            string
	IncludeProviders     []string
	ExcludeOnlyProviders []string
	ReleasedFrom         string
	ReleasedTo           string
	ReleasedMonths       []int
	CollectedFrom        string
	CollectedTo          string
	CollectedMonths      []int
	MinRatingCount       int
	MinRating            float64
	ShowNoResource       bool
	Indexed              bool
	SFWOnly              bool
	RestrictIDs          []int
	Page                 int
	Limit                int
}

func (f GalgameListFilter) HasResourcePredicate() bool {
	return (f.Type != "" && f.Type != "all") ||
		(f.Language != "" && f.Language != "all") ||
		(f.Platform != "" && f.Platform != "all") ||
		len(f.IncludeProviders) > 0 ||
		len(f.ExcludeOnlyProviders) > 0
}

type GalgameResourceMeta struct {
	GalgameID int    `gorm:"column:galgame_id"`
	Platform  string `gorm:"column:platform"`
	Language  string `gorm:"column:language"`
}
