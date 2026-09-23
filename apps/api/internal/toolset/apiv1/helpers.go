package apiv1

import (
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/cron"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/internal/toolset/repository"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

const (
	maxName       = 500
	maxMarkdown   = 2000
	maxAliases    = 17
	maxHomepages  = 10
	maxHomepage   = 500
	maxURL        = 1007
	maxSizeLabel  = 107
	maxNote       = 1007
	maxFileSize   = 2147483648
	minFileSize   = 1
	dailyBase     = 100 * 1024 * 1024
	bytesPerMB    = 1024 * 1024
)

var archiveExts = map[string]bool{".7z": true, ".zip": true, ".rar": true}

var downloadLinkSchemes = map[string]bool{
	"http": true, "https": true,
	"ftp": true, "ftps": true,
	"magnet": true, "ed2k": true, "thunder": true,
}

var sortSpecs = map[string]repository.SortSpec{
	"resource_updated_desc": {Column: "resource_update_time", Desc: true},
	"resource_updated_asc":  {Column: "resource_update_time", Desc: false},
	"created_desc":          {Column: "created", Desc: true},
	"created_asc":           {Column: "created", Desc: false},
	"view_desc":             {Column: "view", Desc: true},
	"view_asc":              {Column: "view", Desc: false},
	"name_asc":              {Column: "name", Desc: false},
	"name_desc":             {Column: "name", Desc: true},
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

func alreadyExists(pointer string) *problem.Problem {
	return problem.New(problem.CodeAlreadyExists, "Another record already uses this value.",
		problem.AtPointer(pointer, problem.ReasonNotAllowedValue, "already used by another record", nil))
}

func invalidTransition(detail string) *problem.Problem {
	return problem.New(problem.CodeInvalidStateTransition, detail)
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

func duplicateItem(pointer string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonDuplicateItem, "duplicate of an earlier item", nil)
}

func invalidFormat(pointer, detail string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonInvalidFormat, detail, nil)
}

func immutable(pointer string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonImmutable, "this field cannot be changed", nil)
}

func inconsistent(pointer, other string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonInconsistentWith, other, nil)
}

func unknownRef(pointer string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonUnknownReference, "the referenced upload is not a completed upload of the caller on this toolset", nil)
}

func requiredField(pointer string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonRequired, "required", nil)
}

func outOfRange(pointer string, min, max float64) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonOutOfRange, "outside the allowed range", &problem.FieldParams{Minimum: &min, Maximum: &max})
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

func trimName(s string) string {
	return strings.TrimFunc(s, unicode.IsSpace)
}

func validHomepage(raw string) bool {
	if len(raw) > maxHomepage {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
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

func archiveFilename(name string) bool {
	return archiveExts[strings.ToLower(filepath.Ext(name))]
}

func distOf(agg repository.PracticalityAgg) []int {
	return []int{agg.Distribution[0], agg.Distribution[1], agg.Distribution[2], agg.Distribution[3], agg.Distribution[4]}
}

func emptyDist() []int { return []int{0, 0, 0, 0, 0} }

func parseInstant(raw string) *repr.DateTime {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			v := repr.Timestamp(t)
			return &v
		}
	}
	return nil
}

func fileSizeOf(r model.GalgameToolsetResource, uploads map[string]model.ToolsetUpload) *int64 {
	if r.Type != "s3" {
		return nil
	}
	if u, ok := uploads[r.ArtifactUUID]; ok && u.FileSize > 0 {
		n := u.FileSize
		return &n
	}
	if n, err := strconv.ParseInt(strings.TrimSpace(r.Size), 10, 64); err == nil && n >= 0 {
		return &n
	}
	z := int64(0)
	return &z
}

func sizeLabelOf(r model.GalgameToolsetResource) *string {
	if r.Type != "user" {
		return nil
	}
	s := r.Size
	return &s
}

func canEditToolset(authorID int, user *middleware.UserInfo) bool {
	if user == nil {
		return false
	}
	return user.ID == authorID || user.Can(perm.ToolsetEditAny)
}

func canDeleteToolset(authorID int, user *middleware.UserInfo) bool {
	if user == nil {
		return false
	}
	return user.ID == authorID || user.Can(perm.ToolsetDeleteAny)
}

func canEditResource(posterID int, user *middleware.UserInfo) bool {
	if user == nil {
		return false
	}
	return user.ID == posterID || user.Can(perm.ToolsetResourceEditAny)
}

func canDeleteResource(posterID int, user *middleware.UserInfo) bool {
	if user == nil {
		return false
	}
	return user.ID == posterID || user.Can(perm.ToolsetResourceDeleteAny)
}

func quotaExceeded() error {
	p := problem.New(problem.CodeQuotaExceeded, "The caller's daily upload quota is exhausted.")
	sec := secondsUntilDailyReset()
	h := http.Header{}
	h.Set("Retry-After", strconv.Itoa(sec))
	return huma.ErrorWithHeaders(p, h)
}

func nilIface(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Ptr && rv.IsNil()
}

func secondsUntilDailyReset() int {
	loc := cron.ScheduleLocation()
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	sec := int(next.Sub(now).Seconds())
	if sec < 1 {
		return 1
	}
	return sec
}
