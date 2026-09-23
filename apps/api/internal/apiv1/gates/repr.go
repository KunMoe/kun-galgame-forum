package gates

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

var g8Allow = map[string]bool{
	"children": true,
	"items":    true,
	"object":   true,
	"state":    true,
	"viewer":   true,
	// A published topic always has a category; a draft is a half-written topic
	// and 39 of the 69 on production have never had one chosen, so the draft's
	// field is the same three-member enum but nullable. G8 went red on the
	// drafts wave (2026-09-22) over exactly that difference. It is real and
	// permanent, not an oversight. Keep the enum members identical on both
	// sides — this entry stops the gate seeing the nullability, and it would
	// equally stop it seeing a genuinely different vocabulary.
	"category": true,
}

var g8Forbidden = []string{
	"kind", "gid", "tid", "uid", "rid", "pid", "cid",
	"created", "updated", "edited", "view", "status_update_time", "user",
}

var g6Banned = []string{"code", "message", "data", "success", "status", "timestamp", "error"}

var closedEnumValue = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// languageTagProperties hold lowercase BCP 47 tags, which contain hyphens: the
// named exception in 01 §3. Their values are checked for that shape instead.
var (
	languageTagProperties = map[string]bool{
		"language": true, "languages": true, "interface_language": true,
		"resource_language": true, "resource_languages": true,
	}
	languageTagValue = regexp.MustCompile(`^[a-z]{2,3}(-[a-z0-9]{2,8})*$`)
	// kebabCaseProperties hold lowercase kebab-case tokens with hyphens: the
	// named exception in 01 §3 (resource_runtimes). Checked for that shape instead.
	kebabCaseProperties = map[string]bool{
		"resource_runtimes": true,
	}
	kebabCaseValue = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
)

func languageTagEnumValue(v string) bool {
	return v == "other" || v == "others" || languageTagValue.MatchString(v)
}

func kebabCaseEnumValue(v string) bool {
	return v == "other" || kebabCaseValue.MatchString(v)
}

func CheckG6(doc *huma.OpenAPI) []string {
	var errs []string
	for path, item := range doc.Paths {
		for _, op := range pathOps(item) {
			for status, resp := range op.Responses {
				if !isSuccess(status) {
					continue
				}
				for mt, content := range resp.Content {
					s := deref(doc, content.Schema)
					if s == nil {
						continue
					}
					for _, k := range g6Banned {
						if _, ok := s.Properties[k]; ok {
							errs = append(errs, fmt.Sprintf("G6: %s %s %s %s has envelope key %s", opMethod(op), path, status, mt, k))
						}
					}
				}
			}
		}
	}
	return errs
}

func CheckG7(doc *huma.OpenAPI) []string {
	var errs []string
	walkNamed(doc, func(where, name string, s *huma.Schema, _ bool) {
		r := deref(doc, s)
		switch {
		case name == "id" || strings.HasSuffix(name, "_id") || name == "cursor" || name == "next_cursor":
			if r.Type != huma.TypeString {
				errs = append(errs, fmt.Sprintf("G7: %s.%s must be a JSON string, got %q", where, name, r.Type))
			}
		case strings.HasSuffix(name, "_ids"):
			if r.Type != huma.TypeArray || deref(doc, r.Items) == nil || deref(doc, r.Items).Type != huma.TypeString {
				errs = append(errs, fmt.Sprintf("G7: %s.%s must be an array of strings", where, name))
			}
		}
	})
	return errs
}

func CheckG8(doc *huma.OpenAPI) []string {
	type occ struct{ where, shape string }
	byName := map[string][]occ{}
	forbidden := map[string]bool{}
	for _, n := range g8Forbidden {
		forbidden[n] = true
	}
	var errs []string
	walkNamed(doc, func(where, name string, s *huma.Schema, param bool) {
		if forbidden[name] {
			errs = append(errs, fmt.Sprintf("G8: %s.%s is a forbidden name", where, name))
		}
		if !param {
			byName[name] = append(byName[name], occ{where + "." + name, shape(s)})
		}
	})
	for name, occs := range byName {
		if g8Allow[name] {
			continue
		}
		for _, o := range occs[1:] {
			if o.shape != occs[0].shape {
				errs = append(errs, fmt.Sprintf("G8: property %s has diverging schemas (%s is %s, %s is %s)", name, occs[0].where, occs[0].shape, o.where, o.shape))
				break
			}
		}
	}
	return errs
}

func shape(s *huma.Schema) string {
	if s.Ref != "" {
		return s.Ref
	}
	if len(s.AnyOf) > 0 {
		parts := make([]string, len(s.AnyOf))
		for i, x := range s.AnyOf {
			parts[i] = shape(x)
		}
		slices.Sort(parts)
		return "anyOf(" + strings.Join(parts, " | ") + ")"
	}
	key := s.Type
	if s.Format != "" {
		key += "/" + s.Format
	}
	if s.Nullable {
		key += "|null"
	}
	if len(s.Enum) > 0 {
		values := make([]string, len(s.Enum))
		for i, v := range s.Enum {
			values[i] = fmt.Sprint(v)
		}
		slices.Sort(values)
		key += "{" + strings.Join(values, ",") + "}"
	}
	if s.Items != nil {
		key += "[" + shape(s.Items) + "]"
	}
	return key
}

func CheckG9(doc *huma.OpenAPI) []string {
	var errs []string
	walkDoc(doc, func(where string, s *huma.Schema) {
		if s.Nullable && isArrayOrMap(s) {
			errs = append(errs, "G9: "+where+" array or map schema is nullable")
		}
		if ap, ok := s.AdditionalProperties.(bool); ok && !ap {
			errs = append(errs, "G9: "+where+" has additionalProperties: false")
		}
	})
	errs = append(errs, nullableObjectsInRequests(doc)...)
	errs = append(errs, checkOmitempty(apiRoot())...)
	return errs
}

func isArrayOrMap(s *huma.Schema) bool {
	if s.Type == huma.TypeArray || s.Items != nil {
		return true
	}
	_, isBool := s.AdditionalProperties.(bool)
	return s.AdditionalProperties != nil && !isBool
}

// huma validates a {"type": "null"} branch as matching anything, so a nullable
// object in a request body would switch off validation of that field.
func nullableObjectsInRequests(doc *huma.OpenAPI) []string {
	var errs []string
	seen := map[*huma.Schema]bool{}
	var walk func(where string, s *huma.Schema)
	walk = func(where string, s *huma.Schema) {
		s = deref(doc, s)
		if s == nil || seen[s] {
			return
		}
		seen[s] = true
		for _, x := range s.AnyOf {
			if x.Type == "null" {
				errs = append(errs, "G9: "+where+" is a nullable object in a request body; make the field omitempty")
			}
			walk(where, x)
		}
		for name, p := range s.Properties {
			walk(where+"."+name, p)
		}
		walk(where+"[]", s.Items)
	}
	for path, item := range doc.Paths {
		for _, op := range pathOps(item) {
			if op.RequestBody == nil {
				continue
			}
			for mt, c := range op.RequestBody.Content {
				walk(fmt.Sprintf("%s %s request %s", opMethod(op), path, mt), c.Schema)
			}
		}
	}
	return errs
}

func CheckG14(doc *huma.OpenAPI) []string {
	var errs []string
	walkDoc(doc, func(where string, s *huma.Schema) {
		if s.Type == huma.TypeString || (s.Type == "" && len(s.Enum) > 0) {
			if s.MaxLength == nil {
				errs = append(errs, "G14: "+where+" string has no maxLength")
			}
			if !stringClassed(s) {
				errs = append(errs, "G14: "+where+" string has no enum, format, pattern, or free-text marker")
			}
		}
		if (s.Type == huma.TypeInteger || s.Type == huma.TypeNumber) && s.Minimum == nil {
			errs = append(errs, "G14: "+where+" number has no minimum")
		}
	})
	return errs
}

func stringClassed(s *huma.Schema) bool {
	if len(s.Enum) > 0 || s.Format != "" || s.Pattern != "" {
		return true
	}
	return strings.Contains(s.Description, repr.FreeTextSentence)
}

func CheckG16(doc *huma.OpenAPI) []string {
	var errs []string
	walkNamed(doc, func(where, name string, s *huma.Schema, _ bool) {
		r := deref(doc, s)
		switch name {
		case "request_id":
			if r.Pattern != `^req_[0-9A-HJKMNP-TV-Z]{26}$` {
				errs = append(errs, fmt.Sprintf("G16: %s.request_id pattern %q", where, r.Pattern))
			}
		case "cursor", "next_cursor":
			if !strings.HasPrefix(r.Pattern, "^cur_") {
				errs = append(errs, fmt.Sprintf("G16: %s.%s pattern %q does not start with ^cur_", where, name, r.Pattern))
			}
		}
	})
	return errs
}

func CheckG17(doc *huma.OpenAPI) []string {
	var errs []string
	for path, item := range doc.Paths {
		for _, op := range []*huma.Operation{item.Put, item.Post, item.Patch, item.Delete} {
			if op == nil {
				continue
			}
			segs := strings.Split(path, "/")
			for i, seg := range segs {
				if !isIDParam(seg) {
					continue
				}
				if prefix := strings.Join(segs[:i+1], "/"); !readable(doc, prefix) {
					errs = append(errs, fmt.Sprintf("G17: %s %s addresses %s, but GET %s has no 200 object with an id", opMethod(op), path, seg, prefix))
				}
			}
		}
	}
	return errs
}

func isIDParam(seg string) bool {
	name, ok := strings.CutPrefix(seg, "{")
	if !ok {
		return false
	}
	name, ok = strings.CutSuffix(name, "}")
	return ok && (name == "id" || strings.HasSuffix(name, "_id"))
}

func readable(doc *huma.OpenAPI, path string) bool {
	item := doc.Paths[path]
	if item == nil || item.Get == nil || item.Get.Responses["200"] == nil {
		return false
	}
	for _, c := range item.Get.Responses["200"].Content {
		if s := deref(doc, c.Schema); s != nil && s.Properties["id"] != nil {
			return true
		}
	}
	return false
}

func CheckF1(doc *huma.OpenAPI) []string {
	var errs []string
	walkNamed(doc, func(where, name string, s *huma.Schema, param bool) {
		r := deref(doc, s)
		at := where + "." + name
		if r.Type == huma.TypeBoolean && !booleanName(name, param) {
			errs = append(errs, "F1: "+at+" boolean must start with is_, has_ or can_ (a query flag may also start with include_)")
		}
		if strings.HasSuffix(name, "_at") != (r.Format == "date-time") {
			errs = append(errs, "F1: "+at+" a name ending in _at and format date-time go together")
		}
		if strings.HasSuffix(name, "_date") != (r.Format == "date") {
			errs = append(errs, "F1: "+at+" a name ending in _date and format date go together")
		}
		if strings.HasSuffix(name, "_count") && (r.Type != huma.TypeInteger || r.Minimum == nil || *r.Minimum != 0) {
			errs = append(errs, "F1: "+at+" must be an integer with minimum 0")
		}
		if name == "sections" || name == "section" {
			return
		}
		if languageTagProperties[name] {
			for _, v := range enumValues(doc, r) {
				if !languageTagEnumValue(v) {
					errs = append(errs, fmt.Sprintf("F1: %s enum value %q is not a lowercase BCP 47 tag", at, v))
				}
			}
			return
		}
		if kebabCaseProperties[name] {
			for _, v := range enumValues(doc, r) {
				if !kebabCaseEnumValue(v) {
					errs = append(errs, fmt.Sprintf("F1: %s enum value %q is not lowercase kebab-case", at, v))
				}
			}
			return
		}
		for _, v := range enumValues(doc, r) {
			if !closedEnumValue.MatchString(v) {
				errs = append(errs, fmt.Sprintf("F1: %s enum value %q is not snake_case", at, v))
			}
		}
	})
	return errs
}

func CheckF9(doc *huma.OpenAPI) []string {
	var errs []string
	for path, item := range doc.Paths {
		for _, op := range pathOps(item) {
			accepts := slices.ContainsFunc(op.Parameters, func(p *huma.Param) bool {
				return p.In == "query" && p.Name == "include_total"
			})
			paged := slices.ContainsFunc(op.Parameters, func(p *huma.Param) bool {
				return p.In == "query" && p.Name == "page"
			})
			for status, resp := range op.Responses {
				if !isSuccess(status) {
					continue
				}
				for mt, content := range resp.Content {
					s := deref(doc, content.Schema)
					if s == nil {
						continue
					}
					at := fmt.Sprintf("%s %s %s %s", opMethod(op), path, status, mt)
					_, declares := s.Properties["total"]
					_, pageNumber := s.Properties["total_relation"]
					switch {
					case pageNumber && !paged:
						errs = append(errs, "F9: "+at+" is a page-number collection but the operation has no page parameter")
					case pageNumber && accepts:
						errs = append(errs, "F9: "+at+" is a page-number collection, which always sends total, but the operation accepts include_total")
					case pageNumber && !declares:
						errs = append(errs, "F9: "+at+" is a page-number collection but does not declare total")
					case pageNumber:
					case declares && !accepts:
						errs = append(errs, "F9: "+at+" declares total but the operation has no include_total parameter")
					case accepts && !declares:
						errs = append(errs, "F9: "+at+" does not declare total but the operation accepts include_total")
					}
				}
			}
		}
	}
	return errs
}

func booleanName(name string, param bool) bool {
	for _, p := range []string{"is_", "has_", "can_"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return param && strings.HasPrefix(name, "include_")
}

func enumValues(doc *huma.OpenAPI, s *huma.Schema) []string {
	if s.Type == huma.TypeArray {
		s = deref(doc, s.Items)
	}
	if s == nil || (s.Type != huma.TypeString && s.Type != "") {
		return nil
	}
	var values []string
	for _, v := range s.Enum {
		if v != nil {
			values = append(values, fmt.Sprint(v))
		}
	}
	return values
}

func walkNamed(doc *huma.OpenAPI, fn func(where, name string, s *huma.Schema, param bool)) {
	seen := map[*huma.Schema]bool{}
	var walk func(where string, s *huma.Schema)
	walk = func(where string, s *huma.Schema) {
		s = deref(doc, s)
		if s == nil || seen[s] {
			return
		}
		seen[s] = true
		for name, p := range s.Properties {
			if p == nil {
				continue
			}
			fn(where, name, p, false)
			walk(where+"."+name, p)
		}
		walk(where+"[]", s.Items)
		for i, x := range s.OneOf {
			walk(fmt.Sprintf("%s.oneOf[%d]", where, i), x)
		}
		for i, x := range s.AnyOf {
			walk(fmt.Sprintf("%s.anyOf[%d]", where, i), x)
		}
		for i, x := range s.AllOf {
			walk(fmt.Sprintf("%s.allOf[%d]", where, i), x)
		}
	}
	if doc.Components != nil && doc.Components.Schemas != nil {
		for name, schema := range doc.Components.Schemas.Map() {
			walk("schema "+name, schema)
		}
	}
	for path, item := range doc.Paths {
		for _, op := range pathOps(item) {
			for _, p := range op.Parameters {
				fn(fmt.Sprintf("%s %s param", opMethod(op), path), p.Name, p.Schema, true)
				walk(fmt.Sprintf("%s %s param %s", opMethod(op), path, p.Name), p.Schema)
			}
			if op.RequestBody != nil {
				for mt, c := range op.RequestBody.Content {
					walk(fmt.Sprintf("%s %s request %s", opMethod(op), path, mt), c.Schema)
				}
			}
			for status, resp := range op.Responses {
				for mt, c := range resp.Content {
					walk(fmt.Sprintf("%s %s %s %s", opMethod(op), path, status, mt), c.Schema)
				}
			}
		}
	}
}
