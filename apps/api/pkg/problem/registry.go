package problem

import (
	"net/http"
	"regexp"
	"strings"
)

const TypeURIPrefix = "https://developer.nextmoe.dev/problems/"

type Domain string

const (
	DomainPlatform   Domain = "platform"
	DomainKungal     Domain = "kungal"
	DomainMe         Domain = "me"
	DomainModeration Domain = "moderation"
)

var DomainOrder = []Domain{DomainPlatform, DomainKungal, DomainMe, DomainModeration}

type ExtDef struct {
	Name string
	Type string
}

type Def struct {
	Code        string
	Domain      Domain
	Status      int
	Title       string
	Description string
	Extensions  []ExtDef
}

type ReasonDef struct {
	Reason      string
	Title       string
	Description string
	Params      []string
}

const (
	CodeMalformedBody                = "MALFORMED_BODY"
	CodeInvalidParameter             = "INVALID_PARAMETER"
	CodeUnknownEnumValue             = "UNKNOWN_ENUM_VALUE"
	CodeLimitTooLarge                = "LIMIT_TOO_LARGE"
	CodeInvalidCursor                = "INVALID_CURSOR"
	CodeUnknownSort                  = "UNKNOWN_SORT"
	CodeMissingCredential            = "MISSING_CREDENTIAL"
	CodeInvalidCredential            = "INVALID_CREDENTIAL"
	CodeScopeRequired                = "SCOPE_REQUIRED"
	CodeAccountBanned                = "ACCOUNT_BANNED"
	CodeNotFound                     = "NOT_FOUND"
	CodeEntityMerged                 = "ENTITY_MERGED"
	CodeMethodNotAllowed             = "METHOD_NOT_ALLOWED"
	CodeIdempotencyKeyReused         = "IDEMPOTENCY_KEY_REUSED"
	CodeIdempotencyRequestInProgress = "IDEMPOTENCY_REQUEST_IN_PROGRESS"
	CodeUnsupportedMediaType         = "UNSUPPORTED_MEDIA_TYPE"
	CodePayloadTooLarge              = "PAYLOAD_TOO_LARGE"
	CodeValidationFailed             = "VALIDATION_FAILED"
	CodeInternalError                = "INTERNAL_ERROR"
	CodeServiceUnavailable           = "SERVICE_UNAVAILABLE"
	CodePermissionRequired           = "PERMISSION_REQUIRED"
	CodeContentRejected              = "CONTENT_REJECTED"
	CodeTopicDailyLimitReached       = "TOPIC_DAILY_LIMIT_REACHED"
	CodeImageDailyLimitReached       = "IMAGE_DAILY_LIMIT_REACHED"
	CodeImageRejected                = "IMAGE_REJECTED"
	CodeDraftLimitReached            = "DRAFT_LIMIT_REACHED"
	CodeMoemoepointInsufficient      = "MOEMOEPOINT_INSUFFICIENT"
	CodeSelfLikeForbidden            = "SELF_LIKE_FORBIDDEN"
	CodeResourcePublishBanned        = "RESOURCE_PUBLISH_BANNED"
	CodePollClosed                   = "POLL_CLOSED"
	CodeVoteAlreadyCast              = "VOTE_ALREADY_CAST"
	CodeSelfUpvoteForbidden          = "SELF_UPVOTE_FORBIDDEN"
	CodeSelfAnswerForbidden          = "SELF_ANSWER_FORBIDDEN"
	CodeRateLimited                  = "RATE_LIMITED"
	CodeQuotaExceeded                = "QUOTA_EXCEEDED"
	CodeQuizAnswerRequired           = "QUIZ_ANSWER_REQUIRED"
	CodeInvalidStateTransition       = "INVALID_STATE_TRANSITION"
	CodeLotteryClosed                = "LOTTERY_CLOSED"
	CodeLotteryIneligible            = "LOTTERY_INELIGIBLE"
	CodeLotteryCreatorIneligible     = "LOTTERY_CREATOR_INELIGIBLE"
	CodeLotteryDrawn                 = "LOTTERY_DRAWN"
	CodeRedemptionCodeForfeited      = "REDEMPTION_CODE_FORFEITED"
	CodeAlreadyExists                = "ALREADY_EXISTS"
	CodeDuplicateSuspects            = "DUPLICATE_SUSPECTS"
	CodePreconditionFailed           = "PRECONDITION_FAILED"
	CodeUsernameTaken                = "USERNAME_TAKEN"
	CodeCreatorIneligible            = "CREATOR_INELIGIBLE"
	CodeCreatorApplicationCooldown   = "CREATOR_APPLICATION_COOLDOWN"
	CodeWebsiteCategoryNotEmpty      = "WEBSITE_CATEGORY_NOT_EMPTY"
	CodeUserProtected                = "USER_PROTECTED"
)

const (
	ReasonRequired         = "REQUIRED"
	ReasonInvalidFormat    = "INVALID_FORMAT"
	ReasonOutOfRange       = "OUT_OF_RANGE"
	ReasonTooLong          = "TOO_LONG"
	ReasonTooShort         = "TOO_SHORT"
	ReasonTooManyItems     = "TOO_MANY_ITEMS"
	ReasonTooFewItems      = "TOO_FEW_ITEMS"
	ReasonDuplicateItem    = "DUPLICATE_ITEM"
	ReasonUnknownValue     = "UNKNOWN_VALUE"
	ReasonNotAllowedValue  = "NOT_ALLOWED_VALUE"
	ReasonUnknownReference = "UNKNOWN_REFERENCE"
	ReasonImmutable        = "IMMUTABLE"
	ReasonInconsistentWith = "INCONSISTENT_WITH"
	ReasonNotPermitted     = "NOT_PERMITTED"
)

const (
	ParamMaxLength = "max_length"
	ParamMinLength = "min_length"
	ParamMinimum   = "minimum"
	ParamMaximum   = "maximum"
	ParamMaxItems  = "max_items"
	ParamMinItems  = "min_items"
	ParamAllowed   = "allowed"
)

var Codes = []Def{
	{CodeMalformedBody, DomainPlatform, http.StatusBadRequest, "Malformed body", "Request body is not valid JSON, or is not a valid instance of the declared media type.", nil},
	{CodeInvalidParameter, DomainPlatform, http.StatusBadRequest, "Invalid parameter", "A parameter is syntactically wrong: a boolean that is not true/false, an integer that is not an integer, a date that is not YYYY-MM-DD.", nil},
	{CodeUnknownEnumValue, DomainPlatform, http.StatusBadRequest, "Unknown enum value", "A closed vocabulary received an unknown token at parse time.", nil},
	{CodeLimitTooLarge, DomainPlatform, http.StatusBadRequest, "Limit too large", "limit is greater than 100. The value is not clamped.", nil},
	{CodeInvalidCursor, DomainPlatform, http.StatusBadRequest, "Invalid cursor", "The cursor cannot be parsed or is no longer valid.", nil},
	{CodeUnknownSort, DomainPlatform, http.StatusBadRequest, "Unknown sort", "sort= received a key this collection has not declared.", nil},
	{CodeMissingCredential, DomainPlatform, http.StatusUnauthorized, "Missing credential", "The request has no Authorization header.", nil},
	{CodeInvalidCredential, DomainPlatform, http.StatusUnauthorized, "Invalid credential", "A credential was sent but it is invalid, expired, or revoked.", nil},
	{CodeScopeRequired, DomainPlatform, http.StatusForbidden, "Scope required", "The credential is valid but lacks the scope this operation needs.", nil},
	{CodeAccountBanned, DomainKungal, http.StatusForbidden, "Account banned", "The signed-in user's account is banned.", nil},
	{CodeNotFound, DomainPlatform, http.StatusNotFound, "Not found", "Nothing visible exists at this URL.", nil},
	{CodeEntityMerged, DomainPlatform, http.StatusNotFound, "Entity merged", "The catalog entity at this URL was merged into another. object names its family and current_id the entity it became; read that one instead.", []ExtDef{{Name: "object", Type: "string"}, {Name: "current_id", Type: "string"}}},
	{CodeMethodNotAllowed, DomainPlatform, http.StatusMethodNotAllowed, "Method not allowed", "The path exists but this method does not.", nil},
	{CodeIdempotencyKeyReused, DomainPlatform, http.StatusConflict, "Idempotency key reused", "The same Idempotency-Key was sent with a different request body.", nil},
	{CodeIdempotencyRequestInProgress, DomainKungal, http.StatusConflict, "Idempotency request in progress", "A request with the same Idempotency-Key is still being processed. Retry after it completes.", nil},
	{CodeUnsupportedMediaType, DomainPlatform, http.StatusUnsupportedMediaType, "Unsupported media type", "The request body media type is not supported.", nil},
	{CodePayloadTooLarge, DomainPlatform, http.StatusRequestEntityTooLarge, "Payload too large", "The request body is larger than this operation accepts. Retrying the same body cannot succeed.", nil},
	{CodeValidationFailed, DomainPlatform, http.StatusUnprocessableEntity, "Validation failed", "The request is syntactically valid but semantically not. errors[] is present and non-empty.", nil},
	{CodeInternalError, DomainPlatform, http.StatusInternalServerError, "Internal error", "A bug on our side, including the output of panic recovery.", nil},
	{CodeServiceUnavailable, DomainPlatform, http.StatusServiceUnavailable, "Service unavailable", "A dependency is unavailable. The request may be retried.", nil},
	{CodePermissionRequired, DomainModeration, http.StatusForbidden, "Permission required", "The token lacks the permission this decision needs.", nil},
	{CodeContentRejected, DomainKungal, http.StatusUnprocessableEntity, "Content rejected", "The trust-and-safety check refused the submitted text. Nothing was written.", nil},
	{CodeTopicDailyLimitReached, DomainKungal, http.StatusTooManyRequests, "Topic daily limit reached", "The caller has created as many topics in the last 24 hours as their moemoepoint balance allows. limit is that number.", []ExtDef{{Name: "limit", Type: "integer"}}},
	{CodeImageDailyLimitReached, DomainKungal, http.StatusTooManyRequests, "Image daily limit reached", "The caller has uploaded as many images today, Asia/Shanghai, as one user may. limit is that number.", []ExtDef{{Name: "limit", Type: "integer"}}},
	{CodeImageRejected, DomainKungal, http.StatusUnprocessableEntity, "Image rejected", "The image service's moderation refused the image. Nothing was stored.", nil},
	{CodeDraftLimitReached, DomainKungal, http.StatusConflict, "Draft limit reached", "The caller already holds the maximum number of drafts. A draft is never overwritten, so the only way to make room is to delete one. limit is that number.", []ExtDef{{Name: "limit", Type: "integer"}}},
	{CodeMoemoepointInsufficient, DomainKungal, http.StatusForbidden, "Moemoepoint insufficient", "The caller's moemoepoint balance, as this forum last cached it, is below what the operation costs. required is that cost.", []ExtDef{{Name: "required", Type: "integer"}}},
	{CodeSelfLikeForbidden, DomainKungal, http.StatusForbidden, "Self like forbidden", "Users cannot like what they wrote themselves.", nil},
	{CodeResourcePublishBanned, DomainKungal, http.StatusForbidden, "Resource publish banned", "This work is banned from publishing download resources.", nil},
	{CodePollClosed, DomainKungal, http.StatusConflict, "Poll closed", "The poll no longer accepts votes: it is past closes_at. Nothing about the request is wrong.", nil},
	{CodeVoteAlreadyCast, DomainKungal, http.StatusConflict, "Vote already cast", "The caller has already voted and this poll does not allow changing a vote.", nil},
	{CodeSelfUpvoteForbidden, DomainKungal, http.StatusForbidden, "Self upvote forbidden", "Users cannot upvote their own topics.", nil},
	{CodeSelfAnswerForbidden, DomainKungal, http.StatusForbidden, "Self answer forbidden", "Users cannot answer a quiz they authored.", nil},
	{CodeRateLimited, DomainPlatform, http.StatusTooManyRequests, "Rate limited", "A rate limit was exceeded. Retry-After, in seconds, is present only when the limiter says when to retry.", nil},
	{CodeQuotaExceeded, DomainPlatform, http.StatusTooManyRequests, "Quota exceeded", "A quota for this operation is exhausted. Retry-After, in seconds, is when it resets.", nil},
	{CodeQuizAnswerRequired, DomainKungal, http.StatusForbidden, "Quiz answer required", "The caller must already have answered this quiz; the author cannot answer their own quiz.", nil},
	{CodeInvalidStateTransition, DomainMe, http.StatusConflict, "Invalid state transition", "The current state does not allow this transition. detail names the current state.", nil},
	{CodeLotteryClosed, DomainKungal, http.StatusConflict, "Lottery closed", "The operation needs an open lottery, and this one has been drawn, cancelled, is being drawn, or is past closes_at. Nothing about the request is wrong.", nil},
	{CodeLotteryIneligible, DomainKungal, http.StatusForbidden, "Lottery ineligible", "The caller does not meet this lottery's entry requirements. reason is one of no_signup, own_lottery, reply_required, moemoepoint_below_minimum, account_too_new; the thresholds are on the lottery itself.", []ExtDef{{Name: "reason", Type: "string"}}},
	{CodeLotteryCreatorIneligible, DomainKungal, http.StatusForbidden, "Lottery creator ineligible", "Starting a lottery needs an account at least min_account_age_days old or at least min_moemoepoint moemoepoint. It is an anti-scam bar, not a permission.", []ExtDef{{Name: "min_account_age_days", Type: "integer"}, {Name: "min_moemoepoint", Type: "integer"}}},
	{CodeLotteryDrawn, DomainKungal, http.StatusConflict, "Lottery drawn", "The lottery is being drawn, or has been drawn and only staff may delete it: its winners still need it to collect their prizes.", nil},
	{CodeRedemptionCodeForfeited, DomainKungal, http.StatusConflict, "Redemption code forfeited", "The caller won this code, but it was given up or not revealed before claim_expires_at, and it can no longer be revealed.", nil},
	{CodeAlreadyExists, DomainMe, http.StatusConflict, "Already exists", "The same subject already has a live record for this target.", nil},
	{CodeDuplicateSuspects, DomainMe, http.StatusConflict, "Duplicate suspects", "The mint's titles match live works of the same medium; suspects[] names them. Nothing was written. Re-send with confirm_duplicates=true to mint anyway — the pairs are still filed for reconciliation.", []ExtDef{{Name: "suspects", Type: "array"}}},
	{CodePreconditionFailed, DomainPlatform, http.StatusPreconditionFailed, "Precondition failed", "If-Match did not match the current representation.", nil},
	{CodeUsernameTaken, DomainKungal, http.StatusConflict, "Username taken", "The requested name is already in use by another account.", nil},
	{CodeCreatorIneligible, DomainKungal, http.StatusForbidden, "Creator ineligible", "The caller does not meet the conditions to apply for the creator role.", nil},
	{CodeCreatorApplicationCooldown, DomainKungal, http.StatusConflict, "Creator application cooldown", "A declined creator application is still inside its cooldown window.", nil},
	{CodeWebsiteCategoryNotEmpty, DomainKungal, http.StatusConflict, "Website category not empty", "Websites are still listed under the category, so it cannot be deleted. website_count is how many.", []ExtDef{{Name: "website_count", Type: "integer"}}},
	{CodeUserProtected, DomainKungal, http.StatusForbidden, "User protected", "The target user holds a staff role, and the operation is never applied to staff: their content includes site documentation other users read.", nil},
}

var Reasons = []ReasonDef{
	{ReasonRequired, "Required", "A required field is missing or null.", nil},
	{ReasonInvalidFormat, "Invalid format", "The value does not match the expected format (date, URI, hash, id string).", nil},
	{ReasonOutOfRange, "Out of range", "A numeric value is out of range, including a date outside the allowed interval.", []string{ParamMinimum, ParamMaximum}},
	{ReasonTooLong, "Too long", "A string is longer than its maxLength.", []string{ParamMaxLength}},
	{ReasonTooShort, "Too short", "A string is shorter than its minLength.", []string{ParamMinLength}},
	{ReasonTooManyItems, "Too many items", "An array exceeds its item limit.", []string{ParamMaxItems}},
	{ReasonTooFewItems, "Too few items", "The array has fewer items than its minimum.", []string{ParamMinItems}},
	{ReasonDuplicateItem, "Duplicate item", "An array that must be unique contains a duplicate.", nil},
	{ReasonUnknownValue, "Unknown value", "The value is not in this field's closed vocabulary.", []string{ParamAllowed}},
	{ReasonNotAllowedValue, "Not allowed value", "The value is in the vocabulary but is not accepted in this context.", nil},
	{ReasonUnknownReference, "Unknown reference", "The value refers to an entity that is not visible. Absence, merge, and visibility filtering are not distinguished.", nil},
	{ReasonImmutable, "Immutable", "This field cannot be changed in the current state.", nil},
	{ReasonInconsistentWith, "Inconsistent with", "The value contradicts another field. detail names that field's pointer.", nil},
	{ReasonNotPermitted, "Not permitted", "The caller is not allowed to act on this position. Unlike IMMUTABLE this is about the actor, not the field's state.", nil},
}

var (
	codeByName   = indexCodes()
	reasonByName = indexReasons()
	NamePattern  = regexp.MustCompile(`^[A-Z][A-Z0-9_]*[A-Z0-9]$`)
)

func indexCodes() map[string]Def {
	m := make(map[string]Def, len(Codes))
	for _, d := range Codes {
		m[d.Code] = d
	}
	return m
}

func indexReasons() map[string]ReasonDef {
	m := make(map[string]ReasonDef, len(Reasons))
	for _, d := range Reasons {
		m[d.Reason] = d
	}
	return m
}

func Lookup(code string) (Def, bool) {
	d, ok := codeByName[code]
	return d, ok
}

func LookupReason(reason string) (ReasonDef, bool) {
	d, ok := reasonByName[reason]
	return d, ok
}

func (d Def) TypeURI() string {
	return TypeURIPrefix + string(d.Domain) + "/" + Kebab(d.Code)
}

func Kebab(code string) string {
	return strings.ToLower(strings.ReplaceAll(code, "_", "-"))
}

func CodeFromKebab(kebab string) string {
	return strings.ToUpper(strings.ReplaceAll(kebab, "-", "_"))
}

func StatusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return CodeInvalidParameter
	case http.StatusUnauthorized:
		return CodeMissingCredential
	case http.StatusForbidden:
		return CodeScopeRequired
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusMethodNotAllowed:
		return CodeMethodNotAllowed
	case http.StatusConflict:
		return CodeIdempotencyKeyReused
	case http.StatusUnsupportedMediaType:
		return CodeUnsupportedMediaType
	case http.StatusRequestEntityTooLarge:
		return CodePayloadTooLarge
	case http.StatusUnprocessableEntity:
		return CodeValidationFailed
	case http.StatusServiceUnavailable:
		return CodeServiceUnavailable
	default:
		return CodeInternalError
	}
}

func reasonAllowsParam(reason, key string) bool {
	d, ok := LookupReason(reason)
	if !ok {
		return false
	}
	for _, k := range d.Params {
		if k == key {
			return true
		}
	}
	return false
}
