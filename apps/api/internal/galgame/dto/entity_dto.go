package dto

type GalgameCard struct {
	ID                         int            `json:"id"`
	Name                       string         `json:"name"`
	NameOriginal               string         `json:"name_original"`
	User                       UserBrief      `json:"user"`
	ContentLimit               string         `json:"content_limit"`
	View                       int            `json:"view"`
	LikeCount                  int            `json:"like_count"`
	Rating                     float64        `json:"rating"`
	RatingCount                int            `json:"rating_count"`
	ResourceUpdateTime         string         `json:"resource_update_time"`
	Platform                   []string       `json:"platform"`
	Language                   []string       `json:"language"`
	ReleaseDate                *string        `json:"release_date"`
	ReleaseDateTBA             bool           `json:"release_date_tba"`
	ReleasePrecision           string         `json:"release_precision,omitempty"`
	EffectiveBannerHash        string         `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL         string         `json:"effective_banner_url,omitempty"`
	EffectiveBannerWidth       int            `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight      int            `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash   string         `json:"effective_banner_thumbhash,omitempty"`
	EffectivePortraitHash      string         `json:"effective_portrait_hash,omitempty"`
	EffectivePortraitURL       string         `json:"effective_portrait_url,omitempty"`
	EffectivePortraitWidth     int            `json:"effective_portrait_width,omitempty"`
	EffectivePortraitHeight    int            `json:"effective_portrait_height,omitempty"`
	EffectivePortraitThumbhash string         `json:"effective_portrait_thumbhash,omitempty"`
	IsOnForum                  bool           `json:"is_on_forum"`
	Status                     int            `json:"status,omitempty"`
	Company                    string         `json:"company,omitempty"`
	ViaOfficial                *OfficialBrief `json:"via_official,omitempty"`
}

type OfficialBrief struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GalgameSample struct {
	Name                     string `json:"name"`
	EffectiveBannerHash      string `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL       string `json:"effective_banner_url,omitempty"`
	EffectiveBannerWidth     int    `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight    int    `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash string `json:"effective_banner_thumbhash,omitempty"`
}

type TaxonomySearchItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Logo string `json:"logo,omitempty"`
}

type OfficialListItem struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Link         string   `json:"link"`
	Category     string   `json:"category"`
	Logo         string   `json:"logo"`
	Lang         string   `json:"lang"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgame_count"`
}

type OfficialListPage struct {
	Officials []OfficialListItem `json:"officials"`
	Total     int64              `json:"total"`
}

type OfficialLink struct {
	Source string `json:"source"`
	Name   string `json:"name"`
	URL    string `json:"url"`
}

type OfficialDetail struct {
	ID                  int            `json:"id"`
	Name                string         `json:"name"`
	Original            string         `json:"original"`
	Links               []OfficialLink `json:"links"`
	Link                string         `json:"link"`
	Logo                string         `json:"logo"`
	Category            string         `json:"category"`
	Lang                string         `json:"lang"`
	Description         string         `json:"description"`
	DescriptionMachine  bool           `json:"description_machine"`
	Alias               []string       `json:"alias"`
	Galgame             []GalgameCard  `json:"galgame"`
	GalgameCount        int64          `json:"galgame_count"`
	OwnGalgameCount     int64          `json:"own_galgame_count"`
	ImprintGalgameCount int64          `json:"imprint_galgame_count"`
	MovedTo             int            `json:"moved_to,omitempty"`
}

type OfficialRelationNode struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Logo      string `json:"logo"`
	WorkCount int    `json:"work_count"`
}

type OfficialRelationEdge struct {
	From     int    `json:"from"`
	To       int    `json:"to"`
	Relation string `json:"relation"`
}

type OfficialRelationGraph struct {
	Nodes []OfficialRelationNode `json:"nodes"`
	Edges []OfficialRelationEdge `json:"edges"`
}

type EngineListItem struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgame_count"`
}

type EngineDetail struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Alias        []string      `json:"alias"`
	Galgame      []GalgameCard `json:"galgame"`
	GalgameCount int64         `json:"galgame_count"`
}

type SeriesListItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	GalgameCount int    `json:"galgame_count"`
}

type SeriesSample struct {
	Name                     string `json:"name"`
	EffectiveBannerHash      string `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL       string `json:"effective_banner_url,omitempty"`
	EffectiveBannerThumbhash string `json:"effective_banner_thumbhash,omitempty"`
}

type SeriesCard struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	IsNSFW bool   `json:"is_nsfw"`
	// GalgameCount is what this site lists — a member with no resource, or one a
	// ban unpublished, is not in it. CatalogGalgameCount is how many the series
	// has upstream, which is what the detail page one click away renders. They
	// are routinely different (斬魔大聖デモンベイン: 1 and 3), and the card used to
	// print only the first under the word 共, so the browse list said "1 部" about
	// a series whose own page then listed three.
	GalgameCount        int            `json:"galgame_count"`
	CatalogGalgameCount int            `json:"catalog_galgame_count"`
	SampleGalgame       []SeriesSample `json:"sample_galgame"`
}

type SeriesCardPage struct {
	Series []SeriesCard `json:"series"`
	Total  int64        `json:"total"`
}

type SeriesDetail struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Galgame      []GalgameCard `json:"galgame"`
	GalgameCount int64         `json:"galgame_count"`
}

type TagListItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	GalgameCount int    `json:"galgame_count"`
}

type TagListPage struct {
	Tags  []TagListItem `json:"tags"`
	Total int64         `json:"total"`
}

type TagDetail struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Category     string        `json:"category"`
	Hidden       bool          `json:"hidden"`
	Description  string        `json:"description"`
	Alias        []string      `json:"alias"`
	Galgame      []GalgameCard `json:"galgame"`
	GalgameCount int64         `json:"galgame_count"`
}

type StaffWork struct {
	GalgameCard
	CatalogID  int      `json:"catalog_id"`
	Roles      []string `json:"roles"`
	Characters []string `json:"characters,omitempty"`
}

type StaffLink struct {
	Source string `json:"source"`
	Name   string `json:"name"`
	URL    string `json:"url,omitempty"`
}

type StaffSibling struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type StaffDetail struct {
	ID           int            `json:"id"`
	Name         string         `json:"name"`
	NameOriginal string         `json:"name_original,omitempty"`
	Latin        string         `json:"latin,omitempty"`
	Intro        string         `json:"intro"`
	IntroMachine bool           `json:"intro_machine"`
	Photo        string         `json:"photo"`
	Gender       *int           `json:"gender"`
	BirthY       *int           `json:"birth_y"`
	BirthM       *int           `json:"birth_m"`
	BirthD       *int           `json:"birth_d"`
	Links        []StaffLink    `json:"links"`
	Siblings     []StaffSibling `json:"siblings"`
	Roles        []string       `json:"roles"`
	Works        []StaffWork    `json:"works"`
	NextOffset   *int           `json:"next_offset"`
	MovedTo      int            `json:"moved_to,omitempty"`
}

type EntitySearchItem struct {
	ID        int    `json:"id"`
	Family    string `json:"family"`
	Name      string `json:"name"`
	Alias     string `json:"alias,omitempty"`
	Image     string `json:"image,omitempty"`
	WorkCount int    `json:"work_count,omitempty"`
}

type EntitySearchGroup struct {
	Family string             `json:"family"`
	Total  int64              `json:"total"`
	Items  []EntitySearchItem `json:"items"`
}
