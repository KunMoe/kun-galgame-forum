package apiv1

import (
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

const (
	mintMaxTitles      = 100
	mintMaxTitleRunes  = 500
	mintMaxIntroRunes  = 50000
	mintMaxNameRunes   = 500
	titleKindOfficial  = 0
	titleKindAlias     = 1
	releaseYearMinimum = 1970
	releaseYearMaximum = 2200
)

// The editspec olang set (nextmoe-infra catalog/editspec/work.go). Official
// titles take their lang from the same set.
var olangAllowed = []string{
	"ar", "be", "bg", "ca", "cs", "da", "de", "el", "en", "eo", "es",
	"fi", "fr", "ga", "gd", "he", "hi", "hr", "hu", "id", "it", "iu",
	"ja", "ko", "la", "lt", "lv", "mk", "ms", "nl", "no", "pl",
	"pt-br", "pt-pt", "ro", "ru", "sk", "sl", "sr", "sv", "ta", "th",
	"tr", "uk", "ur", "vi", "zh", "zh-Hans", "zh-Hant",
}

var introLocales = []string{"en", "ja", "zh-Hans", "zh-Hant"}

var olangSet = func() map[string]bool {
	m := make(map[string]bool, len(olangAllowed))
	for _, l := range olangAllowed {
		m[l] = true
	}
	return m
}()

type mintRequest struct {
	fields   map[string]any
	released *catalogclient.WorkSubmitDate
	display  string
	text     []string
}

func buildMint(body workSubmissionCreate) (*mintRequest, *problem.Problem) {
	var errs []problem.FieldError
	titles := make([]any, 0, len(body.Titles)+len(body.Aliases))
	text := make([]string, 0, len(body.Titles)+len(body.Aliases)+len(body.Introductions)+1)
	seenTitle := map[string]bool{}
	for i, t := range body.Titles {
		at := "/titles/" + strconv.Itoa(i)
		title := strings.TrimSpace(t.Title)
		switch {
		case title == "":
			errs = append(errs, problem.AtPointer(at+"/title", problem.ReasonRequired, "a title must not be blank", nil))
		case utf8.RuneCountInString(t.Title) > mintMaxTitleRunes:
			errs = append(errs, tooLong(at+"/title", mintMaxTitleRunes))
		}
		if !olangSet[t.Locale] {
			errs = append(errs, problem.AtPointer(at+"/locale", problem.ReasonUnknownValue,
				"an official title's locale must be one of catalog's languages", &problem.FieldParams{Allowed: &olangAllowed}))
		}
		key := t.Locale + "\x00" + title
		if seenTitle[key] {
			errs = append(errs, problem.AtPointer(at, problem.ReasonDuplicateItem, "the same title in the same locale is listed twice", nil))
		}
		seenTitle[key] = true
		titles = append(titles, map[string]any{"lang": t.Locale, "title": title, "kind": titleKindOfficial})
		text = append(text, title)
	}
	seenAlias := map[string]bool{}
	for i, a := range body.Aliases {
		alias := string(a)
		v := strings.TrimSpace(alias)
		if v == "" || seenAlias[v] {
			continue
		}
		if utf8.RuneCountInString(alias) > mintMaxTitleRunes {
			errs = append(errs, tooLong("/aliases/"+strconv.Itoa(i), mintMaxTitleRunes))
		}
		seenAlias[v] = true
		titles = append(titles, map[string]any{"lang": "", "title": v, "kind": titleKindAlias})
		text = append(text, v)
	}
	if len(titles) > mintMaxTitles {
		limit := mintMaxTitles
		errs = append(errs, problem.AtPointer("/titles", problem.ReasonTooManyItems,
			"titles and aliases together exceed catalog's limit", &problem.FieldParams{MaxItems: &limit}))
	}

	display := strings.TrimSpace(body.DisplayName)
	if display == "" {
		display = displayNameFromTitles(body.Titles)
	}
	switch {
	case display == "":
		errs = append(errs, problem.AtPointer("/display_name", problem.ReasonRequired, "send display_name or a non-blank title", nil))
	case utf8.RuneCountInString(display) > mintMaxNameRunes:
		errs = append(errs, tooLong("/display_name", mintMaxNameRunes))
	}
	text = append(text, display)

	olang := ""
	if body.OriginalLanguage != nil {
		olang = strings.TrimSpace(*body.OriginalLanguage)
	}
	switch {
	case olang == "":
		errs = append(errs, problem.AtPointer("/original_language", problem.ReasonRequired, "original_language must be a language, not null", nil))
	case !olangSet[olang]:
		errs = append(errs, problem.AtPointer("/original_language", problem.ReasonUnknownValue,
			"not one of catalog's languages", &problem.FieldParams{Allowed: &olangAllowed}))
	}

	intros := make([]any, 0, len(body.Introductions))
	seenIntro := map[string]bool{}
	for i, intro := range body.Introductions {
		at := "/introductions/" + strconv.Itoa(i)
		if !slices.Contains(introLocales, intro.Locale) {
			errs = append(errs, problem.AtPointer(at+"/locale", problem.ReasonUnknownValue,
				"catalog keeps introductions in these languages only", &problem.FieldParams{Allowed: &introLocales}))
		}
		if seenIntro[intro.Locale] {
			errs = append(errs, problem.AtPointer(at+"/locale", problem.ReasonDuplicateItem, "one introduction per locale", nil))
		}
		seenIntro[intro.Locale] = true
		v := strings.TrimSpace(intro.Value)
		if v == "" {
			continue
		}
		if utf8.RuneCountInString(intro.Value) > mintMaxIntroRunes {
			errs = append(errs, tooLong(at+"/value", mintMaxIntroRunes))
		}
		intros = append(intros, map[string]any{"lang": intro.Locale, "intro": v})
		text = append(text, v)
	}

	released, perr := releasedOf(body.ReleaseDate, body.ReleaseDatePrecision)
	if perr != nil {
		errs = append(errs, *perr)
	}
	if len(errs) > 0 {
		return nil, validationFailed(errs...)
	}

	fields := map[string]any{
		"catalog.work.display_name":   display,
		"catalog.work.olang":          olang,
		"catalog.work.content_rating": contentRatingCode(body.ContentRating),
		"catalog.work.display_nsfw":   body.IsNSFW,
		"catalog.work.titles":         titles,
	}
	if len(intros) > 0 {
		fields["catalog.work.intros"] = intros
	}
	return &mintRequest{fields: fields, released: released, display: display, text: text}, nil
}

func tooLong(pointer string, limit int) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonTooLong, "longer than catalog accepts", &problem.FieldParams{MaxLength: &limit})
}

func contentRatingCode(rating string) int {
	switch rating {
	case "sensitive":
		return 1
	case "r18":
		return 2
	default:
		return 0
	}
}

func displayNameFromTitles(titles []SubmissionTitle) string {
	for _, locale := range []string{"ja", "zh-Hans", "zh-Hant", "en"} {
		for _, t := range titles {
			if v := strings.TrimSpace(t.Title); t.Locale == locale && v != "" {
				return v
			}
		}
	}
	for _, t := range titles {
		if v := strings.TrimSpace(t.Title); v != "" {
			return v
		}
	}
	return ""
}

func releasedOf(date, precision *string) (*catalogclient.WorkSubmitDate, *problem.FieldError) {
	if date == nil {
		if precision != nil {
			fe := problem.AtPointer("/release_date_precision", problem.ReasonInconsistentWith, "/release_date is absent", nil)
			return nil, &fe
		}
		return nil, nil
	}
	t, err := time.Parse(time.DateOnly, *date)
	if err != nil {
		fe := problem.AtPointer("/release_date", problem.ReasonInvalidFormat, "release_date must be YYYY-MM-DD", nil)
		return nil, &fe
	}
	if t.Year() < releaseYearMinimum || t.Year() > releaseYearMaximum {
		lo, hi := float64(releaseYearMinimum), float64(releaseYearMaximum)
		fe := problem.AtPointer("/release_date", problem.ReasonOutOfRange, "catalog accepts release years 1970 to 2200",
			&problem.FieldParams{Minimum: &lo, Maximum: &hi})
		return nil, &fe
	}
	out := &catalogclient.WorkSubmitDate{Y: int16(t.Year())}
	p := "day"
	if precision != nil {
		p = *precision
	}
	switch p {
	case "month":
		out.M = int16(t.Month())
	case "day":
		out.M, out.D = int16(t.Month()), int16(t.Day())
	}
	return out, nil
}

func submissionPointer(upstream string) (string, bool) {
	switch {
	case upstream == "/display_name", strings.HasPrefix(upstream, "/field_values/catalog.work.display_name"):
		return "/display_name", true
	case strings.HasPrefix(upstream, "/field_values/catalog.work.olang"):
		return "/original_language", true
	case strings.HasPrefix(upstream, "/field_values/catalog.work.titles"):
		return "/titles", true
	case strings.HasPrefix(upstream, "/field_values/catalog.work.intros"):
		return "/introductions", true
	case strings.HasPrefix(upstream, "/field_values/catalog.work.content_rating"):
		return "/content_rating", true
	case strings.HasPrefix(upstream, "/field_values/catalog.work.display_nsfw"):
		return "/is_nsfw", true
	case strings.HasPrefix(upstream, "/released"):
		return "/release_date", true
	}
	return "", false
}
