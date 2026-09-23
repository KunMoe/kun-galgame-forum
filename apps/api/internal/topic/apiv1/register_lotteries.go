package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

const lotteryNotFound = "NOT_FOUND when the lottery does not exist, was created by a banned user, or belongs to a topic getTopic would not return to the caller."

func RegisterLotteries(l *Lotteries) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "listTopicLotteries",
			Method:      http.MethodGet,
			Path:        "/topics/{topic_id}/lotteries",
			Summary:     "List a topic's lotteries",
			Description: "Lists every lottery of the topic, newest first. It is not paginated: a topic holds at most 10 lotteries. " +
				"Lotteries by banned authors are left out. NOT_FOUND under the same conditions as getTopic.",
			Tags: []string{"topics"},
		}), l.listTopicLotteries)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "getLottery",
			Method:      http.MethodGet,
			Path:        "/lotteries/{lottery_id}",
			Summary:     "Get a lottery",
			Description: "Returns one lottery. seed is null until the draw. A redemption code is never part of this or any other read; " +
				"a winner reveals it with revealLotteryCode. " + lotteryNotFound,
			Tags: []string{"topics"},
		}), l.getLottery)

		huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
			OperationID:   "createLottery",
			Method:        http.MethodPost,
			Path:          "/topics/{topic_id}/lotteries",
			Summary:       "Create a lottery",
			DefaultStatus: http.StatusCreated,
			Description: "Creates a lottery on the topic and returns it. It needs the topic's author, or staff holding the create permission, " +
				"and the topic must be one the caller can read. Anyone but staff also needs an account at least 30 days old or at least 100 moemoepoint. " +
				"**The point prizes are paid for by the caller**: their budget is taken from the caller's balance now, " +
				"and whatever is not paid out comes back when the lottery is cancelled, deleted before the draw, or drawn with slots left over. " +
				"Codes of a code prize are sealed at rest and never returned by a read. Creating a lottery bumps the topic. " +
				"NOT_FOUND under the same conditions as getTopic.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller is neither the topic's author nor staff; LOTTERY_CREATOR_INELIGIBLE when the account is too new and holds too little moemoepoint; " +
					"MOEMOEPOINT_INSUFFICIENT when the balance does not cover the point prizes' budget; SCOPE_REQUIRED or ACCOUNT_BANNED.",
				422: "VALIDATION_FAILED when the lottery's shape is inconsistent or the topic already holds 10 lotteries; CONTENT_REJECTED when the trust-and-safety check refuses the text.",
			}),
		})), l.createLottery)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "updateLottery",
			Method:      http.MethodPatch,
			Path:        "/lotteries/{lottery_id}",
			Summary:     "Update, draw or cancel a lottery",
			Description: "With state, draws the lottery now (drawn) or cancels it and refunds the author's escrow (cancelled); state is accepted only from open and never together with another field. " +
				"Without state, changes the fields present in the body, checked against the lottery as it would be stored. " +
				"Once anyone has entered, entry_mode, floor_rule and prizes are IMMUTABLE. " +
				"Replacing the prizes charges or refunds the difference in point budget to the lottery's author. " +
				"Only text this request submits is checked by trust and safety. It needs can_edit. " + lotteryNotFound,
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller may read but not manage the lottery; MOEMOEPOINT_INSUFFICIENT when a larger point budget is not covered.",
				409: "INVALID_STATE_TRANSITION when state is sent and the lottery is not open; LOTTERY_CLOSED when fields are sent and the lottery is not open.",
				422: "VALIDATION_FAILED, or CONTENT_REJECTED when the trust-and-safety check refuses the text.",
			}),
		}), l.updateLottery)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID:   "deleteLottery",
			Method:        http.MethodDelete,
			Path:          "/lotteries/{lottery_id}",
			Summary:       "Delete a lottery",
			DefaultStatus: http.StatusNoContent,
			Description: "Deletes the lottery with its prizes, entries and held codes, and refunds the author's escrow if it still holds any. " +
				"The author may delete it before the draw; staff holding the manage permission may also delete a drawn one. " +
				"Nobody may delete it while it is being drawn. " + lotteryNotFound,
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller may read but not manage the lottery.",
				409: "LOTTERY_DRAWN when it is being drawn, or it is drawn and the caller is not staff.",
			}),
		}), l.deleteLottery)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "enterLottery",
			Method:      http.MethodPut,
			Path:        "/lotteries/{lottery_id}/entries/me",
			Summary:     "Enter a lottery",
			Description: "Enters the caller and returns the lottery. Entering again is a no-op that answers the same. " +
				"The check is the one behind viewer.can_enter. " + lotteryNotFound,
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "LOTTERY_INELIGIBLE, with reason, when the caller does not meet the entry requirements.",
				409: "LOTTERY_CLOSED when the lottery is not open or is past closes_at.",
			}),
		}), l.enterLottery)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "withdrawLottery",
			Method:      http.MethodDelete,
			Path:        "/lotteries/{lottery_id}/entries/me",
			Summary:     "Withdraw from a lottery",
			Description: "Removes the caller's entry and returns the lottery. Withdrawing without an entry is a no-op that answers the same. " + lotteryNotFound,
			Tags:        []string{"topics"},
			Responses: problemResponses(map[int]string{
				409: "LOTTERY_CLOSED when the lottery is not open, is past closes_at, or the caller has already won.",
			}),
		}), l.withdrawLottery)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "listLotteryEntries",
			Method:      http.MethodGet,
			Path:        "/lotteries/{lottery_id}/entries",
			Summary:     "List a lottery's entries",
			Description: "Lists the entries oldest first, ties broken by ascending id. There is one sort and no sort parameter. " +
				"Banned entrants are left out, so a page can hold fewer than limit items. " + lotteryNotFound,
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when is_entry_list_public is false and the caller is neither the author nor staff holding the view permission.",
			}),
		}), l.listLotteryEntries)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "getLotteryWinner",
			Method:      http.MethodGet,
			Path:        "/lotteries/{lottery_id}/winners/{winner_id}",
			Summary:     "Get a lottery winner",
			Description: "Returns one winner, as in winners of getLottery. NOT_FOUND when the winner is banned, or under the same conditions as getLottery.",
			Tags:        []string{"topics"},
		}), l.getLotteryWinner)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "updateLotteryWinner",
			Method:      http.MethodPatch,
			Path:        "/lotteries/{lottery_id}/winners/{winner_id}",
			Summary:     "Move a winner's fulfillment",
			Description: "Moves an offline prize's fulfillment. The author or staff may move pending and shipped to any state; " +
				"the winner may only confirm receipt or give the prize up. received and forfeited are final, " +
				"and code and point prizes are moved by the site alone. Setting the current state again is a no-op. " +
				"The winner acts on their own prize even when they can no longer read the topic.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller may not make this move.",
				409: "INVALID_STATE_TRANSITION when the move is not allowed from the current state or for this kind of prize.",
			}),
		}), l.updateLotteryWinner)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "revealLotteryCode",
			Method:      http.MethodPost,
			Path:        "/lotteries/{lottery_id}/code-reveals",
			Summary:     "Reveal the caller's redemption code",
			Description: "Returns the redemption code the caller won and marks the prize received. It can be called again and returns the same code. " +
				"**It is a POST so no page load can fetch it**, and it takes no Idempotency-Key, because a stored response would keep the code at rest. " +
				"The winner reveals their code even when they can no longer read the topic. " +
				"NOT_FOUND when the caller did not win a code in this lottery.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				409: "REDEMPTION_CODE_FORFEITED when the code was given up or not revealed before claim_expires_at.",
			}),
		}), l.revealLotteryCode)
	}
}
