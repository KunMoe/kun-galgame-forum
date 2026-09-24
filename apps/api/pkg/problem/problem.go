package problem

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
)

const ContentType = "application/problem+json"

const (
	internalDetail    = "An internal error occurred."
	unavailableDetail = "A dependency is unavailable. Retry the request."
)

type FieldParams struct {
	MaxLength *int      `json:"max_length,omitempty" minimum:"0" doc:"Maximum string length the value exceeded."`
	MinLength *int      `json:"min_length,omitempty" minimum:"0" doc:"Minimum string length the value failed."`
	Minimum   *float64  `json:"minimum,omitempty" minimum:"-9007199254740991" maximum:"9007199254740991" doc:"Inclusive numeric lower bound the value missed."`
	Maximum   *float64  `json:"maximum,omitempty" minimum:"-9007199254740991" maximum:"9007199254740991" doc:"Inclusive numeric upper bound the value missed."`
	MaxItems  *int      `json:"max_items,omitempty" minimum:"0" doc:"Maximum array length the value exceeded."`
	MinItems  *int      `json:"min_items,omitempty" minimum:"0" doc:"Minimum array length the value failed."`
	Allowed   *[]string `json:"allowed,omitempty" maxItems:"256" maxLength:"128" pattern:"^[\\x20-\\x7E]+$" doc:"Closed vocabulary members that were expected."`
}

type FieldError struct {
	Pointer   *string      `json:"pointer,omitempty" maxLength:"512" pattern:"^(/([^/~]|~[01])*)*$" doc:"JSON Pointer to a body location. Exactly one of pointer, parameter, or header is present."`
	Parameter *string      `json:"parameter,omitempty" maxLength:"128" pattern:"^[A-Za-z0-9._\\[\\]-]+$" doc:"Query or path parameter name. Exactly one of pointer, parameter, or header is present."`
	Header    *string      `json:"header,omitempty" maxLength:"128" pattern:"^[A-Za-z0-9-]+$" doc:"Header name. Exactly one of pointer, parameter, or header is present."`
	Reason    string       `json:"reason" pattern:"^[A-Z][A-Z0-9_]*[A-Z0-9]$" minLength:"2" maxLength:"63" doc:"Field-level reason. UPPER_SNAKE. Disjoint from top-level codes."`
	Detail    string       `json:"detail" maxLength:"4096" doc:"English diagnostic for this location. Free text; never use it as a decision input."`
	Params    *FieldParams `json:"params,omitempty" doc:"Reason-specific bounds or allowed values. Omitted when the reason has no params."`
}

type Problem struct {
	Type      string       `json:"type" format:"uri" maxLength:"256" pattern:"^https://developer\\.nextmoe\\.dev/problems/[a-z]+/[a-z0-9-]+$" doc:"Problem type URI. The last path segment is the kebab-case form of code."`
	Title     string       `json:"title" maxLength:"128" pattern:"^[ -~]+$" doc:"Stable English phrase for this type. Does not vary per request."`
	Status    int          `json:"status" minimum:"400" maximum:"599" doc:"HTTP status this code is bound to. One status per code."`
	Detail    string       `json:"detail" maxLength:"4096" doc:"English diagnostic for this request. Free text; never use it as a decision input."`
	Instance  string       `json:"instance" maxLength:"4096" pattern:"^/\\S*$" doc:"Request path and query that failed."`
	Code      string       `json:"code" pattern:"^[A-Z][A-Z0-9_]*[A-Z0-9]$" minLength:"2" maxLength:"63" doc:"Top-level error code. UPPER_SNAKE."`
	RequestID string       `json:"request_id" pattern:"^req_[0-9A-HJKMNP-TV-Z]{26}$" minLength:"30" maxLength:"30" doc:"Request correlation id. Echoed from X-Request-ID when valid."`
	Errors    []FieldError `json:"errors" doc:"Field-level errors. Empty array, never null."`
	extra     map[string]any
	cause     error
}

var (
	_ huma.StatusError       = (*Problem)(nil)
	_ huma.ContentTypeFilter = (*Problem)(nil)
)

func (p *Problem) Error() string {
	if p == nil {
		return ""
	}
	if p.Detail != "" {
		return p.Detail
	}
	return p.Title
}

func (p *Problem) GetStatus() int {
	if p == nil {
		return http.StatusInternalServerError
	}
	return p.Status
}

func (p *Problem) ContentType(ct string) string {
	if ct == "application/json" || ct == "" {
		return ContentType
	}
	if ct == "application/cbor" {
		return "application/problem+cbor"
	}
	return ct
}

func New(code, detail string, fields ...FieldError) *Problem {
	def, ok := Lookup(code)
	if !ok {
		def, _ = Lookup(CodeInternalError)
	}
	if fields == nil {
		fields = []FieldError{}
	}
	return &Problem{
		Type:   def.TypeURI(),
		Title:  def.Title,
		Status: def.Status,
		Detail: detail,
		Code:   def.Code,
		Errors: fields,
	}
}

// Internal builds INTERNAL_ERROR. The wrapped text must not appear in detail;
// Write logs the cause with request_id.
func Internal(err error) *Problem {
	p := New(CodeInternalError, internalDetail)
	p.cause = err
	return p
}

func Unavailable(err error) *Problem {
	p := New(CodeServiceUnavailable, unavailableDetail)
	p.cause = err
	return p
}

// An upstream 429 is an expected quota state (g-plan ②), but it logged ERROR
// until 2026-09-24. A cause marks itself with Throttled, so this package does
// not import the clients.
func LogCause(p *Problem) {
	if p == nil || p.cause == nil {
		return
	}
	var t interface{ Throttled() bool }
	if errors.As(p.cause, &t) && t.Throttled() {
		slog.Warn("problem cause", "code", p.Code, "request_id", p.RequestID, "err", p.cause)
		return
	}
	slog.Error("problem cause", "code", p.Code, "request_id", p.RequestID, "err", p.cause)
}

func AtPointer(pointer, reason, detail string, params *FieldParams) FieldError {
	return FieldError{Pointer: &pointer, Reason: reason, Detail: detail, Params: params}
}

func AtParameter(name, reason, detail string, params *FieldParams) FieldError {
	return FieldError{Parameter: &name, Reason: reason, Detail: detail, Params: params}
}

func AtHeader(name, reason, detail string, params *FieldParams) FieldError {
	return FieldError{Header: &name, Reason: reason, Detail: detail, Params: params}
}

func (p *Problem) SetExtension(name string, value any) {
	if p == nil {
		panic("problem: SetExtension on nil Problem")
	}
	def, ok := Lookup(p.Code)
	if !ok {
		panic("problem: SetExtension on unknown code " + p.Code)
	}
	allowed := false
	for _, e := range def.Extensions {
		if e.Name == name {
			allowed = true
			break
		}
	}
	if !allowed {
		panic("problem: undeclared extension " + name + " on " + p.Code)
	}
	if p.extra == nil {
		p.extra = map[string]any{}
	}
	p.extra[name] = value
}

func (p *Problem) MarshalJSON() ([]byte, error) {
	type wire struct {
		Type      string       `json:"type"`
		Title     string       `json:"title"`
		Status    int          `json:"status"`
		Detail    string       `json:"detail"`
		Instance  string       `json:"instance"`
		Code      string       `json:"code"`
		RequestID string       `json:"request_id"`
		Errors    []FieldError `json:"errors"`
	}
	w := wire{
		Type:      p.Type,
		Title:     p.Title,
		Status:    p.Status,
		Detail:    p.Detail,
		Instance:  p.Instance,
		Code:      p.Code,
		RequestID: p.RequestID,
		Errors:    p.Errors,
	}
	if w.Errors == nil {
		w.Errors = []FieldError{}
	}
	raw, err := json.Marshal(w)
	if err != nil || len(p.extra) == 0 {
		return raw, err
	}
	obj := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	for k, v := range p.extra {
		ev, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		obj[k] = ev
	}
	return json.Marshal(obj)
}

func Write(c fiber.Ctx, p *Problem) error {
	if p == nil {
		p = New(CodeInternalError, internalDetail)
	}
	if p.Errors == nil {
		p.Errors = []FieldError{}
	}
	p.RequestID = RequestID(c)
	p.Instance = Instance(c)
	LogCause(p)
	c.Set(HeaderRequestID, p.RequestID)
	c.Set("Cache-Control", "no-store")
	if p.Status == http.StatusUnauthorized {
		c.Set("WWW-Authenticate", `Bearer realm="kungal"`)
	}
	// c.JSON overwrites Content-Type; pass it as the second argument.
	return c.Status(p.Status).JSON(p, ContentType)
}

func Instance(c fiber.Ctx) string {
	path := c.Path()
	if raw := string(c.Request().URI().QueryString()); raw != "" {
		return path + "?" + raw
	}
	return path
}
