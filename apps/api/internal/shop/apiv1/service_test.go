package apiv1

import (
	"context"
	"errors"
	"testing"
	"time"

	"kun-galgame-api/pkg/userclient"
)

type fakeShop struct {
	fronts int
	fail   bool
}

func (f *fakeShop) ShopStorefront(context.Context) (userclient.ShopStorefront, error) {
	f.fronts++
	if f.fail {
		return userclient.ShopStorefront{}, errors.New("down")
	}
	return userclient.ShopStorefront{Site: userclient.ShopSite{ID: 3}}, nil
}

func (f *fakeShop) ShopInventory(context.Context, int) (userclient.ShopInventory, error) {
	return userclient.ShopInventory{}, nil
}

func (f *fakeShop) ShopPurchase(context.Context, int, int64, string) (userclient.ShopPurchase, error) {
	return userclient.ShopPurchase{}, nil
}

func (f *fakeShop) ShopEquip(context.Context, int, string, int64, *int64) error { return nil }

func (f *fakeShop) Invalidate(...int) {}

func TestV1ShopOffersCacheExpires(t *testing.T) {
	up := &fakeShop{}
	s := New(up, nil)
	now := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	ctx := context.Background()

	for range 2 {
		if _, err := s.storefront(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if up.fronts != 1 {
		t.Fatalf("%d fetches inside the minute, want 1", up.fronts)
	}
	now = now.Add(storefrontTTL)
	if _, err := s.storefront(ctx); err != nil {
		t.Fatal(err)
	}
	if up.fronts != 2 {
		t.Fatalf("%d fetches after the minute, want 2", up.fronts)
	}

	up.fail = true
	now = now.Add(storefrontTTL)
	front, err := s.storefront(ctx)
	if err != nil || front.Site.ID != 3 {
		t.Fatalf("a failed refresh did not serve the last storefront: %v %+v", err, front)
	}
}
