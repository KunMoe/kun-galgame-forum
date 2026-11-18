package storelink

import (
	"testing"

	"kun-galgame-api/pkg/storeclient"
)

const tmpl = "https://dlaf.jp/soft/dlaf/=/t/s/link/work/aid/kungal/id/{workno}.html"

// galgame 4 is the first row of pkg/dlsite's verified whitelist (VJ013550), so
// it resolves a workno with no refs value.
const whitelistedGalgame = 4

func TestResolve_FallsBackToTemplateAndEnqueues(t *testing.T) {
	r := New(Options{LinkTemplate: Fixed(tmpl), StaticCoupon: Fixed("https://coupon.example")})

	got := r.Resolve(whitelistedGalgame, "")
	if got.PurchaseURL != "https://dlaf.jp/soft/dlaf/=/t/s/link/work/aid/kungal/id/VJ013550.html" {
		t.Errorf("purchase = %q, want the bare affiliate template", got.PurchaseURL)
	}
	if got.CouponURL != "https://coupon.example" || got.CampaignName != "" {
		t.Errorf("static coupon must carry no campaign name, got %+v", got)
	}
}

func TestResolve_PrefersTheMintedShortLink(t *testing.T) {
	r := New(Options{LinkTemplate: Fixed(tmpl)})
	r.remember("RJ297925", "https://s.example/s/abc")

	if got := r.Resolve(10, ""); got.PurchaseURL != "https://s.example/s/abc" {
		t.Errorf("purchase = %q, want the short link", got.PurchaseURL)
	}
}

func TestResolve_RefsWinsOverTheWhitelist(t *testing.T) {
	r := New(Options{LinkTemplate: Fixed(tmpl)})
	got := r.Resolve(whitelistedGalgame, "RJ297925")
	if got.PurchaseURL != "https://dlaf.jp/soft/dlaf/=/t/s/link/work/aid/kungal/id/RJ297925.html" {
		t.Errorf("purchase = %q, want catalog's refs.dlsite to win", got.PurchaseURL)
	}
}

func TestResolve_RunningCampaignBeatsTheStaticCoupon(t *testing.T) {
	r := New(Options{LinkTemplate: Fixed(tmpl), StaticCoupon: Fixed("https://coupon.example")})
	r.setCampaign(&storeclient.Campaign{ID: "7", Name: "夏日特惠"}, "https://s.example/s/def")

	got := r.Resolve(whitelistedGalgame, "")
	if got.CouponURL != "https://s.example/s/def" || got.CampaignName != "夏日特惠" {
		t.Errorf("got %+v, want the campaign's link and name", got)
	}

	// An ended campaign has to hand the standing landing page back, not leave the
	// dead campaign link on the button.
	r.setCampaign(nil, "")
	if got := r.Resolve(whitelistedGalgame, ""); got.CouponURL != "https://coupon.example" || got.CampaignName != "" {
		t.Errorf("after the campaign ends: %+v", got)
	}
}

func TestResolve_NoWorknoAndNoTemplateRenderNothing(t *testing.T) {
	r := New(Options{LinkTemplate: Fixed(tmpl)})
	if got := r.Resolve(999999999, ""); got != (Links{}) {
		t.Errorf("a galgame with no workno must render nothing, got %+v", got)
	}

	// A coupon with no purchase link is not a usable 补票提示: the frontend hangs
	// the coupon inside the purchase popover.
	bare := New(Options{StaticCoupon: Fixed("https://coupon.example")})
	if got := bare.Resolve(whitelistedGalgame, ""); got != (Links{}) {
		t.Errorf("no template and no short link must render nothing, got %+v", got)
	}
}

func TestResolve_NilResolverIsSafe(t *testing.T) {
	var r *Resolver
	if got := r.Resolve(whitelistedGalgame, "RJ297925"); got != (Links{}) {
		t.Errorf("nil resolver = %+v", got)
	}
	if r.Configured() {
		t.Error("nil resolver must report unconfigured")
	}
}

func TestEnqueue_IsANoOpWithoutTheStoreFace(t *testing.T) {
	r := New(Options{LinkTemplate: Fixed(tmpl)})
	r.enqueue("RJ297925")
	if len(r.queue) != 0 {
		t.Errorf("queued %d worknos with no store client configured", len(r.queue))
	}
}
