package apiv1

import "kun-galgame-api/internal/apiv1/repr"

const (
	itemAvatarFrame       = "avatar_frame"
	itemProfileBackground = "profile_background"
	itemProfileAbout      = "profile_about"
	itemRedeemCode        = "redeem_code"

	scopeEverywhere = "everywhere"
	scopeThisSite   = "this_site"
)

var (
	itemTypes = map[string]bool{
		itemAvatarFrame: true, itemProfileBackground: true, itemProfileAbout: true, itemRedeemCode: true,
	}
	slots  = []string{itemAvatarFrame, itemProfileBackground}
	scopes = []string{scopeEverywhere, scopeThisSite}
)

type ShopOffer struct {
	Object          string             `json:"object" enum:"shop_offer" maxLength:"10" doc:"Type discriminant. Always shop_offer."`
	ID              repr.DecimalID     `json:"id" doc:"Offer id in the NextMoe shop. JSON string of a decimal integer."`
	Price           int                `json:"price" minimum:"0" maximum:"1000000" doc:"Moemoepoint charged per purchase. The shop never takes real money."`
	IsSiteExclusive bool               `json:"is_site_exclusive" doc:"Whether this offer is sold only in this forum's own zone. Anything bought here still shows on every NextMoe site."`
	Rewards         []ShopReward       `json:"rewards" minItems:"1" maxItems:"10" doc:"What one purchase gives, in the offer's order. The first is the one to show."`
	PurchaseLimit   *ShopPurchaseLimit `json:"purchase_limit" doc:"How many times one user may buy this offer. null when unlimited. The caller's own count is in getMyShopInventory's limit_usage."`
	StockCount      *int               `json:"stock_count" minimum:"0" doc:"Total copies the offer was stocked with. null when it is not stocked; a redeem code offer is never stocked, its supply is remaining_count."`
	RemainingCount  *int               `json:"remaining_count" minimum:"0" doc:"Copies still for sale: stock left, or sellable codes left in a redeem code offer's pool. 0 is sold out. null when unlimited."`
	EndsAt          *repr.DateTime     `json:"ends_at" doc:"When the offer stops selling. null when it has no end."`
}

type ShopReward struct {
	Item         ShopItem `json:"item" doc:"The item given."`
	DurationDays *int     `json:"duration_days" minimum:"1" maximum:"3650" doc:"How long the item lasts; buying again while it lasts extends it. null when it is permanent, and always for a redeem code."`
}

type ShopItem struct {
	Object      string           `json:"object" enum:"shop_item" maxLength:"9" doc:"Type discriminant. Always shop_item."`
	ID          repr.DecimalID   `json:"id" doc:"Item id in the NextMoe shop. JSON string of a decimal integer."`
	ItemType    string           `json:"item_type" enum:"avatar_frame,profile_background,profile_about,redeem_code" maxLength:"18" doc:"What the item is: avatar_frame and profile_background are worn in the slot of the same name; profile_about unlocks writing a profile introduction in the account center; redeem_code issues one code per purchase. Closed: an item of a type this forum does not render is left out."`
	DisplayName string           `json:"display_name" maxLength:"64" doc:"The item's name. Free text; never use it as a decision input."`
	Description string           `json:"description" maxLength:"255" doc:"A sentence about the item. Empty string when there is none. Free text; never use it as a decision input."`
	Artwork     *ShopItemArtwork `json:"artwork" doc:"The artwork of an avatar_frame or profile_background. null for every other type."`
}

type ShopItemArtwork struct {
	StaticURL   string  `json:"static_url" format:"uri" maxLength:"512" doc:"Still image. An avatar frame is a square PNG 1.2 times the avatar, drawn centred over it; a profile background is a 2:1 to 4:1 banner, cropped to cover and centred. Content-addressed: the URL never changes content."`
	AnimatedURL *string `json:"animated_url" format:"uri" maxLength:"512" doc:"Animated WebP of the same size. Never play it under a reduced-motion preference. null when there is no animated version."`
}

type ShopPurchaseLimit struct {
	Quantity int    `json:"quantity" minimum:"1" doc:"Purchases allowed per user in one period. Refunded orders do not count."`
	Period   string `json:"period" enum:"lifetime,month" maxLength:"8" doc:"lifetime counts every purchase ever; month counts the current Asia/Shanghai calendar month, reset at 00:00 on the 1st."`
}

type ShopInventory struct {
	Object      string            `json:"object" enum:"shop_inventory" maxLength:"14" doc:"Type discriminant. Always shop_inventory."`
	Moemoepoint int               `json:"moemoepoint" minimum:"-2147483648" doc:"The caller's live moemoepoint balance from the account service. It can be negative."`
	Items       []ShopOwnedItem   `json:"items" doc:"Items the caller holds, newest first, including expired ones (is_active false). Revoked and refunded items are not listed."`
	Loadout     []ShopLoadoutSlot `json:"loadout" minItems:"4" maxItems:"4" doc:"Every scope and slot pair, always all four."`
	Orders      []ShopOrder       `json:"orders" maxItems:"50" doc:"The caller's latest 50 orders on any NextMoe site or the account center, newest first."`
	LimitUsage  []ShopLimitUsage  `json:"limit_usage" doc:"Purchases counted against each limited offer in its current period. An offer with a purchase_limit that is not listed has none."`
}

type ShopOwnedItem struct {
	Item        ShopItem       `json:"item" doc:"The item held."`
	AcquiredVia string         `json:"acquired_via" enum:"purchase,grant" maxLength:"8" doc:"purchase, or grant when staff gave it."`
	AcquiredAt  repr.DateTime  `json:"acquired_at" doc:"When the caller got the item, or got it back after it lapsed."`
	ExpiresAt   *repr.DateTime `json:"expires_at" doc:"When it lapses. null when it is permanent."`
	IsActive    bool           `json:"is_active" doc:"Whether it is still in effect. An expired item is kept here with false."`
}

type ShopLoadoutSlot struct {
	Object     string          `json:"object" enum:"shop_loadout_slot" maxLength:"17" doc:"Type discriminant. Always shop_loadout_slot."`
	Scope      string          `json:"scope" enum:"everywhere,this_site" maxLength:"10" doc:"everywhere is the caller's default on every NextMoe site; this_site is a choice for this forum only, which wins over the default here."`
	Slot       string          `json:"slot" enum:"avatar_frame,profile_background" maxLength:"18" doc:"The slot, named after the item type worn in it."`
	WornItemID *repr.DecimalID `json:"worn_item_id" doc:"The item worn in this slot and scope. null when nothing is, including when what was chosen has since expired. The forum caches profiles for up to ten minutes, so another reader can see a change that late."`
}

type ShopOrder struct {
	Object      string           `json:"object" enum:"shop_order" maxLength:"10" doc:"Type discriminant. Always shop_order."`
	ID          repr.DecimalID   `json:"id" doc:"Order id in the NextMoe shop. JSON string of a decimal integer."`
	OfferID     repr.DecimalID   `json:"offer_id" doc:"The offer bought."`
	Price       int              `json:"price" minimum:"0" maximum:"1000000" doc:"Moemoepoint charged, as it was when bought."`
	State       string           `json:"state" enum:"completed,refunded" maxLength:"9" doc:"completed, or refunded by staff. A redeem code order is never refunded."`
	Rewards     []ShopReward     `json:"rewards" doc:"What the order gave, as the offer was when bought. An item of a type this forum does not render is left out."`
	RedeemCodes []ShopRedeemCode `json:"redeem_codes" doc:"The redeem codes the order issued. Empty unless the offer sells a redeem_code. Only ever shown to the buyer."`
	CreatedAt   repr.DateTime    `json:"created_at" doc:"When the order was placed."`
	RefundedAt  *repr.DateTime   `json:"refunded_at" doc:"When it was refunded. null unless state is refunded."`
}

type ShopRedeemCode struct {
	Code       string             `json:"code" maxLength:"200" doc:"The code to redeem at the issuer. Show it only to its owner. Free text; never use it as a decision input."`
	ExpiryDate *repr.CalendarDate `json:"expiry_date" doc:"The last day, Japan time, the issuer accepts the code. null when it does not expire."`
}

type ShopLimitUsage struct {
	OfferID        repr.DecimalID `json:"offer_id" doc:"A limited offer."`
	PurchasedCount int            `json:"purchased_count" minimum:"0" doc:"Completed purchases of it by the caller in the current period."`
}

type ShopPurchase struct {
	Object      string    `json:"object" enum:"shop_purchase" maxLength:"13" doc:"Type discriminant. Always shop_purchase."`
	Order       ShopOrder `json:"order" doc:"The order. For a redeem code its redeem_codes are here; show them at once."`
	Moemoepoint int       `json:"moemoepoint" minimum:"-2147483648" doc:"The caller's live moemoepoint balance after the purchase."`
}

type listShopOffersOutput struct {
	Body repr.List[ShopOffer]
}

type getMyShopInventoryOutput struct {
	Body ShopInventory
}

type ShopOrderCreate struct {
	OfferID repr.DecimalID `json:"offer_id" doc:"The offer to buy, from listShopOffers. Show the caller the offer, its price and the balance after it, and send this only from their confirmation: nothing asks them again."`
}

type createShopOrderInput struct {
	Body ShopOrderCreate
}

type createShopOrderOutput struct {
	Body ShopPurchase
}

type ShopLoadoutSet struct {
	ItemID repr.DecimalID `json:"item_id" doc:"An item the caller holds and is active, of the slot's type."`
}

type setShopLoadoutSlotInput struct {
	Scope string `path:"scope" enum:"everywhere,this_site" maxLength:"10" doc:"everywhere for every NextMoe site, this_site for this forum only."`
	Slot  string `path:"slot" enum:"avatar_frame,profile_background" maxLength:"18" doc:"The slot."`
	Body  ShopLoadoutSet
}

type clearShopLoadoutSlotInput struct {
	Scope string `path:"scope" enum:"everywhere,this_site" maxLength:"10" doc:"everywhere for every NextMoe site, this_site for this forum only."`
	Slot  string `path:"slot" enum:"avatar_frame,profile_background" maxLength:"18" doc:"The slot."`
}

type shopLoadoutSlotOutput struct {
	Body ShopLoadoutSlot
}
