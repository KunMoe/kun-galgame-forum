package apiv1

import (
	"context"
	"log/slog"
	"math"
	"time"
)

const accountPricesTTL = 5 * time.Minute

type getAccountPricesOutput struct {
	Body AccountPrices
}

func (s *Users) getAccountPrices(ctx context.Context, _ *struct{}) (*getAccountPricesOutput, error) {
	if prob := s.readyAccounts(); prob != nil {
		return nil, prob
	}
	now := time.Now()
	s.pricesMu.Lock()
	if !s.pricesUntil.IsZero() && now.Before(s.pricesUntil) {
		cost := s.pricesCost
		s.pricesMu.Unlock()
		return accountPricesOut(cost), nil
	}
	s.pricesMu.Unlock()

	settings, err := s.accounts.PublicSettings(ctx)
	if err != nil {
		slog.Warn("account prices: public settings unavailable", "err", err)
		return accountPricesOut(nil), nil
	}
	cost := renameCostFromSettings(settings)
	s.pricesMu.Lock()
	s.pricesCost = cost
	s.pricesUntil = time.Now().Add(accountPricesTTL)
	s.pricesMu.Unlock()
	return accountPricesOut(cost), nil
}

func accountPricesOut(cost *int) *getAccountPricesOutput {
	return &getAccountPricesOutput{Body: AccountPrices{
		Object:     "account_prices",
		RenameCost: cost,
	}}
}

func renameCostFromSettings(settings map[string]any) *int {
	f, ok := settings["auth.name_change_cost"].(float64)
	if !ok || f < 0 || f != math.Trunc(f) {
		return nil
	}
	n := int(f)
	return &n
}
