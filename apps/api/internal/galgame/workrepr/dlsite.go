package workrepr

import "kun-galgame-api/internal/infrastructure/storelink"

type DlsiteOffer struct {
	PurchaseURL  string  `json:"purchase_url" format:"uri" maxLength:"4096" doc:"Short link or affiliate template for the DLsite product."`
	CouponURL    *string `json:"coupon_url" format:"uri" maxLength:"4096" doc:"Coupon or campaign landing URL. null when none."`
	CampaignName *string `json:"campaign_name" maxLength:"256" doc:"Name of a running campaign. null on the static coupon page. Free text; never use it as a decision input."`
}

func DlsiteOf(links storelink.Links) *DlsiteOffer {
	if links.PurchaseURL == "" {
		return nil
	}
	out := &DlsiteOffer{PurchaseURL: links.PurchaseURL}
	if links.CouponURL != "" {
		c := links.CouponURL
		out.CouponURL = &c
	}
	if links.CampaignName != "" {
		n := links.CampaignName
		out.CampaignName = &n
	}
	return out
}
