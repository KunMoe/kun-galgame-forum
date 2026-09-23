package apiv1

import (
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
	tagQuizzes = "quizzes"
	tagMe      = "me"
	tagWorks   = "works"

	activeCheck = "The caller is checked against the account service's current record first: a banned account is ACCOUNT_BANNED, and a failure of that lookup is SERVICE_UNAVAILABLE with nothing written. "
	bearerNote  = "Requests authenticated with a Bearer token never carry quiz permissions. "
)

func Register(svc *Service) func(huma.API) {
	return func(api huma.API) {
		registerQuizzes(api, svc)
		registerAnswers(api, svc)
		registerEngage(api, svc)
		registerMe(api, svc)
		registerSuggestions(api, svc)
	}
}

func registerQuizzes(api huma.API, svc *Service) {
	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listQuizzes",
		Method:      http.MethodGet,
		Path:        "/quizzes",
		Summary:     "List quizzes",
		Description: "Lists quizzes as a page-number collection. Default sort is bumped_at_desc, default limit 50. " +
			"Quizzes whose author is not renderable are omitted from both items and total. " +
			"include_nsfw=false excludes quizzes linked to a local NSFW work.",
		Tags: []string{tagQuizzes},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE, UNKNOWN_SORT, LIMIT_TOO_LARGE, or INVALID_PARAMETER.",
		}),
	}), svc.listQuizzes)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "getQuiz",
		Method:      http.MethodGet,
		Path:        "/quizzes/{quiz_id}",
		Summary:     "Get a quiz",
		Description: "Returns the quiz and counts one view. An unrenderable author is NOT_FOUND. viewer is null for an anonymous caller. " +
			"solution is null until the caller has answered, is the author, or can_edit.",
		Tags: []string{tagQuizzes},
		Responses: problemResponses(map[int]string{
			404: "NOT_FOUND when the quiz does not exist or its author is not renderable.",
		}),
	}), svc.getQuiz)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getQuizSource",
		Method:      http.MethodGet,
		Path:        "/quizzes/{quiz_id}/source",
		Summary:     "Get a quiz's edit source",
		Description: "Returns the stored fields including the answer key. Needs viewer.can_edit. " + bearerNote + activeCheck,
		Tags:        []string{tagQuizzes},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without can_edit; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the quiz does not exist or its author is not renderable.",
		}),
	}), svc.getQuizSource)

	huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
		OperationID:   "createQuiz",
		Method:        http.MethodPost,
		Path:          "/quizzes",
		Summary:       "Create a quiz",
		DefaultStatus: http.StatusCreated,
		Description: "Creates a quiz and returns it. Idempotency-Key is required. " + activeCheck +
			"prompt_text is length-checked on the raw value; only whitespace is TOO_SHORT. " +
			"Location is the new quiz's path.",
		Tags: []string{tagQuizzes},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER when Idempotency-Key is missing or malformed.",
			403: "ACCOUNT_BANNED.",
			409: "IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "VALIDATION_FAILED or CONTENT_REJECTED.",
		}),
	})), svc.createQuiz)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateQuiz",
		Method:      http.MethodPatch,
		Path:        "/quizzes/{quiz_id}",
		Summary:     "Edit a quiz",
		Description: "Changes the fields that are sent and returns the quiz. Needs can_edit. " + bearerNote + activeCheck +
			"Only prompt_text, description_markdown, explanation_markdown and choices, when sent, go through the trust-and-safety check. " +
			"quiz_type cannot change. A changed answer key regrades every answerer in the same transaction.",
		Tags: []string{tagQuizzes},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without can_edit; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the quiz does not exist or its author is not renderable.",
			422: "VALIDATION_FAILED or CONTENT_REJECTED.",
		}),
	}), svc.updateQuiz)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "deleteQuiz",
		Method:        http.MethodDelete,
		Path:          "/quizzes/{quiz_id}",
		Summary:       "Delete a quiz",
		DefaultStatus: http.StatusNoContent,
		Description:   "Deletes a quiz and its answers, favorites, links and quality votes. Needs can_delete. " + bearerNote + activeCheck,
		Tags:          []string{tagQuizzes},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without can_delete; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the quiz does not exist or its author is not renderable.",
		}),
	}), svc.deleteQuiz)
}

func registerAnswers(api huma.API, svc *Service) {
	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listQuizAnswers",
		Method:      http.MethodGet,
		Path:        "/quizzes/{quiz_id}/answers",
		Summary:     "List answers to a quiz",
		Description: "Lists answerer rows newest first as a cursor collection. " +
			"submission and is_correct are null unless the caller has answered, is the author, or can_edit. " +
			"Unrenderable answerers are omitted.",
		Tags: []string{tagQuizzes},
		Responses: problemResponses(map[int]string{
			400: "INVALID_CURSOR or LIMIT_TOO_LARGE.",
			404: "NOT_FOUND when the quiz does not exist or its author is not renderable.",
		}),
	}), svc.listQuizAnswers)

	huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
		OperationID:   "createQuizAnswer",
		Method:        http.MethodPost,
		Path:          "/quizzes/{quiz_id}/answers",
		Summary:       "Answer a quiz",
		DefaultStatus: http.StatusCreated,
		Description: "Records the caller's answer. Idempotency-Key is required. " + activeCheck +
			"The author cannot answer their own quiz. A second answer is ALREADY_EXISTS. " +
			"Location is the quiz path.",
		Tags: []string{tagQuizzes},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER when Idempotency-Key is missing or malformed.",
			403: "SELF_ANSWER_FORBIDDEN or ACCOUNT_BANNED.",
			404: "NOT_FOUND when the quiz does not exist or its author is not renderable.",
			409: "ALREADY_EXISTS, IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "VALIDATION_FAILED or CONTENT_REJECTED.",
		}),
	})), svc.createQuizAnswer)
}

func registerEngage(api huma.API, svc *Service) {
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "putQuizFavorite",
		Method:      http.MethodPut,
		Path:        "/quizzes/{quiz_id}/favorite",
		Summary:     "Favorite a quiz",
		Description: "Sets the caller's favorite. Favoriting again changes nothing. " + activeCheck,
		Tags:        []string{tagQuizzes},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			404: "NOT_FOUND when the quiz does not exist or its author is not renderable.",
		}),
	}), svc.putQuizFavorite)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "deleteQuizFavorite",
		Method:      http.MethodDelete,
		Path:        "/quizzes/{quiz_id}/favorite",
		Summary:     "Unfavorite a quiz",
		Description: "Clears the caller's favorite. Unfavoriting when not favorited changes nothing. " + activeCheck,
		Tags:        []string{tagQuizzes},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			404: "NOT_FOUND when the quiz does not exist or its author is not renderable.",
		}),
	}), svc.deleteQuizFavorite)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "putQuizQualityRating",
		Method:      http.MethodPut,
		Path:        "/quizzes/{quiz_id}/quality-rating",
		Summary:     "Rate a quiz's quality",
		Description: "Sets the caller's 1–10 rating. Needs an answerer row; the author has none. Rating again with the same value changes nothing. " + activeCheck,
		Tags:        []string{tagQuizzes},
		Responses: problemResponses(map[int]string{
			403: "QUIZ_ANSWER_REQUIRED when the caller has not answered; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the quiz does not exist or its author is not renderable.",
			422: "VALIDATION_FAILED.",
		}),
	}), svc.putQuizQualityRating)
}

func registerMe(api huma.API, svc *Service) {
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyAnsweredQuizzes",
		Method:      http.MethodGet,
		Path:        "/me/answered-quizzes",
		Summary:     "List quizzes the caller has answered",
		Description: "Lists quizzes the caller has an answerer row on, newest answer first. " + activeCheck +
			"Quizzes whose author is not renderable are omitted from both items and total.",
		Tags: []string{tagMe},
		Responses: problemResponses(map[int]string{
			400: "LIMIT_TOO_LARGE or INVALID_PARAMETER.",
			403: "ACCOUNT_BANNED.",
		}),
	}), svc.listMyAnsweredQuizzes)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyQuizStates",
		Method:      http.MethodGet,
		Path:        "/me/quiz-states",
		Summary:     "Batch-read the caller's quiz favorite states",
		Description: "Answers, for each quiz id named in quiz_ids, whether the caller favorited it. " +
			"It is a batch read and is not paginated: quiz_ids is required, holds 1 to 100 ids. " +
			"An id that does not exist, is deleted, or whose author is not renderable is missing.",
		Tags: []string{tagMe},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			422: "VALIDATION_FAILED when quiz_ids is absent, empty, holds more than 100 ids, or holds something that is not a positive decimal integer.",
		}),
	}), svc.listMyQuizStates)
}

func registerSuggestions(api huma.API, svc *Service) {
	huma.Register(api, v1.Public(huma.Operation{
		OperationID: "listWorkSuggestions",
		Method:      http.MethodGet,
		Path:        "/work-suggestions",
		Summary:     "Suggest works for linking to a quiz",
		Description: "Searches catalog works by free text and returns at most 12 WorkRef values. Not paginated. " +
			"include_nsfw=false applies the SFW content_limit gate. Hidden claims are omitted.",
		Tags: []string{tagWorks},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER.",
			422: "VALIDATION_FAILED when q is too short or too long.",
			503: "SERVICE_UNAVAILABLE when catalog cannot be reached.",
		}),
	}), svc.listWorkSuggestions)
}
