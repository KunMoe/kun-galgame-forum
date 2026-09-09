package service

import (
	"strconv"
	"strings"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
)

var submissionLocales = []struct{ product, lang string }{
	{"ja-jp", "ja"},
	{"zh-cn", "zh-Hans"},
	{"zh-tw", "zh-Hant"},
	{"en-us", "en"},
}

func olangOf(productCode string) string {
	code := strings.ToLower(strings.TrimSpace(productCode))
	for _, l := range submissionLocales {
		if l.product == code {
			return l.lang
		}
	}
	return code
}

const (
	contentRatingAllAges = 0
	contentRatingR18     = 2
)

const (
	titleKindOfficial = 0
	titleKindAlias    = 1
)

type SubmissionForm struct {
	NameEnUS string `json:"name_en_us"`
	NameJaJP string `json:"name_ja_jp"`
	NameZhCN string `json:"name_zh_cn"`
	NameZhTW string `json:"name_zh_tw"`

	IntroEnUS string `json:"intro_en_us"`
	IntroJaJP string `json:"intro_ja_jp"`
	IntroZhCN string `json:"intro_zh_cn"`
	IntroZhTW string `json:"intro_zh_tw"`

	ContentLimit     string   `json:"content_limit"`
	AgeLimit         string   `json:"age_limit"`
	OriginalLanguage string   `json:"original_language"`
	Aliases          []string `json:"aliases"`

	ReleaseDate string `json:"release_date"`
	BannerHash  string `json:"banner_hash"`

	// Set by the client only after the submitter has been shown the same-title
	// works catalog found and said it is a different game anyway.
	ConfirmDuplicates bool `json:"confirm_duplicates"`
}

func (f *SubmissionForm) names() map[string]string {
	return map[string]string{
		"ja-jp": f.NameJaJP, "zh-cn": f.NameZhCN,
		"zh-tw": f.NameZhTW, "en-us": f.NameEnUS,
	}
}

func (f *SubmissionForm) intros() map[string]string {
	return map[string]string{
		"ja-jp": f.IntroJaJP, "zh-cn": f.IntroZhCN,
		"zh-tw": f.IntroZhTW, "en-us": f.IntroEnUS,
	}
}

func (f *SubmissionForm) DisplayName() string {
	names := f.names()
	for _, l := range submissionLocales {
		if v := strings.TrimSpace(names[l.product]); v != "" {
			return v
		}
	}
	return ""
}

func (f *SubmissionForm) Fields() map[string]any {
	fields := map[string]any{
		"catalog.work.display_name": f.DisplayName(),
		"catalog.work.olang":        olangOf(f.OriginalLanguage),
		"catalog.work.display_nsfw": f.ContentLimit == "nsfw",
	}
	rating := contentRatingAllAges
	if f.AgeLimit == "r18" {
		rating = contentRatingR18
	}
	fields["catalog.work.content_rating"] = rating

	names := f.names()
	titles := make([]any, 0, len(submissionLocales)+len(f.Aliases))
	for _, l := range submissionLocales {
		if v := strings.TrimSpace(names[l.product]); v != "" {
			titles = append(titles, map[string]any{"lang": l.lang, "title": v, "kind": titleKindOfficial})
		}
	}
	for _, alias := range f.Aliases {
		if v := strings.TrimSpace(alias); v != "" {
			titles = append(titles, map[string]any{"lang": "", "title": v, "kind": titleKindAlias})
		}
	}
	if len(titles) > 0 {
		fields["catalog.work.titles"] = titles
	}

	intros := f.intros()
	introList := make([]any, 0, len(submissionLocales))
	for _, l := range submissionLocales {
		if v := strings.TrimSpace(intros[l.product]); v != "" {
			introList = append(introList, map[string]any{"lang": l.lang, "intro": v})
		}
	}
	if len(introList) > 0 {
		fields["catalog.work.intros"] = introList
	}
	return fields
}

func (f *SubmissionForm) Released() (*catalogclient.WorkSubmitDate, *errors.AppError) {
	raw := strings.TrimSpace(f.ReleaseDate)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, "-")
	if len(parts) == 0 || len(parts) > 3 {
		return nil, errors.ErrValidation("发售日期格式应为 YYYY-MM-DD")
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, errors.ErrValidation("发售日期格式应为 YYYY-MM-DD")
		}
		nums[i] = n
	}
	if nums[0] <= 0 {
		return nil, errors.ErrValidation("发售日期需要年份")
	}
	if nums[1] == 0 && nums[2] != 0 {
		return nil, errors.ErrValidation("发售日期不能只有日而没有月")
	}
	return &catalogclient.WorkSubmitDate{
		Y: int16(nums[0]), M: int16(nums[1]), D: int16(nums[2]),
	}, nil
}

// A cover element accepts image_hash, kind, portrait_pinned, sexual and
// violence and NOTHING else — catalog's editspec rejects an unknown key
// outright. The original patch also sent sort_order, source and source_key
// (column names, not patch keys), so every submitted banner came back
// `element 0: unknown key "sort_order"` and was dropped by the warn-and-carry-on
// in Submit: zero submission banners reached production before 2026-08.
func (f *SubmissionForm) CoverPatch() map[string]any {
	hash := strings.TrimSpace(f.BannerHash)
	if hash == "" {
		return nil
	}
	return map[string]any{
		"catalog.work.covers": []any{map[string]any{"image_hash": hash}},
	}
}
