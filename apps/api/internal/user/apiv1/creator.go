package apiv1

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	galgameService "kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type getCreatorStatusOutput struct {
	Body CreatorStatus
}

type createCreatorApplicationInput struct {
	Body createCreatorApplicationBody
}

type createCreatorApplicationBody struct {
	Statement string `json:"statement" required:"false" maxLength:"500" doc:"Statement for the reviewers. Empty string is allowed. Free text; never use it as a decision input."`
}

type createCreatorApplicationOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the caller's creator status, /api/v1/me/creator-status."`
	Body     CreatorApplication
}

func (s *Users) getCreatorStatus(ctx context.Context, _ *struct{}) (*getCreatorStatusOutput, error) {
	if prob := s.readyCreators(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	elig, app, isCreator, err := s.creators.Status(ctx, user.ID, accessToken(ctx))
	if err != nil {
		if errors.Is(err, galgameService.ErrAccountUnavailable) {
			return nil, unavailable(err)
		}
		return nil, problem.Internal(err)
	}
	mapped, err := mapCreatorApplication(app)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &getCreatorStatusOutput{Body: CreatorStatus{
		Object:      "creator_status",
		IsCreator:   isCreator,
		Eligibility: mapEligibility(elig),
		Application: mapped,
	}}, nil
}

func (s *Users) createCreatorApplication(ctx context.Context, in *createCreatorApplicationInput) (*createCreatorApplicationOutput, error) {
	if prob := s.readyCreators(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	app, err := s.creators.Apply(ctx, user.ID, accessToken(ctx), in.Body.Statement)
	if err != nil {
		if errors.Is(err, galgameService.ErrCreatorIneligible) {
			return nil, problem.New(problem.CodeCreatorIneligible,
				"The caller does not meet the conditions to apply for the creator role.")
		}
		if userclientCode(err) != 0 {
			return nil, mapCreatorApplyError(err)
		}
		if errors.Is(err, galgameService.ErrAccountUnavailable) {
			return nil, unavailable(err)
		}
		return nil, problem.Internal(err)
	}
	mapped, err := mapCreatorApplication(app)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if mapped == nil {
		return nil, problem.Internal(fmt.Errorf("creator application missing from upstream"))
	}
	return &createCreatorApplicationOutput{
		Location: "/api/v1/me/creator-status",
		Body:     *mapped,
	}, nil
}

func mapEligibility(e *galgameService.CreatorEligibility) CreatorEligibility {
	if e == nil {
		return CreatorEligibility{}
	}
	return CreatorEligibility{
		IsEligible:                    e.Eligible,
		MergedPrCount:                 int(e.MergedPRs),
		PublishedGalgameCount:         int(e.GalgamesPublished),
		LongReviewCount:               int(e.Reviews100),
		Moemoepoint:                   int(e.Moemoepoint),
		RequiredMergedPrCount:         e.NeedMergedPRs,
		RequiredPublishedGalgameCount: e.NeedGalgames,
		RequiredLongReviewCount:       e.NeedReviews,
		RequiredMoemoepoint:           e.NeedMoemoepoint,
	}
}

func mapCreatorApplication(app *userclient.CreatorApplication) (*CreatorApplication, error) {
	if app == nil {
		return nil, nil
	}
	switch app.Status {
	case "pending", "approved", "declined":
	default:
		return nil, fmt.Errorf("unknown creator application state %q", app.Status)
	}
	created, err := parseDateTime(app.CreatedAt)
	if err != nil {
		return nil, err
	}
	var reviewed *repr.DateTime
	if app.ReviewedAt != nil && *app.ReviewedAt != "" {
		ts, perr := parseDateTime(*app.ReviewedAt)
		if perr != nil {
			return nil, perr
		}
		reviewed = &ts
	}
	var decline *string
	if app.DeclineReason != "" {
		r := app.DeclineReason
		decline = &r
	}
	return &CreatorApplication{
		Object:        "creator_application",
		ID:            repr.ID(app.ID),
		State:         app.Status,
		Statement:     app.Message,
		DeclineReason: decline,
		CreatedAt:     created,
		ReviewedAt:    reviewed,
	}, nil
}

func parseDateTime(raw string) (repr.DateTime, error) {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, raw)
	}
	if err != nil {
		return "", err
	}
	return repr.Timestamp(t), nil
}
