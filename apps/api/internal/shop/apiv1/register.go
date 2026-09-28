package apiv1

import (
	"context"
	"net/http"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
)

type ctxKey int

const idempotencyKeyCtx ctxKey = iota

// Runs after the API-level idempotency middleware, which has already required
// and validated the header.
func withIdempotencyKey(ctx huma.Context, next func(huma.Context)) {
	key := humafiber.Unwrap(ctx).Get("Idempotency-Key")
	next(huma.WithValue(ctx, idempotencyKeyCtx, key))
}

func idempotencyKey(ctx context.Context) string {
	s, _ := ctx.Value(idempotencyKeyCtx).(string)
	return s
}

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listShopOffers",
			Method:      http.MethodGet,
			Path:        "/shop/offers",
			Summary:     "List what this forum's moemoepoint shop sells",
			Description: "Every offer on sale in this forum's shop: the NextMoe-wide offers and this forum's own zone (is_site_exclusive), in shop order. Not paged. " +
				"The same for every caller; the caller's balance, holdings and purchase counts are in getMyShopInventory. " +
				"Refreshed from the account service at most once a minute; while it cannot be reached the last list is served. " +
				"An offer holding an item of a type this forum does not render is left out.",
			Tags: []string{"shop"},
			Responses: problemResponses(map[int]string{
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached and no list has been fetched yet.",
			}),
		}), s.listShopOffers)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getMyShopInventory",
			Method:      http.MethodGet,
			Path:        "/me/shop",
			Summary:     "Get the caller's shop balance, items, loadout and orders",
			Description: "Live from the account service. It holds the caller's redeem codes: never show it to anyone else. " +
				"An offer cannot be bought again while any of its non-redeem_code rewards is in items with is_active true and no expires_at.",
			Tags: []string{"shop"},
			Responses: problemResponses(map[int]string{
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), s.getMyShopInventory)

		huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
			OperationID:   "createShopOrder",
			Method:        http.MethodPost,
			Path:          "/me/shop/orders",
			Summary:       "Buy a shop offer with moemoepoint",
			DefaultStatus: http.StatusCreated,
			Description: "Charges the offer's price and gives its rewards in one step at the account service, which does not ask the caller again. " +
				"Send it only from the caller's confirmation of the offer, its price and the balance after it; for a redeem code, say that it cannot be refunded. " +
				"Retrying with the same Idempotency-Key never charges twice. A redeem code is in order.redeem_codes; show it at once.",
			Tags:        []string{"shop"},
			Middlewares: huma.Middlewares{withIdempotencyKey},
			Responses: problemResponses(map[int]string{
				403: "MOEMOEPOINT_INSUFFICIENT, with required when the offer is known, when the live balance is below the price.",
				409: "SHOP_OFFER_UNAVAILABLE, SHOP_OFFER_SOLD_OUT, SHOP_PURCHASE_LIMIT_REACHED, or ALREADY_EXISTS when the caller already holds one of its items permanently.",
				422: "VALIDATION_FAILED when offer_id is not an offer id.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached. Retry with the same Idempotency-Key.",
			}),
		})), s.createShopOrder)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "setShopLoadoutSlot",
			Method:      http.MethodPut,
			Path:        "/me/shop/loadout/{scope}/{slot}",
			Summary:     "Wear an item",
			Description: "Puts an item the caller holds into the slot, for every NextMoe site or for this forum only. Wearing what is already worn is a no-op that answers the same.",
			Tags:        []string{"shop"},
			Responses: problemResponses(map[int]string{
				409: "SHOP_ITEM_NOT_OWNED when the caller does not hold the item or it has expired.",
				422: "VALIDATION_FAILED at /item_id when it names no item, or an item of another slot's type.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), s.setShopLoadoutSlot)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "clearShopLoadoutSlot",
			Method:      http.MethodDelete,
			Path:        "/me/shop/loadout/{scope}/{slot}",
			Summary:     "Take an item off",
			Description: "Empties the slot in that scope. Emptying an empty slot is a no-op that answers the same. Taking off the this_site choice shows the everywhere one here again.",
			Tags:        []string{"shop"},
			Responses: problemResponses(map[int]string{
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), s.clearShopLoadoutSlot)
	}
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
