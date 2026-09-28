package app

import (
	"context"
	"net/http"
	"testing"
)

func offerByID(t *testing.T, body map[string]any, id string) map[string]any {
	t.Helper()
	items, _ := body["items"].([]any)
	for _, it := range items {
		m, _ := it.(map[string]any)
		if m["id"] == id {
			return m
		}
	}
	return nil
}

func TestV1ShopOffersArePublic(t *testing.T) {
	f := newShopFix(t)
	resp, body := f.call(t, http.MethodGet, shopPath, "/shop/offers", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if n := len(body["items"].([]any)); n != 3 {
		t.Fatalf("%d offers, want 3", n)
	}
}

func TestV1ShopOfferShape(t *testing.T) {
	f := newShopFix(t)
	_, body := f.call(t, http.MethodGet, shopPath, "/shop/offers", "", "", nil)

	about := offerByID(t, body, "6")
	if about["purchase_limit"] != nil || about["remaining_count"] != nil || about["is_site_exclusive"] != false {
		t.Fatalf("unlimited offer %+v", about)
	}
	reward := about["rewards"].([]any)[0].(map[string]any)
	if reward["duration_days"] != nil {
		t.Fatalf("permanent reward duration %v, want null", reward["duration_days"])
	}
	item := reward["item"].(map[string]any)
	if item["item_type"] != "profile_about" || item["artwork"] != nil || item["display_name"] != "主页介绍" {
		t.Fatalf("perk item %+v", item)
	}

	coupon := offerByID(t, body, "7")
	limit, _ := coupon["purchase_limit"].(map[string]any)
	if asInt(limit["quantity"]) != 2 || limit["period"] != "month" || asInt(coupon["remaining_count"]) != 21 {
		t.Fatalf("limited offer %+v", coupon)
	}

	frame := offerByID(t, body, "9")
	if frame["is_site_exclusive"] != true || asInt(frame["stock_count"]) != 50 || asInt(frame["remaining_count"]) != 0 {
		t.Fatalf("zone offer %+v", frame)
	}
	if frame["ends_at"] != "2026-10-31T15:59:59Z" {
		t.Fatalf("ends_at %v", frame["ends_at"])
	}
	fr := frame["rewards"].([]any)[0].(map[string]any)
	art := fr["item"].(map[string]any)["artwork"].(map[string]any)
	if asInt(fr["duration_days"]) != 30 || art["static_url"] != "https://image.test.example/decorations/9.png" ||
		art["animated_url"] != "https://image.test.example/decorations/9.webp" {
		t.Fatalf("frame reward %+v", fr)
	}
}

func TestV1ShopOffersDropUnknownItemTypes(t *testing.T) {
	f := newShopFix(t)
	offers := fakeShopOffers()
	offers = append(offers, shopOffer(12, 200, nil,
		map[string]any{"item": shopItem(12, "avatar_frame", "框")},
		map[string]any{"item": shopItem(13, "nameplate", "铭牌")},
	))
	f.setOffers(offers)
	_, body := f.call(t, http.MethodGet, shopPath, "/shop/offers", "", "", nil)
	if offerByID(t, body, "12") != nil {
		t.Fatal("an offer holding an unknown item type was listed")
	}
	if len(body["items"].([]any)) != 3 {
		t.Fatalf("offers %+v", body["items"])
	}
}

func TestV1ShopOffersServeLastGoodListWhileUpstreamIsDown(t *testing.T) {
	f := newShopFix(t)
	f.frontHTTP.Store(http.StatusBadGateway)
	resp, body := f.call(t, http.MethodGet, shopPath, "/shop/offers", "", "", nil)
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")

	f.frontHTTP.Store(0)
	resp, _ = f.call(t, http.MethodGet, shopPath, "/shop/offers", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d after recovery", resp.StatusCode)
	}
	n := f.nFront.Load()
	resp, _ = f.call(t, http.MethodGet, shopPath, "/shop/offers", "", "", nil)
	if resp.StatusCode != http.StatusOK || f.nFront.Load() != n {
		t.Fatalf("second read within a minute refetched (%d → %d)", n, f.nFront.Load())
	}
}

func TestV1ShopInventoryNeedsCredential(t *testing.T) {
	f := newShopFix(t)
	resp, body := f.call(t, http.MethodGet, myShopPath, "/me/shop", "", "", nil)
	mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
	if f.nInventory.Load() != 0 {
		t.Fatal("an anonymous request reached the account service")
	}
}

func TestV1ShopInventoryUpstreamDownIs503(t *testing.T) {
	f := newShopFix(t)
	f.invHTTP.Store(http.StatusInternalServerError)
	resp, body := f.call(t, http.MethodGet, myShopPath, "/me/shop", "sess-alice", "", nil)
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1ShopInventoryShape(t *testing.T) {
	f := newShopFix(t)
	resp, body := f.call(t, http.MethodGet, myShopPath, "/me/shop", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if asInt(body["moemoepoint"]) != 500 {
		t.Fatalf("moemoepoint %v", body["moemoepoint"])
	}
	items := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items %+v, want the nameplate left out", items)
	}
	lapsed := items[1].(map[string]any)
	if lapsed["is_active"] != false || lapsed["acquired_via"] != "grant" || lapsed["expires_at"] != "2026-09-01T00:00:00Z" {
		t.Fatalf("lapsed item %+v", lapsed)
	}
	order := body["orders"].([]any)[0].(map[string]any)
	code := order["redeem_codes"].([]any)[0].(map[string]any)
	if code["code"] != "ABCD-EFGH-IJKL" || code["expiry_date"] != "2026-10-31" || order["state"] != "completed" {
		t.Fatalf("order %+v", order)
	}
	usage := body["limit_usage"].([]any)
	first, second := usage[0].(map[string]any), usage[1].(map[string]any)
	if first["offer_id"] != "7" || asInt(first["purchased_count"]) != 1 || second["offer_id"] != "12" {
		t.Fatalf("limit_usage %+v", usage)
	}
}

func TestV1ShopInventoryWornItemMustBeActive(t *testing.T) {
	f := newShopFix(t)
	_, body := f.call(t, http.MethodGet, myShopPath, "/me/shop", "sess-alice", "", nil)
	loadout := body["loadout"].([]any)
	if len(loadout) != 4 {
		t.Fatalf("loadout %+v, want all four scope and slot pairs", loadout)
	}
	worn := map[string]any{}
	for _, l := range loadout {
		m := l.(map[string]any)
		worn[m["scope"].(string)+"/"+m["slot"].(string)] = m["worn_item_id"]
	}
	want := map[string]any{
		"everywhere/avatar_frame": "9", "everywhere/profile_background": nil,
		"this_site/avatar_frame": nil, "this_site/profile_background": nil,
	}
	for k, v := range want {
		if worn[k] != v {
			t.Errorf("%s worn %v, want %v", k, worn[k], v)
		}
	}
}

func TestV1ShopOrderNeedsIdempotencyKey(t *testing.T) {
	f := newShopFix(t)
	resp, body := f.call(t, http.MethodPost, myShopPath+"/orders", "/me/shop/orders", "sess-alice", "", map[string]any{"offer_id": "7"})
	mustCode(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	if len(f.purchases) != 0 {
		t.Fatal("a purchase without a key reached the account service")
	}
}

func TestV1ShopOrderUsesTheCaller(t *testing.T) {
	f := newShopFix(t)
	resp, body := f.call(t, http.MethodPost, myShopPath+"/orders", "/me/shop/orders", "sess-alice", keyUUID(1), map[string]any{"offer_id": "7"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if got := f.lastPurchase(t); got.userID != w3UserAlice || got.offerID != 7 {
		t.Fatalf("purchase %+v, want alice buying offer 7", got)
	}
	order := body["order"].(map[string]any)
	if asInt(body["moemoepoint"]) != 400 || order["id"] != "88" || order["offer_id"] != "7" {
		t.Fatalf("purchase %+v", body)
	}
	if order["redeem_codes"].([]any)[0].(map[string]any)["code"] != "ABCD-EFGH-IJKL" {
		t.Fatalf("codes %+v", order["redeem_codes"])
	}
}

func TestV1ShopOrderForwardsPrefixedKey(t *testing.T) {
	f := newShopFix(t)
	f.call(t, http.MethodPost, myShopPath+"/orders", "/me/shop/orders", "sess-alice", keyUUID(2), map[string]any{"offer_id": "7"})
	if got := f.lastPurchase(t).key; got != "kungal:"+keyUUID(2) {
		t.Fatalf("upstream key %q", got)
	}
}

func TestV1ShopOrderReplaysWithoutChargingTwice(t *testing.T) {
	f := newShopFix(t)
	body := map[string]any{"offer_id": "7"}
	f.call(t, http.MethodPost, myShopPath+"/orders", "/me/shop/orders", "sess-alice", keyUUID(3), body)
	resp, again := f.call(t, http.MethodPost, myShopPath+"/orders", "/me/shop/orders", "sess-alice", keyUUID(3), body)
	if resp.StatusCode != http.StatusCreated || resp.Header.Get("Idempotency-Replayed") != "true" {
		t.Fatalf("%d replayed=%q %+v", resp.StatusCode, resp.Header.Get("Idempotency-Replayed"), again)
	}
	if len(f.purchases) != 1 {
		t.Fatalf("%d purchases reached the account service, want 1", len(f.purchases))
	}
}

func TestV1ShopOrderMirrorsBalance(t *testing.T) {
	f := newShopFix(t)
	f.call(t, http.MethodPost, myShopPath+"/orders", "/me/shop/orders", "sess-alice", keyUUID(4), map[string]any{"offer_id": "7"})
	if got := f.scalar(t, `SELECT moemoepoint FROM kungal_user_state WHERE user_id = ?`, w3UserAlice); got != 400 {
		t.Fatalf("cached moemoepoint %d, want 400", got)
	}
}

func TestV1ShopOrderErrorMapping(t *testing.T) {
	cases := []struct {
		upstream int
		status   int
		code     string
	}{
		{19002, http.StatusConflict, "SHOP_OFFER_UNAVAILABLE"},
		{19003, http.StatusConflict, "ALREADY_EXISTS"},
		{19004, http.StatusConflict, "SHOP_PURCHASE_LIMIT_REACHED"},
		{19005, http.StatusConflict, "SHOP_OFFER_SOLD_OUT"},
		{16006, http.StatusForbidden, "MOEMOEPOINT_INSUFFICIENT"},
		{19014, http.StatusConflict, "IDEMPOTENCY_KEY_REUSED"},
		{19017, http.StatusInternalServerError, "INTERNAL_ERROR"},
		{10, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
	}
	f := newShopFix(t)
	for i, c := range cases {
		f.buyCode.Store(int32(c.upstream))
		resp, body := f.call(t, http.MethodPost, myShopPath+"/orders", "/me/shop/orders", "sess-alice", keyUUID(100+i), map[string]any{"offer_id": "9"})
		mustCode(t, resp, body, c.status, c.code)
		if c.upstream == 16006 && asInt(body["required"]) != 300 {
			t.Fatalf("MOEMOEPOINT_INSUFFICIENT required %v, want the offer's price 300", body["required"])
		}
	}
	if got := f.scalar(t, `SELECT moemoepoint FROM kungal_user_state WHERE user_id = ?`, w3UserAlice); got != 30 {
		t.Fatalf("a refused purchase moved the cached balance to %d", got)
	}
}

func TestV1ShopOrderRefusesANonID(t *testing.T) {
	f := newShopFix(t)
	resp, body := f.call(t, http.MethodPost, myShopPath+"/orders", "/me/shop/orders", "sess-alice", keyUUID(5), map[string]any{"offer_id": "0"})
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	fieldErr(t, body, "pointer", "/offer_id", "UNKNOWN_REFERENCE")
}

func TestV1ShopLoadoutThisSiteUsesStorefrontSite(t *testing.T) {
	f := newShopFix(t)
	resp, body := f.call(t, http.MethodPut, myShopPath+"/loadout/this_site/avatar_frame", "/me/shop/loadout/{scope}/{slot}", "sess-alice", "", map[string]any{"item_id": "9"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	got := f.lastEquip(t)
	if got.userID != w3UserAlice || got.siteID != shopSiteID || got.slot != "avatar_frame" || got.itemID == nil || *got.itemID != 9 {
		t.Fatalf("equip %+v", got)
	}
	if body["scope"] != "this_site" || body["slot"] != "avatar_frame" || body["worn_item_id"] != "9" {
		t.Fatalf("slot %+v", body)
	}

	resp, body = f.call(t, http.MethodDelete, myShopPath+"/loadout/everywhere/profile_background", "/me/shop/loadout/{scope}/{slot}", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || body["worn_item_id"] != nil {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	got = f.lastEquip(t)
	if got.siteID != 0 || got.slot != "profile_background" || got.itemID != nil {
		t.Fatalf("clear %+v", got)
	}
}

func TestV1ShopLoadoutErrorMapping(t *testing.T) {
	f := newShopFix(t)
	put := func() (*http.Response, map[string]any) {
		return f.call(t, http.MethodPut, myShopPath+"/loadout/everywhere/avatar_frame", "/me/shop/loadout/{scope}/{slot}", "sess-alice", "", map[string]any{"item_id": "10"})
	}
	f.equipCode.Store(19006)
	resp, body := put()
	mustCode(t, resp, body, http.StatusConflict, "SHOP_ITEM_NOT_OWNED")
	f.equipCode.Store(19001)
	resp, body = put()
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	fieldErr(t, body, "pointer", "/item_id", "UNKNOWN_REFERENCE")
	f.equipCode.Store(19008)
	resp, body = put()
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	fieldErr(t, body, "pointer", "/item_id", "INCONSISTENT_WITH")
	f.equipCode.Store(10)
	resp, body = put()
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	resp, body = f.call(t, http.MethodDelete, myShopPath+"/loadout/this_site/avatar_frame", "/me/shop/loadout/{scope}/{slot}", "sess-alice", "", nil)
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1ShopLoadoutRefusesUnknownSlot(t *testing.T) {
	f := newShopFix(t)
	resp, body := f.call(t, http.MethodPut, myShopPath+"/loadout/everywhere/nameplate", "/me/shop/loadout/{scope}/{slot}", "sess-alice", "", map[string]any{"item_id": "9"})
	mustCode(t, resp, body, http.StatusBadRequest, "UNKNOWN_ENUM_VALUE")
	if len(f.equips) != 0 {
		t.Fatal("an unknown slot reached the account service")
	}
}

func TestV1ShopLoadoutInvalidatesUserCache(t *testing.T) {
	f := newShopFix(t)
	ctx := context.Background()
	if _, err := f.UserClient.Users(ctx, []int{w3UserAlice}); err != nil {
		t.Fatal(err)
	}
	warm := f.nBatch.Load()
	if _, err := f.UserClient.Users(ctx, []int{w3UserAlice}); err != nil || f.nBatch.Load() != warm {
		t.Fatal("the profile was not cached")
	}
	f.call(t, http.MethodPut, myShopPath+"/loadout/everywhere/avatar_frame", "/me/shop/loadout/{scope}/{slot}", "sess-alice", "", map[string]any{"item_id": "9"})
	if _, err := f.UserClient.Users(ctx, []int{w3UserAlice}); err != nil {
		t.Fatal(err)
	}
	if f.nBatch.Load() == warm {
		t.Fatal("wearing an item left the caller's cached profile, and its old frame, in place")
	}

	warm = f.nBatch.Load()
	f.call(t, http.MethodPost, myShopPath+"/orders", "/me/shop/orders", "sess-alice", keyUUID(6), map[string]any{"offer_id": "6"})
	if _, err := f.UserClient.Users(ctx, []int{w3UserAlice}); err != nil {
		t.Fatal(err)
	}
	if f.nBatch.Load() == warm {
		t.Fatal("a purchase left the caller's cached profile in place")
	}
}
