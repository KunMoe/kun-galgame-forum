package app

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

const (
	shopPath   = "/api/v1/shop/offers"
	myShopPath = "/api/v1/me/shop"
	shopSiteID = 3
)

type shopPurchaseCall struct {
	userID  int
	offerID int64
	key     string
}

type shopEquipCall struct {
	userID int
	slot   string
	siteID int64
	itemID *int64
}

type shopFix struct {
	*writeFix
	mu          sync.Mutex
	front       map[string]any
	inventory   map[string]any
	frontHTTP   atomic.Int32
	nFront      atomic.Int32
	buyCode     atomic.Int32
	equipCode   atomic.Int32
	balance     atomic.Int32
	purchases   []shopPurchaseCall
	equips      []shopEquipCall
	nInventory  atomic.Int32
	invHTTP     atomic.Int32
	orderRecord map[string]any
}

func newShopFix(t *testing.T) *shopFix {
	t.Helper()
	f := &shopFix{writeFix: newWriteFix(t, nil)}
	f.balance.Store(500)
	f.front = map[string]any{
		"site":   map[string]any{"id": shopSiteID, "name": "KUN Galgame 论坛", "domain": "www.kungal.com"},
		"offers": fakeShopOffers(),
	}
	f.inventory = fakeShopInventory()
	f.orderRecord = fakeShopOrder(88, 7, 100)
	f.installShop()
	f.alice(t)
	return f
}

func (f *shopFix) installShop() {
	f.mux.HandleFunc("GET /shop/storefront", func(w http.ResponseWriter, _ *http.Request) {
		f.nFront.Add(1)
		if status := int(f.frontHTTP.Load()); status != 0 {
			w.WriteHeader(status)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		writeHouse(w, 200, 0, f.front)
	})
	f.mux.HandleFunc("GET /users/{id}/shop", func(w http.ResponseWriter, _ *http.Request) {
		f.nInventory.Add(1)
		if status := int(f.invHTTP.Load()); status != 0 {
			w.WriteHeader(status)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		inv := map[string]any{"balance": int(f.balance.Load())}
		for k, v := range f.inventory {
			inv[k] = v
		}
		writeHouse(w, 200, 0, inv)
	})
	f.mux.HandleFunc("POST /users/{id}/shop/orders", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			OfferID        int64  `json:"offer_id"`
			IdempotencyKey string `json:"idempotency_key"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		uid, _ := strconv.Atoi(r.PathValue("id"))
		f.mu.Lock()
		f.purchases = append(f.purchases, shopPurchaseCall{uid, body.OfferID, body.IdempotencyKey})
		order := f.orderRecord
		f.mu.Unlock()
		if code := int(f.buyCode.Load()); code != 0 {
			status := 400
			if code == 19017 {
				status = 403
			}
			writeHouse(w, status, code, nil)
			return
		}
		bal := f.balance.Add(-100)
		writeHouse(w, 200, 0, map[string]any{"order": order, "balance": int(bal), "replay": false})
	})
	f.mux.HandleFunc("PUT /users/{id}/shop/loadout", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Slot   string `json:"slot"`
			SiteID int64  `json:"site_id"`
			ItemID *int64 `json:"item_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		uid, _ := strconv.Atoi(r.PathValue("id"))
		f.mu.Lock()
		f.equips = append(f.equips, shopEquipCall{uid, body.Slot, body.SiteID, body.ItemID})
		f.mu.Unlock()
		if code := int(f.equipCode.Load()); code != 0 {
			writeHouse(w, 400, code, nil)
			return
		}
		writeHouse(w, 200, 0, map[string]any{"cosmetics": map[string]any{}})
	})
}

func (f *shopFix) setOffers(offers []map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.front["offers"] = offers
}

func (f *shopFix) setInventory(key string, v any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inventory[key] = v
}

func (f *shopFix) lastPurchase(t *testing.T) shopPurchaseCall {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.purchases) == 0 {
		t.Fatal("no purchase reached the account service")
	}
	return f.purchases[len(f.purchases)-1]
}

func (f *shopFix) lastEquip(t *testing.T) shopEquipCall {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.equips) == 0 {
		t.Fatal("no loadout change reached the account service")
	}
	return f.equips[len(f.equips)-1]
}

func (f *shopFix) call(t *testing.T, method, rawURL, spec, session, key string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, method, rawURL, session, spec, key, nil, payload)
	return resp, problemMap(t, body)
}

func shopItem(id int, kind, name string) map[string]any {
	it := map[string]any{
		"id": id, "kind": kind, "site_id": nil, "status": "published", "name": name,
		"description": name + " description", "render": map[string]any{},
	}
	if kind == "avatar_frame" || kind == "profile_background" {
		it["preview"] = map[string]any{
			"item_id": id, "name": name,
			"static_url":   "https://image.test.example/decorations/" + strconv.Itoa(id) + ".png",
			"animated_url": "https://image.test.example/decorations/" + strconv.Itoa(id) + ".webp",
		}
	}
	return it
}

func shopOffer(id int, price int, siteID any, rewards ...map[string]any) map[string]any {
	return map[string]any{
		"id": id, "site_id": siteID, "status": "active", "price": price,
		"costs":   []map[string]any{{"asset": "moemoepoint", "amount": price}},
		"rewards": rewards, "starts_at": nil, "ends_at": nil,
		"per_user_limit": 0, "limit_period": "", "stock": nil, "sold": 0, "remaining": nil,
		"sort_order": 0, "created_at": "2026-09-27T14:25:15.571078Z",
	}
}

func fakeShopOffers() []map[string]any {
	about := shopOffer(6, 100, nil, map[string]any{"item": shopItem(6, "profile_about", "主页介绍")})
	coupon := shopOffer(7, 100, nil, map[string]any{"item": shopItem(7, "redeem_code", "DLsite 1000 円优惠券")})
	coupon["per_user_limit"], coupon["limit_period"], coupon["remaining"] = 2, "month", 21
	frame := shopOffer(9, 300, shopSiteID, map[string]any{"item": shopItem(9, "avatar_frame", "樱花"), "duration_days": 30})
	frame["stock"], frame["sold"], frame["remaining"] = 50, 50, 0
	frame["ends_at"] = "2026-10-31T15:59:59.5Z"
	return []map[string]any{about, coupon, frame}
}

func fakeShopOrder(id, offerID, price int) map[string]any {
	return map[string]any{
		"id": id, "user_id": w3UserAlice, "recipient_user_id": w3UserAlice, "offer_id": offerID, "site_id": shopSiteID,
		"costs":       []map[string]any{{"asset": "moemoepoint", "amount": price}},
		"rewards":     []map[string]any{{"item": shopItem(offerID, "redeem_code", "DLsite 1000 円优惠券")}},
		"transfer_id": 5, "status": "completed", "refund_transfer_id": nil, "refunded_at": nil,
		"created_at": "2026-09-28T01:02:03.456Z", "price": price,
		"codes": []map[string]any{{"item_id": offerID, "code": "ABCD-EFGH-IJKL", "expires_on": "2026-10-31"}},
	}
}

func fakeShopInventory() map[string]any {
	return map[string]any{
		"items": []map[string]any{
			{"item": shopItem(9, "avatar_frame", "樱花"), "source": "purchase", "acquired_at": "2026-09-20T00:00:00Z", "expires_at": nil, "active": true},
			{"item": shopItem(10, "profile_background", "星空"), "source": "grant", "acquired_at": "2026-08-01T00:00:00Z", "expires_at": "2026-09-01T00:00:00Z", "active": false},
			{"item": shopItem(11, "nameplate", "铭牌"), "source": "purchase", "acquired_at": "2026-09-21T00:00:00Z", "expires_at": nil, "active": true},
		},
		"loadout": []map[string]any{
			{"user_id": w3UserAlice, "slot": "avatar_frame", "site_id": 0, "item_id": 9, "updated_at": "2026-09-20T00:00:00Z"},
			{"user_id": w3UserAlice, "slot": "profile_background", "site_id": shopSiteID, "item_id": 10, "updated_at": "2026-08-01T00:00:00Z"},
			{"user_id": w3UserAlice, "slot": "avatar_frame", "site_id": 5, "item_id": 9, "updated_at": "2026-09-20T00:00:00Z"},
		},
		"orders":     []map[string]any{fakeShopOrder(88, 7, 100)},
		"limit_used": map[string]any{"7": 1, "12": 2},
	}
}
