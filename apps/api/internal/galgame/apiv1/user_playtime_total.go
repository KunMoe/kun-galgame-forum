package apiv1

import (
	"context"
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

type getUserPlaytimeTotalInput struct {
	UserID      string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, playtime on adult works counts toward minutes. Default false."`
}

type UserPlaytimeTotal struct {
	Object  string `json:"object" enum:"user_playtime_total" maxLength:"19" doc:"Type discriminant. Always user_playtime_total."`
	Minutes *int64 `json:"minutes" minimum:"0" doc:"The user's playtime summed over every work they have one on, each work counted once at the most minutes any application reported. null when catalog does not report a total yet."`
}

type userPlaytimeTotalOutput struct {
	Body UserPlaytimeTotal
}

func registerUserPlaytimeTotal(api huma.API, s *Service) {
	huma.Register(api, v1.ProfileGated(v1.Optional(huma.Operation{
		OperationID: "getUserPlaytimeTotal",
		Method:      http.MethodGet,
		Path:        "/users/{user_id}/playtime-total",
		Summary:     "Get a user's total playtime",
		Description: "The total playtime a user has recorded across NextMoe, under the same privacy switch as listUserWorkRecords: " +
			"public unless the user turns play records off, and always readable by the user.",
		Tags:        []string{"users"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			401: "INVALID_CREDENTIAL for a bad Bearer.",
			403: "ACCOUNT_BANNED when the caller's account is banned. PLAY_RECORDS_PRIVATE when the user keeps their play records private and the caller is not that user.",
			404: "NOT_FOUND when the account does not exist or is not renderable.",
			503: "SERVICE_UNAVAILABLE when the catalog or the account service cannot be reached.",
		}),
	}), "user_id"), s.getUserPlaytimeTotal)
}

func (s *Service) getUserPlaytimeTotal(ctx context.Context, in *getUserPlaytimeTotalInput) (*userPlaytimeTotalOutput, error) {
	ownerID, ok := parseWorkID(in.UserID)
	if !ok {
		return nil, notFound()
	}
	if s.users == nil || s.catalog == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	_, renderable, p := s.lookupOwner(ctx, int64(ownerID))
	if p != nil {
		return nil, p
	}
	user := callerUser(ctx)
	isOwner := user != nil && user.ID == ownerID
	if !renderable && !isOwner {
		return nil, notFound()
	}
	token := ""
	if isOwner {
		if token, p = requireToken(ctx); p != nil {
			return nil, p
		}
	}
	page, err := s.catalog.WorkRecords(ctx, token, catalogclient.WorkRecordsQuery{
		UserID:        int64(ownerID),
		SFWOnly:       !in.IncludeNSFW,
		Limit:         1,
		Record:        "playtime",
		PlaytimeTotal: true,
	})
	if err != nil {
		return nil, mapWorkRecordsErr(err, isOwner)
	}
	return &userPlaytimeTotalOutput{Body: UserPlaytimeTotal{Object: "user_playtime_total", Minutes: page.PlaytimeTotalMinutes}}, nil
}
