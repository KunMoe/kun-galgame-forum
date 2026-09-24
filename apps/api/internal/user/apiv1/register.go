package apiv1

import (
	"context"
	"net/http"

	adminService "kun-galgame-api/internal/admin/service"
	v1 "kun-galgame-api/internal/apiv1"
	resourceapiv1 "kun-galgame-api/internal/galgame/resourceapiv1"
	galgameService "kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/internal/user/service"
	wallapiv1 "kun-galgame-api/internal/wall/apiv1"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
	"github.com/redis/go-redis/v9"
)

type Users struct {
	users       *service.UserService
	creators    *galgameService.CreatorService
	oauth       *oauth.Client
	accounts    *userclient.Client
	content     *repository.UserContentRepository
	works       *workrepr.Hydrator
	resources   *resourceapiv1.Service
	community   *communityclient.Client
	wall        *wallapiv1.Service
	contributed func(context.Context, int64) ([]int, error)
	redis       *redis.Client
	state       *repository.StateRepository
	purge       *adminService.PurgeService
	cdn         string
}

type Deps struct {
	Users       *service.UserService
	Creators    *galgameService.CreatorService
	OAuth       *oauth.Client
	Accounts    *userclient.Client
	Content     *repository.UserContentRepository
	Works       *workrepr.Hydrator
	Resources   *resourceapiv1.Service
	Community   *communityclient.Client
	Wall        *wallapiv1.Service
	Contributed func(context.Context, int64) ([]int, error)
	Redis       *redis.Client
	State       *repository.StateRepository
	Purge       *adminService.PurgeService
	CDN         string
}

func New(d Deps) *Users {
	return &Users{
		users:       d.Users,
		creators:    d.Creators,
		oauth:       d.OAuth,
		accounts:    d.Accounts,
		content:     d.Content,
		works:       d.Works,
		resources:   d.Resources,
		community:   d.Community,
		wall:        d.Wall,
		contributed: d.Contributed,
		redis:       d.Redis,
		state:       d.State,
		purge:       d.Purge,
		cdn:         d.CDN,
	}
}

func (s *Users) readyUsers() *problem.Problem {
	if s == nil || s.users == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Users) readyAccounts() *problem.Problem {
	if s == nil || s.accounts == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Users) readyOAuth() *problem.Problem {
	if s == nil || s.oauth == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Users) readyCreators() *problem.Problem {
	if s == nil || s.creators == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func Register(u *Users) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getMe",
			Method:      http.MethodGet,
			Path:        "/me",
			Summary:     "Get the caller's forum status",
			Description: "Returns the caller's cached moemoepoint, today's check-in gate, unread-message flag, creator flag, and today's toolset upload bytes.",
			Tags:        []string{"users"},
		}), u.getMe)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getNotificationPreferences",
			Method:      http.MethodGet,
			Path:        "/me/notification-preferences",
			Summary:     "Get the caller's muted notification types",
			Description: "Returns the notification types the caller has muted. Unknown stored values are dropped. An account with no forum state returns an empty list.",
			Tags:        []string{"users"},
		}), u.getNotificationPreferences)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "putNotificationPreferences",
			Method:      http.MethodPut,
			Path:        "/me/notification-preferences",
			Summary:     "Replace the caller's muted notification types",
			Description: "Replaces the notification types the caller has muted. Unknown tokens are refused. The stored values are the forum's database keys; the response uses the same v1 tokens as the request.",
			Tags:        []string{"users"},
			Responses: problemResponses(map[int]string{
				422: "VALIDATION_FAILED when muted_types holds an unknown token or a duplicate.",
			}),
		}), u.putNotificationPreferences)

		huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
			OperationID: "createCheckIn",
			Method:      http.MethodPost,
			Path:        "/me/check-ins",
			Summary:     "Check in for today",
			Description: "Records today's Asia/Shanghai check-in and credits a deterministic moemoepoint reward. " +
				"Checking in again on the same Beijing day is ALREADY_EXISTS.",
			Tags: []string{"users"},
			Responses: problemResponses(map[int]string{
				409: "ALREADY_EXISTS when the caller has already checked in on the current Asia/Shanghai day.",
				503: "SERVICE_UNAVAILABLE when the moemoepoint award cannot be written.",
			}),
		})), u.createCheckIn)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "listMoemoepointEntries",
			Method:      http.MethodGet,
			Path:        "/me/moemoepoint-entries",
			Summary:     "List the caller's moemoepoint ledger",
			Description: "Lists the caller's moemoepoint ledger as a cursor page, newest first. " +
				"limit is 1–50 because that is the upstream page cap. " +
				"The cursor is bound to the caller and the reason filter.",
			Tags: []string{"users"},
			Responses: problemResponses(map[int]string{
				400: "LIMIT_TOO_LARGE when limit is greater than 50. INVALID_CURSOR when the cursor was issued for a different reason. INVALID_PARAMETER when reason is not a lowercase token. An unknown but well-formed reason is an empty page, not an error: the ledger filters by it and never rejects it.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), u.listMoemoepointEntries)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getPreferences",
			Method:      http.MethodGet,
			Path:        "/me/preferences",
			Summary:     "Get the caller's cloud preferences",
			Description: "Returns this site's cloud preference document. The namespace is this site's OAuth client id and cannot be chosen by the caller.",
			Tags:        []string{"users"},
			Middlewares: huma.Middlewares{withUpstream},
			Responses: problemResponses(map[int]string{
				403: "SCOPE_REQUIRED when the token lacks the preferences scope.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), u.getPreferences)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "putPreferences",
			Method:      http.MethodPut,
			Path:        "/me/preferences",
			Summary:     "Replace the caller's cloud preferences",
			Description: "Replaces this site's cloud preference document. If-Match is forwarded as a version check.",
			Tags:        []string{"users"},
			Middlewares: huma.Middlewares{withUpstream},
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when If-Match is not a document version.",
				403: "SCOPE_REQUIRED when the token lacks the preferences scope.",
				412: "PRECONDITION_FAILED when If-Match does not match the current version.",
				422: "VALIDATION_FAILED when the document exceeds 64 KiB after compaction.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), u.putPreferences)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "putNsfwDisplay",
			Method:      http.MethodPut,
			Path:        "/me/nsfw-display",
			Summary:     "Set the caller's adult-content display",
			Description: "Sets how adult content is shown for the caller and writes the new stance into the cookie session when one is present.",
			Tags:        []string{"users"},
			Middlewares: huma.Middlewares{withUpstream},
			Responses: problemResponses(map[int]string{
				403: "SCOPE_REQUIRED when the account has not granted this site the preferences scope.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), u.putNsfwDisplay)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "listUsers",
			Method:      http.MethodGet,
			Path:        "/users",
			Summary:     "Search or resolve users",
			Description: "Returns users whose names match q, or the users named in ids. Exactly one of q or ids is required. " +
				"It is not paginated. Banned accounts are omitted from q results and listed in missing for ids. " +
				"limit is 1–20 and applies to q.",
			Tags:        []string{"users"},
			Middlewares: huma.Middlewares{withUpstream},
			Responses: problemResponses(map[int]string{
				400: "LIMIT_TOO_LARGE when limit is greater than 20. INVALID_PARAMETER when ids holds more than 100 values, q is longer than 50 characters, or the account service refuses q.",
				422: "VALIDATION_FAILED when neither q nor ids is given, both are given, q is only whitespace, or an id is not a positive integer.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), u.listUsers)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "getUser",
			Method:      http.MethodGet,
			Path:        "/users/{user_id}",
			Summary:     "Get a user's public profile",
			Description: "Returns one user's public profile and activity counts. " +
				"NOT_FOUND when the account does not exist or is not renderable. " +
				"topic_count excludes hidden topics. topic_today_count uses the Asia/Shanghai calendar day. " +
				"community_comment_count is null when the community service is unavailable and there is no cached value.",
			Tags: []string{"users"},
			Responses: problemResponses(map[int]string{
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), u.getUser)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "patchMyProfile",
			Method:      http.MethodPatch,
			Path:        "/me/profile",
			Summary:     "Update the caller's name or bio",
			Description: "Changes the fields present in the body. At least one of name or bio is required.",
			Tags:        []string{"users"},
			Middlewares: huma.Middlewares{withUpstream},
			Responses: problemResponses(map[int]string{
				403: "MOEMOEPOINT_INSUFFICIENT when a name change costs more than the live balance.",
				409: "USERNAME_TAKEN when the name is already in use.",
				422: "VALIDATION_FAILED when both fields are absent, or the name fails the upstream character whitelist.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), u.patchMyProfile)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "putMyAvatar",
			Method:      http.MethodPut,
			Path:        "/me/avatar",
			Summary:     "Replace the caller's avatar",
			Description: "Uploads a new avatar image as multipart field file. The image must be image/* and at most 4 MiB.",
			Tags:        []string{"users"},
			Middlewares: huma.Middlewares{withUpstream},
			RequestBody: avatarRequestBody(),
			Responses: problemResponses(map[int]string{
				415: "UNSUPPORTED_MEDIA_TYPE when file is not an image.",
				422: "VALIDATION_FAILED when file is missing or larger than 4 MiB.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), u.putMyAvatar)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getCreatorStatus",
			Method:      http.MethodGet,
			Path:        "/me/creator-status",
			Summary:     "Get the caller's creator eligibility",
			Description: "Returns whether the caller is a creator, the live eligibility snapshot, and their latest application if any.",
			Tags:        []string{"users"},
			Middlewares: huma.Middlewares{withUpstream},
			Responses: problemResponses(map[int]string{
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), u.getCreatorStatus)

		huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
			OperationID:   "createCreatorApplication",
			Method:        http.MethodPost,
			Path:          "/me/creator-applications",
			Summary:       "Apply for the creator role",
			DefaultStatus: http.StatusCreated,
			Description:   "Submits a creator-role application. Location points at the caller's creator-status resource.",
			Tags:          []string{"users"},
			Middlewares:   huma.Middlewares{withUpstream},
			Responses: problemResponses(map[int]string{
				403: "CREATOR_INELIGIBLE when the caller does not meet any eligibility threshold.",
				409: "ALREADY_EXISTS when a pending application already exists. INVALID_STATE_TRANSITION when the caller is already a creator. CREATOR_APPLICATION_COOLDOWN when a declined application is still cooling down.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		})), u.createCreatorApplication)

		u.registerLists(api)
		u.registerAdminPurge(api)
	}
}
