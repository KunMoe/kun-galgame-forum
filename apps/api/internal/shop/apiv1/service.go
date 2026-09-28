package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"sync"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

var errUnconfigured = errors.New("apiv1 shop: service is not configured")

const (
	storefrontTTL      = time.Minute
	upstreamKeyPrefix  = "kungal:"
	storefrontDeadline = 10 * time.Second
)

type upstream interface {
	ShopStorefront(ctx context.Context) (userclient.ShopStorefront, error)
	ShopInventory(ctx context.Context, userID int) (userclient.ShopInventory, error)
	ShopPurchase(ctx context.Context, userID int, offerID int64, idempotencyKey string) (userclient.ShopPurchase, error)
	ShopEquip(ctx context.Context, userID int, slot string, siteID int64, itemID *int64) error
	Invalidate(ids ...int)
}

var _ upstream = (*userclient.Client)(nil)

type Service struct {
	shop          upstream
	mirrorBalance func(userID, balance int) error
	now           func() time.Time

	mu      sync.Mutex
	front   *userclient.ShopStorefront
	expires time.Time
}

func New(shop upstream, mirrorBalance func(userID, balance int) error) *Service {
	return &Service{shop: shop, mirrorBalance: mirrorBalance, now: time.Now}
}

func (s *Service) ready() *problem.Problem {
	if s == nil || s.shop == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Service) storefront(ctx context.Context) (*userclient.ShopStorefront, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if s.front != nil && now.Before(s.expires) {
		return s.front, nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), storefrontDeadline)
	defer cancel()
	fresh, err := s.shop.ShopStorefront(ctx)
	if err != nil {
		if s.front == nil {
			return nil, err
		}
		slog.Warn("shop storefront: refresh failed, serving the last good one", "err", err)
		s.expires = now.Add(storefrontTTL)
		return s.front, nil
	}
	s.front, s.expires = &fresh, now.Add(storefrontTTL)
	return s.front, nil
}

func caller(ctx context.Context) (int, *problem.Problem) {
	user := v1.User(ctx)
	if user == nil {
		return 0, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	return user.ID, nil
}

func (s *Service) listShopOffers(ctx context.Context, _ *struct{}) (*listShopOffersOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	front, err := s.storefront(ctx)
	if err != nil {
		return nil, upstreamProblem(err)
	}
	return &listShopOffersOutput{Body: repr.NewList(mapOffers(front.Offers), nil)}, nil
}

func (s *Service) getMyShopInventory(ctx context.Context, _ *struct{}) (*getMyShopInventoryOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	userID, p := caller(ctx)
	if p != nil {
		return nil, p
	}
	front, err := s.storefront(ctx)
	if err != nil {
		return nil, upstreamProblem(err)
	}
	inv, err := s.shop.ShopInventory(ctx, userID)
	if err != nil {
		return nil, upstreamProblem(err)
	}
	return &getMyShopInventoryOutput{Body: mapInventory(inv, front.Site.ID)}, nil
}

func (s *Service) createShopOrder(ctx context.Context, in *createShopOrderInput) (*createShopOrderOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	userID, p := caller(ctx)
	if p != nil {
		return nil, p
	}
	offerID, ok := parseID(in.Body.OfferID)
	if !ok {
		return nil, validationFailed(problem.AtPointer("/offer_id", problem.ReasonUnknownReference, "no offer has this id", nil))
	}
	key := upstreamKeyPrefix + idempotencyKey(ctx)
	bought, err := s.shop.ShopPurchase(ctx, userID, offerID, key)
	if err != nil {
		return nil, s.purchaseProblem(ctx, err, offerID)
	}
	s.shop.Invalidate(userID)
	if s.mirrorBalance != nil {
		if err := s.mirrorBalance(userID, int(bought.Balance)); err != nil {
			slog.Warn("shop: moemoepoint cache mirror failed", "user_id", userID, "err", err)
		}
	}
	return &createShopOrderOutput{Body: ShopPurchase{
		Object:      "shop_purchase",
		Order:       mapOrder(bought.Order),
		Moemoepoint: int(bought.Balance),
	}}, nil
}

func (s *Service) purchaseProblem(ctx context.Context, err error, offerID int64) *problem.Problem {
	p := upstreamProblem(err)
	if p.Code != problem.CodeMoemoepointInsufficient {
		return p
	}
	if front, ferr := s.storefront(ctx); ferr == nil {
		for _, o := range front.Offers {
			if o.ID == offerID {
				p.SetExtension("required", int(o.Price))
			}
		}
	}
	return p
}

func (s *Service) setShopLoadoutSlot(ctx context.Context, in *setShopLoadoutSlotInput) (*shopLoadoutSlotOutput, error) {
	itemID, ok := parseID(in.Body.ItemID)
	if !ok {
		return nil, validationFailed(problem.AtPointer("/item_id", problem.ReasonUnknownReference, "no item has this id", nil))
	}
	return s.equip(ctx, in.Scope, in.Slot, &itemID)
}

func (s *Service) clearShopLoadoutSlot(ctx context.Context, in *clearShopLoadoutSlotInput) (*shopLoadoutSlotOutput, error) {
	return s.equip(ctx, in.Scope, in.Slot, nil)
}

func (s *Service) equip(ctx context.Context, scope, slot string, itemID *int64) (*shopLoadoutSlotOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	userID, p := caller(ctx)
	if p != nil {
		return nil, p
	}
	var siteID int64
	if scope == scopeThisSite {
		front, err := s.storefront(ctx)
		if err != nil {
			return nil, upstreamProblem(err)
		}
		siteID = front.Site.ID
	}
	if err := s.shop.ShopEquip(ctx, userID, slot, siteID, itemID); err != nil {
		return nil, equipProblem(err)
	}
	s.shop.Invalidate(userID)
	out := ShopLoadoutSlot{Object: "shop_loadout_slot", Scope: scope, Slot: slot}
	if itemID != nil {
		worn := repr.DecimalID(strconv.FormatInt(*itemID, 10))
		out.WornItemID = &worn
	}
	return &shopLoadoutSlotOutput{Body: out}, nil
}

func parseID(id repr.DecimalID) (int64, bool) {
	n, err := strconv.ParseInt(string(id), 10, 64)
	return n, err == nil && n > 0
}
