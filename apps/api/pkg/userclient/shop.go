package userclient

import (
	"context"
	"fmt"
	"time"
)

type ShopDecoration struct {
	StaticURL   string `json:"static_url"`
	AnimatedURL string `json:"animated_url"`
}

type ShopItem struct {
	ID          int64           `json:"id"`
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Preview     *ShopDecoration `json:"preview"`
}

type ShopReward struct {
	Item         ShopItem `json:"item"`
	DurationDays int      `json:"duration_days"`
}

type ShopSite struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type ShopOffer struct {
	ID           int64        `json:"id"`
	SiteID       *int64       `json:"site_id"`
	Price        int64        `json:"price"`
	Rewards      []ShopReward `json:"rewards"`
	EndsAt       *time.Time   `json:"ends_at"`
	PerUserLimit int          `json:"per_user_limit"`
	LimitPeriod  string       `json:"limit_period"`
	Stock        *int         `json:"stock"`
	Remaining    *int         `json:"remaining"`
}

type ShopStorefront struct {
	Site   ShopSite    `json:"site"`
	Offers []ShopOffer `json:"offers"`
}

type ShopOwnedItem struct {
	Item       ShopItem   `json:"item"`
	Source     string     `json:"source"`
	AcquiredAt time.Time  `json:"acquired_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	Active     bool       `json:"active"`
}

type ShopLoadout struct {
	Slot   string `json:"slot"`
	SiteID int64  `json:"site_id"`
	ItemID int64  `json:"item_id"`
}

type ShopCode struct {
	Code      string  `json:"code"`
	ExpiresOn *string `json:"expires_on"`
}

type ShopOrder struct {
	ID         int64        `json:"id"`
	OfferID    int64        `json:"offer_id"`
	Price      int64        `json:"price"`
	Status     string       `json:"status"`
	Rewards    []ShopReward `json:"rewards"`
	Codes      []ShopCode   `json:"codes"`
	CreatedAt  time.Time    `json:"created_at"`
	RefundedAt *time.Time   `json:"refunded_at"`
}

type ShopInventory struct {
	Balance   int64           `json:"balance"`
	Items     []ShopOwnedItem `json:"items"`
	Loadout   []ShopLoadout   `json:"loadout"`
	Orders    []ShopOrder     `json:"orders"`
	LimitUsed map[int64]int   `json:"limit_used"`
}

type ShopPurchase struct {
	Order   ShopOrder `json:"order"`
	Balance int64     `json:"balance"`
	Replay  bool      `json:"replay"`
}

func (c *Client) ShopStorefront(ctx context.Context) (ShopStorefront, error) {
	var out ShopStorefront
	err := c.do(ctx, "GET", c.cfg.BaseURL+"/shop/storefront", &out)
	return out, err
}

func (c *Client) ShopInventory(ctx context.Context, userID int) (ShopInventory, error) {
	var out ShopInventory
	err := c.do(ctx, "GET", fmt.Sprintf("%s/users/%d/shop", c.cfg.BaseURL, userID), &out)
	return out, err
}

func (c *Client) ShopPurchase(ctx context.Context, userID int, offerID int64, idempotencyKey string) (ShopPurchase, error) {
	var out ShopPurchase
	err := c.doJSON(ctx, "POST", fmt.Sprintf("%s/users/%d/shop/orders", c.cfg.BaseURL, userID), map[string]any{
		"offer_id":        offerID,
		"idempotency_key": idempotencyKey,
	}, &out)
	return out, err
}

func (c *Client) ShopEquip(ctx context.Context, userID int, slot string, siteID int64, itemID *int64) error {
	return c.doJSON(ctx, "PUT", fmt.Sprintf("%s/users/%d/shop/loadout", c.cfg.BaseURL, userID), map[string]any{
		"slot":    slot,
		"site_id": siteID,
		"item_id": itemID,
	}, nil)
}
