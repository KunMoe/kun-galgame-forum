package apiv1

import (
	"context"
	"net/http"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

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

const (
	tagWebsites = "websites"
	tagTaxonomy = "website-taxonomy"

	activeCheck = "The caller is checked against the account service's current record first: a banned account is ACCOUNT_BANNED, and a failure of that lookup is SERVICE_UNAVAILABLE with nothing written. "
	bearerNote  = "Requests authenticated with a Bearer token never carry website permissions. "
)

func Register(svc *Service) func(huma.API) {
	return func(api huma.API) {
		registerWebsites(api, svc)
		registerAdminWebsites(api, svc)
		registerCategories(api, svc)
		registerTags(api, svc)
		registerTagGroups(api, svc)
	}
}

func registerWebsites(api huma.API, svc *Service) {
	huma.Register(api, v1.Public(huma.Operation{
		OperationID: "listWebsites",
		Method:      http.MethodGet,
		Path:        "/websites",
		Summary:     "List the website directory",
		Description: "Lists websites as a cursor page, newest listing first, ties broken by descending id. There is one sort and no sort parameter. " +
			"NSFW sites are left out unless include_nsfw=true. website_category_id and website_tag_id narrow the list; an id that matches nothing gives an empty list. " +
			"The cursor is bound to include_nsfw, website_category_id and website_tag_id. include_total=true adds total under the same predicate as items.",
		Tags: []string{tagWebsites},
		Responses: problemResponses(map[int]string{
			400: "INVALID_CURSOR or LIMIT_TOO_LARGE.",
		}),
	}), svc.listWebsites)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "getWebsite",
		Method:      http.MethodGet,
		Path:        "/websites/{website_host}",
		Summary:     "Get a website by its host",
		Description: "Returns the website whose main host is website_host and counts one view. The host can be renamed; refer to a website by id. " +
			"An NSFW site is returned like any other; a client that hides NSFW content gates it. viewer is null for an anonymous caller.",
		Tags: []string{tagWebsites},
		Responses: problemResponses(map[int]string{
			404: "NOT_FOUND when no website has this host.",
		}),
	}), svc.getWebsite)

	registerSlot(api, "likeWebsite", http.MethodPut, "/websites/{website_host}/like", "Like a website",
		"Sets the caller's like. Liking again changes nothing and returns the same state.", svc.likeWebsite)
	registerSlot(api, "unlikeWebsite", http.MethodDelete, "/websites/{website_host}/like", "Remove a like",
		"Clears the caller's like. Removing a like that is not there changes nothing.", svc.unlikeWebsite)
	registerSlot(api, "favoriteWebsite", http.MethodPut, "/websites/{website_host}/favorite", "Favorite a website",
		"Sets the caller's favorite. Favoriting again changes nothing and returns the same state.", svc.favoriteWebsite)
	registerSlot(api, "unfavoriteWebsite", http.MethodDelete, "/websites/{website_host}/favorite", "Remove a favorite",
		"Clears the caller's favorite. Removing a favorite that is not there changes nothing.", svc.unfavoriteWebsite)
}

func registerSlot(api huma.API, id, method, path, summary, desc string, handler func(context.Context, *hostInput) (*engagementOutput, error)) {
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: id,
		Method:      method,
		Path:        path,
		Summary:     summary,
		Description: desc + " The counters move only when the caller's row is really added or removed. NSFW sites can be liked and favorited like any other. " +
			activeCheck + "Returns both counters and the caller's state after the request.",
		Tags: []string{tagWebsites},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			404: "NOT_FOUND when no website has this host.",
		}),
	}), handler)
}

func registerAdminWebsites(api huma.API, svc *Service) {
	huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
		OperationID:   "createAdminWebsite",
		Method:        http.MethodPost,
		Path:          "/admin/websites",
		Summary:       "List a new website",
		DefaultStatus: http.StatusCreated,
		Description: "Creates a website and returns its edit source. Needs website.create. " + bearerNote + activeCheck +
			"host is stored lower case. title needs a non-whitespace character; description needs 10 once trimmed. " +
			"website_category_id must exist (UNKNOWN_REFERENCE). website_tag_ids: no duplicates (DUPLICATE_ITEM), each must exist (UNKNOWN_REFERENCE), at most one per single-select group (INCONSISTENT_WITH). " +
			"urls must be http or https URLs of at most 100 characters (INVALID_FORMAT). language is stored lower case. " +
			"A host or title another website already uses is ALREADY_EXISTS with the field as the pointer. Location is the edit source's path.",
		Tags: []string{tagWebsites},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without website.create; ACCOUNT_BANNED.",
			409: "ALREADY_EXISTS when the host or title is taken; IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "VALIDATION_FAILED for a field outside its format, an unknown category or tag, a duplicate tag, or two tags of a single-select group.",
		}),
	})), svc.createAdminWebsite)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getAdminWebsite",
		Method:      http.MethodGet,
		Path:        "/admin/websites/{website_id}",
		Summary:     "Get a website's edit source",
		Description: "Returns the fields updateAdminWebsite takes, as stored. Needs website.edit. " + bearerNote + activeCheck,
		Tags:        []string{tagWebsites},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without website.edit; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the website does not exist.",
		}),
	}), svc.getAdminWebsite)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateAdminWebsite",
		Method:      http.MethodPatch,
		Path:        "/admin/websites/{website_id}",
		Summary:     "Edit a website",
		Description: "Changes the fields that are sent and returns the edit source. Needs website.edit, checked before the website is looked up. " + bearerNote + activeCheck +
			"Fields are checked as in createAdminWebsite. website_tag_ids and urls, when present, replace the whole set; absent keeps it. An empty object writes nothing.",
		Tags: []string{tagWebsites},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without website.edit; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the website does not exist.",
			409: "ALREADY_EXISTS when the new host or title is taken.",
			422: "VALIDATION_FAILED as in createAdminWebsite.",
		}),
	}), svc.updateAdminWebsite)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "deleteAdminWebsite",
		Method:        http.MethodDelete,
		Path:          "/admin/websites/{website_id}",
		Summary:       "Delete a website",
		DefaultStatus: http.StatusNoContent,
		Description:   "Deletes a website with its likes, favorites and tag links, and its home-feed card. Needs website.delete, checked before the website is looked up. " + bearerNote + activeCheck,
		Tags:          []string{tagWebsites},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without website.delete; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the website does not exist.",
		}),
	}), svc.deleteAdminWebsite)
}

func registerCategories(api huma.API, svc *Service) {
	huma.Register(api, v1.Public(huma.Operation{
		OperationID: "listWebsiteCategories",
		Method:      http.MethodGet,
		Path:        "/website-categories",
		Summary:     "List website categories",
		Description: "Lists categories as a cursor page by ascending sort_order, ties broken by ascending id. website_count counts NSFW sites too.",
		Tags:        []string{tagTaxonomy},
		Responses: problemResponses(map[int]string{
			400: "INVALID_CURSOR or LIMIT_TOO_LARGE.",
		}),
	}), svc.listWebsiteCategories)

	huma.Register(api, v1.Public(huma.Operation{
		OperationID: "getWebsiteCategory",
		Method:      http.MethodGet,
		Path:        "/website-categories/{website_category_slug}",
		Summary:     "Get a website category by its slug",
		Description: "Returns the category whose slug is website_category_slug. Its websites are listWebsites with website_category_id.",
		Tags:        []string{tagTaxonomy},
		Responses: problemResponses(map[int]string{
			404: "NOT_FOUND when no category has this slug.",
		}),
	}), svc.getWebsiteCategory)

	registerVocabularyAdmin(api, vocabularyOps{
		noun: "category", path: "/admin/website-categories", param: "website_category_id", suffix: "WebsiteCategory",
		createDesc: "slug must be unused (ALREADY_EXISTS); label needs a non-whitespace character.",
		deleteDesc: "A category that still has websites is WEBSITE_CATEGORY_NOT_EMPTY with website_count, and nothing is deleted.",
		deleteErrs: map[int]string{409: "WEBSITE_CATEGORY_NOT_EMPTY when websites are still listed under the category."},
	}, svc.createAdminWebsiteCategory, svc.getAdminWebsiteCategory, svc.updateAdminWebsiteCategory, svc.deleteAdminWebsiteCategory)
}

func registerTags(api huma.API, svc *Service) {
	huma.Register(api, v1.Public(huma.Operation{
		OperationID: "listWebsiteTags",
		Method:      http.MethodGet,
		Path:        "/website-tags",
		Summary:     "List website tags",
		Description: "Lists tags as a cursor page by ascending id.",
		Tags:        []string{tagTaxonomy},
		Responses: problemResponses(map[int]string{
			400: "INVALID_CURSOR or LIMIT_TOO_LARGE.",
		}),
	}), svc.listWebsiteTags)

	huma.Register(api, v1.Public(huma.Operation{
		OperationID: "getWebsiteTag",
		Method:      http.MethodGet,
		Path:        "/website-tags/{website_tag_slug}",
		Summary:     "Get a website tag by its slug",
		Description: "Returns the tag whose slug is website_tag_slug. Its websites are listWebsites with website_tag_id.",
		Tags:        []string{tagTaxonomy},
		Responses: problemResponses(map[int]string{
			404: "NOT_FOUND when no tag has this slug.",
		}),
	}), svc.getWebsiteTag)

	registerVocabularyAdmin(api, vocabularyOps{
		noun: "tag", path: "/admin/website-tags", param: "website_tag_id", suffix: "WebsiteTag",
		createDesc: "slug must be unused (ALREADY_EXISTS); label needs a non-whitespace character; website_tag_group_id must exist (UNKNOWN_REFERENCE). In an update, website_tag_group_id null moves the tag out of every group.",
		deleteDesc: "Deleting a tag takes it off every website.",
	}, svc.createAdminWebsiteTag, svc.getAdminWebsiteTag, svc.updateAdminWebsiteTag, svc.deleteAdminWebsiteTag)
}

func registerTagGroups(api huma.API, svc *Service) {
	huma.Register(api, v1.Public(huma.Operation{
		OperationID: "listWebsiteTagGroups",
		Method:      http.MethodGet,
		Path:        "/website-tag-groups",
		Summary:     "List website tag groups",
		Description: "Lists tag groups as a cursor page by ascending sort_order, ties broken by ascending id.",
		Tags:        []string{tagTaxonomy},
		Responses: problemResponses(map[int]string{
			400: "INVALID_CURSOR or LIMIT_TOO_LARGE.",
		}),
	}), svc.listWebsiteTagGroups)

	registerVocabularyAdmin(api, vocabularyOps{
		noun: "tag group", path: "/admin/website-tag-groups", param: "website_tag_group_id", suffix: "WebsiteTagGroup",
		createDesc: "slug must be unused (ALREADY_EXISTS); label needs a non-whitespace character. Turning is_multi_select off does not touch websites that already carry several of its tags.",
		deleteDesc: "The group's tags stay and become ungrouped.",
	}, svc.createAdminWebsiteTagGroup, svc.getAdminWebsiteTagGroup, svc.updateAdminWebsiteTagGroup, svc.deleteAdminWebsiteTagGroup)
}

type vocabularyOps struct {
	noun, path, param, suffix string
	createDesc, deleteDesc    string
	deleteErrs                map[int]string
}

func registerVocabularyAdmin[CI, CO, GI, GO, PI, DI any](
	api huma.API, v vocabularyOps,
	create func(context.Context, *CI) (*CO, error),
	get func(context.Context, *GI) (*GO, error),
	patch func(context.Context, *PI) (*GO, error),
	del func(context.Context, *DI) (*noContentOutput, error),
) {
	item := v.path + "/{" + v.param + "}"
	missing := "NOT_FOUND when the " + v.noun + " does not exist."
	huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
		OperationID:   "createAdmin" + v.suffix,
		Method:        http.MethodPost,
		Path:          v.path,
		Summary:       "Create a website " + v.noun,
		DefaultStatus: http.StatusCreated,
		Description:   "Creates a website " + v.noun + " and returns it. Needs website.create. " + bearerNote + activeCheck + v.createDesc,
		Tags:          []string{tagTaxonomy},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without website.create; ACCOUNT_BANNED.",
			409: "ALREADY_EXISTS when the slug is taken; IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "VALIDATION_FAILED for a blank label, a field outside its format, or an unknown reference.",
		}),
	})), create)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getAdmin" + v.suffix,
		Method:      http.MethodGet,
		Path:        item,
		Summary:     "Get a website " + v.noun + " for editing",
		Description: "Returns the " + v.noun + " with its timestamps. Needs website.edit. " + bearerNote + activeCheck,
		Tags:        []string{tagTaxonomy},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without website.edit; ACCOUNT_BANNED.",
			404: missing,
		}),
	}), get)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateAdmin" + v.suffix,
		Method:      http.MethodPatch,
		Path:        item,
		Summary:     "Edit a website " + v.noun,
		Description: "Changes the fields that are sent and returns the " + v.noun + ". Needs website.edit, checked before the " + v.noun + " is looked up. " + bearerNote + activeCheck + v.createDesc + " An empty object writes nothing.",
		Tags:        []string{tagTaxonomy},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without website.edit; ACCOUNT_BANNED.",
			404: missing,
			409: "ALREADY_EXISTS when the new slug is taken.",
			422: "VALIDATION_FAILED for a blank label, a field outside its format, or an unknown reference.",
		}),
	}), patch)

	delErrs := map[int]string{
		403: "PERMISSION_REQUIRED without website.delete; ACCOUNT_BANNED.",
		404: missing,
	}
	for status, desc := range v.deleteErrs {
		delErrs[status] = desc
	}
	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "deleteAdmin" + v.suffix,
		Method:        http.MethodDelete,
		Path:          item,
		Summary:       "Delete a website " + v.noun,
		DefaultStatus: http.StatusNoContent,
		Description:   "Deletes the " + v.noun + ". Needs website.delete, checked before the " + v.noun + " is looked up. " + bearerNote + activeCheck + v.deleteDesc,
		Tags:          []string{tagTaxonomy},
		Responses:     problemResponses(delErrs),
	}), del)
}
