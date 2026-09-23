package apiv1

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/filesize"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/resourcevocab"
	"kun-galgame-api/internal/infrastructure/storelink"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

const (
	maxTitle    = 200
	maxNote     = 10000
	maxCode     = 1007
	maxSize     = 64
	maxURLs     = 20
	minURLs     = 1
	maxURLLen   = 4096
	catalogCap  = 100
	previewLen  = 233
	includeRefs = "names,covers,refs"
)

var versionToToken = map[string]string{
	"官方最新": "official_latest",
	"稳定版":  "stable",
	"镜像版":  "mirror",
	"汉化版":  "localized",
	"未知版本": "unknown",
}

var tokenToVersion = map[string]string{
	"official_latest": "官方最新",
	"stable":          "稳定版",
	"mirror":          "镜像版",
	"localized":       "汉化版",
	"unknown":         "未知版本",
}

var downloadLinkSchemes = map[string]bool{
	"http": true, "https": true,
	"ftp": true, "ftps": true,
	"magnet": true, "ed2k": true, "thunder": true,
}

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func validationFailed(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
}

func permissionRequired() *problem.Problem {
	return problem.New(problem.CodePermissionRequired, "The token lacks the permission this decision needs.")
}

func contentRejected() *problem.Problem {
	return problem.New(problem.CodeContentRejected, "The trust-and-safety check refused the submitted text. Nothing was written.")
}

func selfLikeForbidden() *problem.Problem {
	return problem.New(problem.CodeSelfLikeForbidden, "Users cannot like their own topics, replies, comments or galgame resources.")
}

func resourcePublishBanned() *problem.Problem {
	return problem.New(problem.CodeResourcePublishBanned, "This work is banned from publishing download resources.")
}

func tooShort(pointer string, min int) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonTooShort, "too short once surrounding whitespace is removed", &problem.FieldParams{MinLength: &min})
}

func tooLong(pointer string, max int) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonTooLong, "longer than the field allows", &problem.FieldParams{MaxLength: &max})
}

func tooMany(pointer string, max int) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonTooManyItems, "more items than the field allows", &problem.FieldParams{MaxItems: &max})
}

func tooFew(pointer string, min int) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonTooFewItems, "fewer items than the field allows", &problem.FieldParams{MinItems: &min})
}

func duplicateItem(pointer string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonDuplicateItem, "duplicate of an earlier item", nil)
}

func invalidFormat(pointer, detail string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonInvalidFormat, detail, nil)
}

func unknownValue(pointer string, allowed []string) problem.FieldError {
	a := allowed
	return problem.AtPointer(pointer, problem.ReasonUnknownValue, "not in this field's closed vocabulary", &problem.FieldParams{Allowed: &a})
}

func inconsistent(pointer, other string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonInconsistentWith, other, nil)
}

func notAllowed(pointer, detail string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonNotAllowedValue, detail, nil)
}

func parseID(raw string) (int, bool) {
	return repr.ParseID(repr.DecimalID(raw))
}

func pageOf(page, limit int) collect.PageNumber {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultLimit
	}
	return collect.PageNumber{Page: page, Limit: limit}
}

func trimSpace(s string) string {
	return strings.TrimFunc(s, unicode.IsSpace)
}

func canEditResource(authorID int, user *middleware.UserInfo) bool {
	if user == nil {
		return false
	}
	return user.ID == authorID || user.Can(perm.ResourceEditAny)
}

func canDeleteResource(authorID int, user *middleware.UserInfo) bool {
	if user == nil {
		return false
	}
	return user.ID == authorID || user.Can(perm.ResourceDeleteAny)
}

func renderable(users map[int]userclient.User, id int) bool {
	u, ok := users[id]
	return ok && userclient.IsRenderable(u)
}

func stateOf(status int) string {
	if status == 1 {
		return "expired"
	}
	return "valid"
}

func versionToken(stored string) *string {
	tok, ok := versionToToken[strings.TrimSpace(stored)]
	if !ok {
		return nil
	}
	return &tok
}

func versionStored(token *string) string {
	if token == nil || *token == "" {
		return ""
	}
	return tokenToVersion[*token]
}

func languagesOf(r model.GalgameResource) []string {
	langs := []string(r.Languages)
	if len(langs) == 0 {
		langs = []string(resourcevocab.LegacyLanguage(r.Language))
	}
	if langs == nil {
		langs = []string{}
	}
	return langs
}

func platformsOf(r model.GalgameResource) (plats, runs []string) {
	plats, runs = []string(r.Platforms), []string(r.Runtimes)
	if len(plats) == 0 && len(runs) == 0 {
		p, rt := resourcevocab.LegacyPlatform(r.Platform)
		plats, runs = []string(p), []string(rt)
	}
	if plats == nil {
		plats = []string{}
	}
	if runs == nil {
		runs = []string{}
	}
	return plats, runs
}

func typedLangs(in []string) []ResourceLanguage {
	out := make([]ResourceLanguage, len(in))
	for i, v := range in {
		out[i] = ResourceLanguage(v)
	}
	return out
}

func typedPlats(in []string) []ResourcePlatform {
	out := make([]ResourcePlatform, len(in))
	for i, v := range in {
		out[i] = ResourcePlatform(v)
	}
	return out
}

func typedRuns(in []string) []ResourceRuntime {
	out := make([]ResourceRuntime, len(in))
	for i, v := range in {
		out[i] = ResourceRuntime(v)
	}
	return out
}

func typedURLs(in []string) []DownloadURL {
	if in == nil {
		in = []string{}
	}
	out := make([]DownloadURL, len(in))
	for i, v := range in {
		out[i] = DownloadURL(v)
	}
	return out
}

func typedProviders(in []string) []ProviderName {
	if in == nil {
		in = []string{}
	}
	out := make([]ProviderName, len(in))
	for i, v := range in {
		out[i] = ProviderName(v)
	}
	return out
}

func strs[T ~string](in []T) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = string(v)
	}
	return out
}

func validDownloadLink(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" || strings.HasPrefix(value, "#") {
		return false
	}
	u, err := url.Parse(value)
	if err != nil {
		return false
	}
	if !downloadLinkSchemes[strings.ToLower(u.Scheme)] {
		return false
	}
	return u.Host != "" || u.Opaque != "" || u.RawQuery != "" || u.Fragment != ""
}

func uniqueVocab(pointer string, items []string, allowed []string, requiredMin int) ([]string, []problem.FieldError) {
	var errs []problem.FieldError
	if requiredMin > 0 && len(items) < requiredMin {
		return nil, []problem.FieldError{tooFew(pointer, requiredMin)}
	}
	out := make([]string, 0, len(items))
	seen := map[string]int{}
	for i, raw := range items {
		ptr := pointer + "/" + strconv.Itoa(i)
		v := trimSpace(raw)
		if v == "" {
			errs = append(errs, tooShort(ptr, 1))
			continue
		}
		if !slices.Contains(allowed, v) {
			errs = append(errs, unknownValue(ptr, allowed))
			continue
		}
		if _, ok := seen[v]; ok {
			errs = append(errs, duplicateItem(ptr))
			continue
		}
		seen[v] = i
		out = append(out, v)
	}
	return out, errs
}

func validateTitle(raw string, pointer string) (string, []problem.FieldError) {
	if strings.ContainsAny(raw, "\r\n\t") {
		return "", []problem.FieldError{invalidFormat(pointer, "must be a single line")}
	}
	t, ok := resourcevocab.Title(raw)
	if !ok {
		return "", []problem.FieldError{tooLong(pointer, maxTitle)}
	}
	return t, nil
}

func validateSize(raw string, pointer string) (string, []problem.FieldError) {
	size, ok := filesize.Parse(raw)
	if !ok {
		return "", []problem.FieldError{invalidFormat(pointer, "must be N[.NN] MB or GB")}
	}
	if len(size) > maxSize {
		return "", []problem.FieldError{tooLong(pointer, maxSize)}
	}
	return size, nil
}

func validateURLs(raw []DownloadURL, required bool) ([]string, []problem.FieldError) {
	if raw == nil && !required {
		return nil, nil
	}
	if len(raw) > maxURLs {
		return nil, []problem.FieldError{tooMany("/download_urls", maxURLs)}
	}
	if len(raw) < minURLs {
		return nil, []problem.FieldError{tooFew("/download_urls", minURLs)}
	}
	var errs []problem.FieldError
	out := make([]string, 0, len(raw))
	for i, u := range raw {
		ptr := "/download_urls/" + strconv.Itoa(i)
		if len(u) > maxURLLen {
			errs = append(errs, tooLong(ptr, maxURLLen))
			continue
		}
		if !validDownloadLink(string(u)) {
			errs = append(errs, invalidFormat(ptr, "must be an http, https, ftp, ftps, magnet, ed2k or thunder URL"))
			continue
		}
		out = append(out, strings.TrimSpace(string(u)))
	}
	return out, errs
}

type axes struct {
	Type         string
	Title        string
	VersionLabel string
	Language     string
	Platform     string
	Languages    resourcevocab.Keys
	Platforms    resourcevocab.Keys
	Runtimes     resourcevocab.Keys
}

func validateAxes(typ string, langs, plats, runs []string, title string, typeRequired bool) (axes, []problem.FieldError) {
	var errs []problem.FieldError
	if typeRequired && typ == "" {
		errs = append(errs, problem.AtPointer("/resource_type", problem.ReasonRequired, "required", nil))
	} else if typ != "" && !slices.Contains(resourcevocab.TypeKeys, typ) {
		errs = append(errs, unknownValue("/resource_type", resourcevocab.TypeKeys))
	}
	if langs != nil {
		cleaned, lerrs := uniqueVocab("/resource_languages", langs, resourcevocab.LanguageKeys, 1)
		errs = append(errs, lerrs...)
		langs = cleaned
	}
	if plats != nil {
		cleaned, perrs := uniqueVocab("/resource_platforms", plats, resourcevocab.PlatformKeys, 0)
		errs = append(errs, perrs...)
		plats = cleaned
	}
	if runs != nil {
		cleaned, rerrs := uniqueVocab("/resource_runtimes", runs, resourcevocab.RuntimeKeys, 0)
		errs = append(errs, rerrs...)
		runs = cleaned
	}
	t, terrs := validateTitle(title, "/title")
	errs = append(errs, terrs...)
	if len(errs) > 0 {
		return axes{}, errs
	}
	if langs != nil && plats != nil && runs != nil {
		if len(plats) == 0 && len(runs) == 0 {
			errs = append(errs, inconsistent("/resource_platforms", "/resource_runtimes"))
		}
		if resourcevocab.HasRuntimeAxis(typ) && len(runs) == 0 {
			errs = append(errs, tooFew("/resource_runtimes", 1))
		}
		if typ != "" && !resourcevocab.HasRuntimeAxis(typ) && len(runs) > 0 {
			errs = append(errs, inconsistent("/resource_runtimes", "/resource_type"))
		}
	}
	if len(errs) > 0 {
		return axes{}, errs
	}
	out := axes{Type: typ, Title: t, Languages: langs, Platforms: plats, Runtimes: runs}
	if langs != nil {
		out.Language = resourcevocab.CompatLanguage(langs)
	}
	if plats != nil || runs != nil {
		out.Platform = resourcevocab.CompatPlatform(plats, runs)
	}
	return out, nil
}

func dlsiteOf(links storelink.Links) *DlsiteOffer {
	if links.PurchaseURL == "" {
		return nil
	}
	out := &DlsiteOffer{PurchaseURL: links.PurchaseURL}
	if links.CouponURL != "" {
		c := links.CouponURL
		out.CouponURL = &c
	}
	if links.CampaignName != "" {
		n := links.CampaignName
		out.CampaignName = &n
	}
	return out
}

func previewOf(links []string) string {
	if len(links) == 0 {
		return ""
	}
	runes := []rune(links[0])
	if len(runes) <= previewLen {
		return links[0]
	}
	return string(runes[:previewLen])
}

func fromRow(
	r model.GalgameResource,
	author userclient.User,
	work *repr.WorkRef,
	providers []string,
	doc content.ContentDocument,
	dlsite *DlsiteOffer,
	viewer *GalgameResourceViewer,
	cdn string,
) GalgameResource {
	langs := languagesOf(r)
	plats, runs := platformsOf(r)
	out := GalgameResource{
		Object: "galgame_resource", ID: repr.ID(r.ID), Work: work,
		Author: repr.NewUserRef(cdn, author),
		ResourceType: resourcevocab.CompatType(r.Type),
		ResourceLanguages: typedLangs(langs),
		ResourcePlatforms: typedPlats(plats),
		ResourceRuntimes:  typedRuns(runs),
		Title: r.Title, VersionLabel: versionToken(r.VersionLabel), Size: r.Size,
		ProviderNames: typedProviders(providers), Content: doc, State: stateOf(r.Status),
		ViewCount: r.View, DownloadCount: r.Download, LikeCount: r.LikeCount, CommentCount: r.CommentCount,
		CreatedAt: repr.Timestamp(r.CreatedAt), UpdatedAt: repr.Timestamp(r.UpdatedAt),
		EditedAt: repr.TimestampPtr(r.Edited), Dlsite: dlsite, Viewer: viewer,
	}
	if out.ResourceLanguages == nil {
		out.ResourceLanguages = []ResourceLanguage{}
	}
	return out
}
