package calendarapiv1

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

type myMonthInput struct {
	Month       string `query:"month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$" maxLength:"7" doc:"Calendar month as YYYY-MM. Omitted means the current month in Asia/Tokyo."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false."`
	workrepr.TagExclusionInput
}

type myPendingInput struct {
	Year        int  `query:"year" minimum:"0" maximum:"9999" doc:"Calendar year. 0 or omitted means the current year in Asia/Tokyo."`
	IncludeNSFW bool `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false."`
	workrepr.TagExclusionInput
}

type myNSFWInput struct {
	IncludeNSFW bool `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false."`
	workrepr.TagExclusionInput
}

type myCalendarClient interface {
	CatalogMyCalendar(ctx context.Context, accessToken, bucket string, q url.Values) (*client.CatalogWorksPage, error)
}

func (s *Service) myCalendar(token string) calendarSource {
	return func(ctx context.Context, bucket string, q url.Values) (*client.CatalogWorksPage, error) {
		cli, ok := s.catalog.(myCalendarClient)
		if !ok || cli == nil {
			return nil, problem.Internal(errUnconfigured)
		}
		page, err := cli.CatalogMyCalendar(ctx, token, bucket, q)
		if err != nil {
			return nil, galgameapiv1.MapUserPlane(err)
		}
		return page, nil
	}
}

func myBucketQuery(includeNSFW bool) url.Values {
	q := url.Values{
		"limit":         {strconv.Itoa(calendarPageLimit)},
		"include":       {workrepr.RowInclude},
		"include_total": {"true"},
	}
	client.ApplyWorksGate(q, !includeNSFW)
	return q
}

func myMonthQuery(month string, includeNSFW bool) url.Values {
	q := myBucketQuery(includeNSFW)
	if month != "" {
		q.Set("month", month)
	}
	return q
}

func (s *Service) listMyReleaseCalendarMonth(ctx context.Context, in *myMonthInput) (*monthOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	token, p := galgameapiv1.RequireToken(ctx)
	if p != nil {
		return nil, p
	}
	now := s.clock()
	month := in.Month
	if month == "" {
		month = now.Format("2006-01")
	}
	return s.renderMonth(ctx, s.myCalendar(token), now, month, in.Exclusion().Apply(myMonthQuery(month, in.IncludeNSFW)))
}

func (s *Service) listMyReleaseCalendarPending(ctx context.Context, in *myPendingInput) (*pendingOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	token, p := galgameapiv1.RequireToken(ctx)
	if p != nil {
		return nil, p
	}
	now := s.clock()
	year := in.Year
	if year == 0 {
		year = now.Year()
	}
	q := in.Exclusion().Apply(myBucketQuery(in.IncludeNSFW))
	q.Set("year", strconv.Itoa(year))
	q.Set("precision", "year")
	return s.renderPending(ctx, s.myCalendar(token), year, q)
}

func (s *Service) listMyReleaseCalendarTBA(ctx context.Context, in *myNSFWInput) (*tbaOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	token, p := galgameapiv1.RequireToken(ctx)
	if p != nil {
		return nil, p
	}
	q := in.Exclusion().Apply(myBucketQuery(in.IncludeNSFW))
	q.Set("status", "unknown")
	return s.renderTBA(ctx, s.myCalendar(token), q)
}

func (s *Service) listMyReleaseCalendarUpcoming(ctx context.Context, in *myNSFWInput) (*upcomingOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	token, p := galgameapiv1.RequireToken(ctx)
	if p != nil {
		return nil, p
	}
	return s.renderUpcoming(ctx, s.myCalendar(token), func(month string) url.Values {
		return in.Exclusion().Apply(myMonthQuery(month, in.IncludeNSFW))
	})
}

func registerMyCalendar(api huma.API, svc *Service) {
	tag := []string{"me"}
	auth := problemResponses(map[int]string{
		401: "MISSING_CREDENTIAL when the request has no credentials; INVALID_CREDENTIAL when the session has no OAuth access token or catalog rejects it.",
		403: "SCOPE_REQUIRED when the token lacks a scope catalog demands; ACCOUNT_BANNED.",
	})
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyReleaseCalendarPending",
		Method:      http.MethodGet,
		Path:        "/me/release-calendar/pending",
		Summary:     "List the caller's pending releases in a year",
		Description: "Works of companies the caller follows whose release date is known only to year precision. The walk follows catalog's cursor, at most 2,000 works. Every original language is included, and works credited only to doujin circles are included.",
		Tags:        tag,
		Middlewares: huma.Middlewares{galgameapiv1.WithAccessToken},
		Responses: mergeResponses(auth, problemResponses(map[int]string{
			400: "INVALID_PARAMETER when year is malformed.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached, including a failure mid-walk.",
		})),
	}), svc.listMyReleaseCalendarPending)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyReleaseCalendarTBA",
		Method:      http.MethodGet,
		Path:        "/me/release-calendar/tba",
		Summary:     "List the caller's works with an unknown release date",
		Description: "Works of companies the caller follows that catalog files as status unknown. The walk follows catalog's cursor, at most 2,000 works. Every original language is included, and works credited only to doujin circles are included.",
		Tags:        tag,
		Middlewares: huma.Middlewares{galgameapiv1.WithAccessToken},
		Responses: mergeResponses(auth, problemResponses(map[int]string{
			400: "INVALID_PARAMETER when include_nsfw is not a boolean.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached, including a failure mid-walk.",
		})),
	}), svc.listMyReleaseCalendarTBA)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyReleaseCalendarUpcoming",
		Method:      http.MethodGet,
		Path:        "/me/release-calendar/upcoming",
		Summary:     "List the caller's upcoming releases by month",
		Description: "Walks from the current Asia/Tokyo month to catalog's max_month, at most 24 months, limited to works of companies the caller follows. A failed month is SERVICE_UNAVAILABLE for the whole response. Empty months are omitted. Every original language is included, and works credited only to doujin circles are included.",
		Tags:        tag,
		Middlewares: huma.Middlewares{galgameapiv1.WithAccessToken},
		Responses: mergeResponses(auth, problemResponses(map[int]string{
			400: "INVALID_PARAMETER when include_nsfw is not a boolean.",
			503: "SERVICE_UNAVAILABLE when any month's catalog walk fails.",
		})),
	}), svc.listMyReleaseCalendarUpcoming)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyReleaseCalendarMonth",
		Method:      http.MethodGet,
		Path:        "/me/release-calendar",
		Summary:     "List the caller's releases in a month",
		Description: "Releases in a month from companies the caller follows. The walk follows catalog's cursor, at most 2,000 works. is_truncated is true when the cap is hit. An id catalog does not render is dropped with a warning. Every original language is included, and works credited only to doujin circles are included.",
		Tags:        tag,
		Middlewares: huma.Middlewares{galgameapiv1.WithAccessToken},
		Responses: mergeResponses(auth, problemResponses(map[int]string{
			400: "INVALID_PARAMETER when month is malformed.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached, including a failure mid-walk.",
		})),
	}), svc.listMyReleaseCalendarMonth)
}

func mergeResponses(parts ...map[string]*huma.Response) map[string]*huma.Response {
	out := map[string]*huma.Response{}
	for _, part := range parts {
		for status, resp := range part {
			out[status] = resp
		}
	}
	return out
}
