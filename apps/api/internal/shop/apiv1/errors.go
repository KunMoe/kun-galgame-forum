package apiv1

import (
	"errors"

	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

const (
	upstreamBadRequest       = 1
	upstreamInvalidID        = 2
	upstreamMissingParam     = 8
	upstreamItemNotFound     = 19001
	upstreamOfferUnavailable = 19002
	upstreamAlreadyOwned     = 19003
	upstreamLimitReached     = 19004
	upstreamSoldOut          = 19005
	upstreamNotOwned         = 19006
	upstreamInvalidItem      = 19008
	upstreamIdemConflict     = 19014
	upstreamNotStorefront    = 19017
	upstreamForbidden        = 5
	upstreamInsufficient     = 16006
)

func upstreamCode(err error) int {
	var oe *userclient.OAuthError
	if errors.As(err, &oe) {
		return oe.Code
	}
	return 0
}

func upstreamProblem(err error) *problem.Problem {
	switch upstreamCode(err) {
	case upstreamOfferUnavailable:
		return problem.New(problem.CodeShopOfferUnavailable, "The offer is not on sale in this forum's shop.")
	case upstreamAlreadyOwned:
		return problem.New(problem.CodeAlreadyExists, "The caller already holds one of this offer's items permanently.")
	case upstreamLimitReached:
		return problem.New(problem.CodeShopPurchaseLimitReached, "The caller has reached this offer's purchase limit for the current period.")
	case upstreamSoldOut:
		return problem.New(problem.CodeShopOfferSoldOut, "The offer is sold out.")
	case upstreamInsufficient:
		return problem.New(problem.CodeMoemoepointInsufficient, "The caller's live moemoepoint balance is below the offer's price.")
	case upstreamIdemConflict:
		return problem.New(problem.CodeIdempotencyKeyReused, "The same Idempotency-Key was already used to buy another offer.")
	case upstreamNotStorefront, upstreamForbidden, upstreamBadRequest, upstreamInvalidID, upstreamMissingParam:
		return problem.Internal(err)
	}
	return problem.Unavailable(err)
}

func equipProblem(err error) *problem.Problem {
	switch upstreamCode(err) {
	case upstreamNotOwned:
		return problem.New(problem.CodeShopItemNotOwned, "The caller does not currently own the item.")
	case upstreamItemNotFound:
		return validationFailed(problem.AtPointer("/item_id", problem.ReasonUnknownReference, "no item has this id", nil))
	case upstreamInvalidItem:
		return validationFailed(problem.AtPointer("/item_id", problem.ReasonInconsistentWith, "the item is not worn in /slot", nil))
	}
	return upstreamProblem(err)
}

func validationFailed(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
}
