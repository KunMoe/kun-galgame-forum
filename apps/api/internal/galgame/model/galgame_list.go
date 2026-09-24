package model

type GalgameListFilter struct {
	Type                 string
	Language             string
	Platform             string
	PlatformAxis         string
	LanguageAxis         string
	PlatformAxes         []string
	LanguageAxes         []string
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
		f.PlatformAxis != "" || f.LanguageAxis != "" ||
		len(f.PlatformAxes) > 0 || len(f.LanguageAxes) > 0 ||
		len(f.IncludeProviders) > 0 ||
		len(f.ExcludeOnlyProviders) > 0
}
