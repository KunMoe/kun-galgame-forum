package apiv1

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
)

var errUnconfigured = errors.New("auth v1 is not configured")

var rankedRoles = []string{"creator", "moderator", "admin", "ren"}

var nsfwDisplays = []string{"hide", "blur", "show"}

type Users interface {
	Users(ctx context.Context, ids []int) (map[int]userclient.User, error)
}

type Service struct {
	users Users
	cdn   string
}

func New(users Users, cdn string) *Service {
	return &Service{users: users, cdn: cdn}
}

type Account struct {
	Object        string         `json:"object" enum:"account" maxLength:"7" doc:"Type discriminant. Always account."`
	ID            repr.DecimalID `json:"id" doc:"The caller's user id. JSON string of a decimal integer."`
	Name          *string        `json:"name" maxLength:"64" doc:"Display name from the account center's current record. null when the account no longer exists; show a localized label. Free text; never use it as a decision input."`
	Avatar        *repr.Image    `json:"avatar" doc:"Avatar from the account center's current record. null when the account has no image-service hash."`
	Roles         []AccountRole  `json:"roles" maxItems:"4" doc:"Ranked roles this credential carries, lowest rank first. A Bearer request never carries moderator, admin or ren. Other account roles are not listed."`
	ContentStance *ContentStance `json:"content_stance" doc:"The adult-content stance this web session carries. null for a Bearer request, which carries none: read it from the account center."`
}

type ContentStance struct {
	IsAdultConfirmed bool   `json:"is_adult_confirmed" doc:"Whether the account has confirmed it is an adult."`
	NsfwDisplay      string `json:"nsfw_display" enum:"hide,blur,show" maxLength:"4" doc:"How adult content is displayed for this account."`
}

type AccountRole string

func (AccountRole) Schema(huma.Registry) *huma.Schema {
	n := 9
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        []any{"creator", "moderator", "admin", "ren"},
		MaxLength:   &n,
		Description: "A ranked role carried by the credential.",
	}
}

type accountOutput struct {
	Body Account
}

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getMyAccount",
			Method:      http.MethodGet,
			Path:        "/me/account",
			Summary:     "Get the account behind this credential",
			Description: "Who the caller is: name and avatar from the account center's current record, roles and the adult-content stance as this credential carries them. " +
				"A Bearer request and a web session of the same person can therefore differ: Bearer carries no staff role and no stance.",
			Tags: []string{"me"},
			Responses: problemResponses(map[int]string{
				503: "SERVICE_UNAVAILABLE when the account center cannot be reached.",
			}),
		}), s.getMyAccount)
	}
}

func (s *Service) getMyAccount(ctx context.Context, _ *struct{}) (*accountOutput, error) {
	if s == nil || s.users == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	u := v1.User(ctx)
	if u == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	found, err := s.users.Users(ctx, []int{u.ID})
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	ref := repr.DeletedUserRef(u.ID)
	if record, ok := found[u.ID]; ok {
		ref = repr.NewUserRef(s.cdn, record)
	}
	roles := make([]AccountRole, 0, len(rankedRoles))
	for _, r := range rankedRoles {
		if slices.Contains(u.Roles, r) {
			roles = append(roles, AccountRole(r))
		}
	}
	var stance *ContentStance
	if !u.ViaBearer() {
		display := u.NSFWDisplay
		if !slices.Contains(nsfwDisplays, display) {
			display = "hide"
		}
		stance = &ContentStance{IsAdultConfirmed: u.AdultConfirmed, NsfwDisplay: display}
	}
	return &accountOutput{Body: Account{
		Object:        "account",
		ID:            repr.ID(u.ID),
		Name:          ref.Name,
		Avatar:        ref.Avatar,
		Roles:         roles,
		ContentStance: stance,
	}}, nil
}

func problemResponses(byStatus map[int]string) map[string]*huma.Response {
	out := make(map[string]*huma.Response, len(byStatus))
	for status, desc := range byStatus {
		out[strconv.Itoa(status)] = &huma.Response{
			Description: desc,
			Content: map[string]*huma.MediaType{
				problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
			},
		}
	}
	return out
}
