package apiv1

import (
	"maps"
	"slices"
	"strconv"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/userclient"
)

func id64(n int64) repr.DecimalID {
	return repr.DecimalID(strconv.FormatInt(n, 10))
}

func mapItem(it userclient.ShopItem) (ShopItem, bool) {
	if !itemTypes[it.Kind] {
		return ShopItem{}, false
	}
	out := ShopItem{Object: "shop_item", ID: id64(it.ID), ItemType: it.Kind, DisplayName: it.Name, Description: it.Description}
	if d := it.Preview; d != nil && d.StaticURL != "" {
		out.Artwork = &ShopItemArtwork{StaticURL: d.StaticURL}
		if d.AnimatedURL != "" {
			animated := d.AnimatedURL
			out.Artwork.AnimatedURL = &animated
		}
	}
	return out, true
}

func mapRewards(in []userclient.ShopReward) ([]ShopReward, bool) {
	out := make([]ShopReward, 0, len(in))
	complete := true
	for _, r := range in {
		item, ok := mapItem(r.Item)
		if !ok {
			complete = false
			continue
		}
		reward := ShopReward{Item: item}
		if r.DurationDays > 0 {
			days := r.DurationDays
			reward.DurationDays = &days
		}
		out = append(out, reward)
	}
	return out, complete
}

func mapOffers(in []userclient.ShopOffer) []ShopOffer {
	out := make([]ShopOffer, 0, len(in))
	for _, o := range in {
		rewards, complete := mapRewards(o.Rewards)
		if !complete || len(rewards) == 0 {
			continue
		}
		offer := ShopOffer{
			Object:          "shop_offer",
			ID:              id64(o.ID),
			Price:           int(o.Price),
			IsSiteExclusive: o.SiteID != nil,
			Rewards:         rewards,
			StockCount:      o.Stock,
			RemainingCount:  o.Remaining,
			EndsAt:          repr.TimestampPtr(o.EndsAt),
		}
		if o.PerUserLimit > 0 {
			period := "lifetime"
			if o.LimitPeriod == "month" {
				period = "month"
			}
			offer.PurchaseLimit = &ShopPurchaseLimit{Quantity: o.PerUserLimit, Period: period}
		}
		out = append(out, offer)
	}
	return out
}

func mapOrder(o userclient.ShopOrder) ShopOrder {
	rewards, _ := mapRewards(o.Rewards)
	codes := make([]ShopRedeemCode, 0, len(o.Codes))
	for _, c := range o.Codes {
		code := ShopRedeemCode{Code: c.Code}
		if c.ExpiresOn != nil {
			day := repr.CalendarDate(*c.ExpiresOn)
			code.ExpiryDate = &day
		}
		codes = append(codes, code)
	}
	state := "completed"
	if o.Status == "refunded" {
		state = "refunded"
	}
	return ShopOrder{
		Object:      "shop_order",
		ID:          id64(o.ID),
		OfferID:     id64(o.OfferID),
		Price:       int(o.Price),
		State:       state,
		Rewards:     rewards,
		RedeemCodes: codes,
		CreatedAt:   repr.Timestamp(o.CreatedAt),
		RefundedAt:  repr.TimestampPtr(o.RefundedAt),
	}
}

func mapInventory(inv userclient.ShopInventory, site int64) ShopInventory {
	out := ShopInventory{
		Object:      "shop_inventory",
		Moemoepoint: int(inv.Balance),
		Items:       make([]ShopOwnedItem, 0, len(inv.Items)),
		Loadout:     make([]ShopLoadoutSlot, 0, len(scopes)*len(slots)),
		Orders:      make([]ShopOrder, 0, len(inv.Orders)),
		LimitUsage:  make([]ShopLimitUsage, 0, len(inv.LimitUsed)),
	}
	active := map[int64]bool{}
	for _, owned := range inv.Items {
		item, ok := mapItem(owned.Item)
		if !ok {
			continue
		}
		via := "purchase"
		if owned.Source == "grant" {
			via = "grant"
		}
		out.Items = append(out.Items, ShopOwnedItem{
			Item:        item,
			AcquiredVia: via,
			AcquiredAt:  repr.Timestamp(owned.AcquiredAt),
			ExpiresAt:   repr.TimestampPtr(owned.ExpiresAt),
			IsActive:    owned.Active,
		})
		if owned.Active {
			active[owned.Item.ID] = true
		}
	}
	for _, scope := range scopes {
		want := int64(0)
		if scope == scopeThisSite {
			want = site
		}
		for _, slot := range slots {
			entry := ShopLoadoutSlot{Object: "shop_loadout_slot", Scope: scope, Slot: slot}
			for _, l := range inv.Loadout {
				if l.SiteID == want && l.Slot == slot && active[l.ItemID] {
					worn := id64(l.ItemID)
					entry.WornItemID = &worn
				}
			}
			out.Loadout = append(out.Loadout, entry)
		}
	}
	for _, o := range inv.Orders {
		out.Orders = append(out.Orders, mapOrder(o))
	}
	for _, offerID := range slices.Sorted(maps.Keys(inv.LimitUsed)) {
		out.LimitUsage = append(out.LimitUsage, ShopLimitUsage{OfferID: id64(offerID), PurchasedCount: inv.LimitUsed[offerID]})
	}
	return out
}
