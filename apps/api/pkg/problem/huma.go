package problem

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/validation"
)

var bodyIndex = regexp.MustCompile(`\[(\d+)]`)

func FromHuma(ctx huma.Context, status int, msg string, errs ...error) *Problem {
	requestID, instance := "", ""
	if ctx != nil {
		u := ctx.URL()
		instance = u.RequestURI()
		if v := ctx.Header(HeaderRequestID); ValidRequestID(v) {
			requestID = v
		}
	}
	if requestID == "" {
		requestID = NewRequestID()
	}

	fields := make([]FieldError, 0, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		fields = append(fields, fieldFromHuma(err))
	}

	fields = dropImpliedLengthErrors(fields)
	code := pickCode(ctx, status, msg, fields)
	detail := msg
	var cause error
	if code == CodeInternalError {
		detail = internalDetail
		fields = []FieldError{}
		for _, err := range errs {
			if err != nil {
				cause = err
				break
			}
		}
	}
	p := New(code, detail, fields...)
	p.RequestID = requestID
	p.Instance = instance
	p.cause = cause
	return p
}

// An enum's maxLength is its longest key, so an unknown token longer than that
// ("windows" against the three-letter platform keys) also failed maxLength, and
// the extra TOO_LONG turned UNKNOWN_ENUM_VALUE into INVALID_PARAMETER.
func dropImpliedLengthErrors(fields []FieldError) []FieldError {
	where := func(f FieldError) string {
		switch {
		case f.Parameter != nil:
			return "p:" + *f.Parameter
		case f.Pointer != nil:
			return "b:" + *f.Pointer
		}
		return ""
	}
	unknown := map[string]bool{}
	for _, f := range fields {
		if k := where(f); k != "" && f.Reason == ReasonUnknownValue {
			unknown[k] = true
		}
	}
	if len(unknown) == 0 {
		return fields
	}
	out := fields[:0:0]
	for _, f := range fields {
		if unknown[where(f)] && (f.Reason == ReasonTooLong || f.Reason == ReasonTooShort) {
			continue
		}
		out = append(out, f)
	}
	return out
}

func pickCode(ctx huma.Context, status int, msg string, fields []FieldError) string {
	switch status {
	case http.StatusUnsupportedMediaType:
		return CodeUnsupportedMediaType
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusMethodNotAllowed:
		return CodeMethodNotAllowed
	case http.StatusInternalServerError:
		return CodeInternalError
	case http.StatusServiceUnavailable:
		return CodeServiceUnavailable
	case http.StatusUnauthorized:
		if ctx == nil || ctx.Header("Authorization") == "" {
			return CodeMissingCredential
		}
		return CodeInvalidCredential
	}

	if len(fields) > 0 {
		allParamOrHeader := true
		allUnknown := true
		anyBody := false
		for _, f := range fields {
			if f.Pointer != nil {
				anyBody = true
				allParamOrHeader = false
			}
			if f.Parameter == nil && f.Header == nil {
				allParamOrHeader = false
			}
			if f.Reason != ReasonUnknownValue {
				allUnknown = false
			}
		}
		if anyBody {
			if status == http.StatusBadRequest {
				return CodeMalformedBody
			}
			return CodeValidationFailed
		}
		if allParamOrHeader {
			if allUnknown {
				if namedParam(fields, "sort") {
					return CodeUnknownSort
				}
				return CodeUnknownEnumValue
			}
			if namedParam(fields, "cursor") {
				return CodeInvalidCursor
			}
			if limitTooLarge(fields) {
				return CodeLimitTooLarge
			}
			return CodeInvalidParameter
		}
	}

	if status == http.StatusBadRequest && isMalformedBodyMessage(msg) {
		return CodeMalformedBody
	}
	return StatusToCode(status)
}

func namedParam(fields []FieldError, name string) bool {
	for _, f := range fields {
		if f.Parameter != nil && *f.Parameter == name {
			return true
		}
	}
	return false
}

func limitTooLarge(fields []FieldError) bool {
	for _, f := range fields {
		if f.Parameter == nil || *f.Parameter != "limit" {
			continue
		}
		if f.Reason != ReasonOutOfRange {
			continue
		}
		if f.Params != nil && f.Params.Maximum != nil {
			return true
		}
	}
	return false
}

func isMalformedBodyMessage(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "request body") || strings.Contains(lower, "json")
}

func fieldFromHuma(err error) FieldError {
	var d *huma.ErrorDetail
	if errors.As(err, &d) && d != nil {
		return mapDetail(d)
	}
	var ed huma.ErrorDetailer
	if errors.As(err, &ed) && ed != nil {
		if got := ed.ErrorDetail(); got != nil {
			return mapDetail(got)
		}
	}
	empty := ""
	return FieldError{Pointer: &empty, Reason: ReasonInvalidFormat, Detail: err.Error()}
}

func mapDetail(d *huma.ErrorDetail) FieldError {
	reason, params, requiredProp := mapReason(d.Message)
	fe := FieldError{Reason: reason, Detail: d.Message, Params: params}
	loc := d.Location
	switch {
	case strings.HasPrefix(loc, "query."):
		name := strings.TrimPrefix(loc, "query.")
		fe.Parameter = &name
	case strings.HasPrefix(loc, "path."):
		name := strings.TrimPrefix(loc, "path.")
		fe.Parameter = &name
	case strings.HasPrefix(loc, "header."):
		name := strings.TrimPrefix(loc, "header.")
		fe.Header = &name
	case loc == "body" || strings.HasPrefix(loc, "body.") || strings.HasPrefix(loc, "body["):
		ptr := bodyPointer(strings.TrimPrefix(loc, "body"))
		// huma names the parent object; the message names the missing property.
		if requiredProp != "" {
			if ptr == "" {
				ptr = "/" + escapePointerToken(requiredProp)
			} else {
				ptr = ptr + "/" + escapePointerToken(requiredProp)
			}
		}
		fe.Pointer = &ptr
	default:
		if loc != "" {
			fe.Parameter = &loc
		} else {
			empty := ""
			fe.Pointer = &empty
		}
	}
	return fe
}

func mapReason(msg string) (reason string, params *FieldParams, requiredProp string) {
	if name, ok := parseRequiredProperty(msg); ok {
		return ReasonRequired, nil, name
	}
	if strings.HasPrefix(msg, "required ") && strings.HasSuffix(msg, " parameter is missing") {
		return ReasonRequired, nil, ""
	}
	if n, ok := parseAfter(msg, "expected array length <= "); ok {
		return ReasonTooManyItems, &FieldParams{MaxItems: intPtr(n)}, ""
	}
	if n, ok := parseAfter(msg, "expected array length >= "); ok {
		return ReasonTooFewItems, &FieldParams{MinItems: intPtr(n)}, ""
	}
	if n, ok := parseAfter(msg, "expected length <= "); ok {
		return ReasonTooLong, &FieldParams{MaxLength: intPtr(n)}, ""
	}
	if n, ok := parseAfter(msg, "expected length >= "); ok {
		return ReasonTooShort, &FieldParams{MinLength: intPtr(n)}, ""
	}
	if f, ok := parseFloatAfter(msg, "expected number >= "); ok {
		return ReasonOutOfRange, &FieldParams{Minimum: floatPtr(f)}, ""
	}
	if f, ok := parseFloatAfter(msg, "expected number > "); ok {
		return ReasonOutOfRange, &FieldParams{Minimum: floatPtr(f)}, ""
	}
	if f, ok := parseFloatAfter(msg, "expected number <= "); ok {
		return ReasonOutOfRange, &FieldParams{Maximum: floatPtr(f)}, ""
	}
	if f, ok := parseFloatAfter(msg, "expected number < "); ok {
		return ReasonOutOfRange, &FieldParams{Maximum: floatPtr(f)}, ""
	}
	if msg == validation.MsgExpectedArrayItemsUnique {
		return ReasonDuplicateItem, nil, ""
	}
	const oneOfPrefix = `expected value to be one of "`
	if strings.HasPrefix(msg, oneOfPrefix) && strings.HasSuffix(msg, `"`) {
		inner := strings.TrimSuffix(strings.TrimPrefix(msg, oneOfPrefix), `"`)
		allowed := strings.Split(inner, ", ")
		return ReasonUnknownValue, &FieldParams{Allowed: &allowed}, ""
	}
	if strings.HasPrefix(msg, "expected value to be ") && !strings.Contains(msg, "match") {
		v := strings.TrimPrefix(msg, "expected value to be ")
		allowed := []string{v}
		return ReasonUnknownValue, &FieldParams{Allowed: &allowed}, ""
	}
	return ReasonInvalidFormat, nil, ""
}

func parseRequiredProperty(msg string) (string, bool) {
	const prefix = "expected required property "
	const suffix = " to be present"
	if strings.HasPrefix(msg, prefix) && strings.HasSuffix(msg, suffix) {
		name := strings.TrimSuffix(strings.TrimPrefix(msg, prefix), suffix)
		if name != "" {
			return name, true
		}
	}
	const depPrefix = "expected property "
	const depMid = " to be present when "
	if strings.HasPrefix(msg, depPrefix) {
		rest := strings.TrimPrefix(msg, depPrefix)
		name, after, ok := strings.Cut(rest, depMid)
		if ok && name != "" && strings.HasSuffix(after, " is present") {
			return name, true
		}
	}
	return "", false
}

func parseAfter(msg, prefix string) (int, bool) {
	if !strings.HasPrefix(msg, prefix) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(msg, prefix))
	return n, err == nil
}

func parseFloatAfter(msg, prefix string) (float64, bool) {
	if !strings.HasPrefix(msg, prefix) {
		return 0, false
	}
	f, err := strconv.ParseFloat(strings.TrimPrefix(msg, prefix), 64)
	return f, err == nil
}

func bodyPointer(loc string) string {
	loc = strings.TrimPrefix(loc, ".")
	loc = bodyIndex.ReplaceAllString(loc, "/$1")
	loc = strings.ReplaceAll(loc, ".", "/")
	if loc == "" {
		return ""
	}
	if !strings.HasPrefix(loc, "/") {
		loc = "/" + loc
	}
	return loc
}

func escapePointerToken(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}

func intPtr(v int) *int { return &v }

func floatPtr(v float64) *float64 { return &v }
