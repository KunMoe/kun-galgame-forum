package push

import (
	"context"
	"testing"

	activityapiv1 "kun-galgame-api/internal/activity/apiv1"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/communityclient"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type stubPresentationUsers struct{}

func (stubPresentationUsers) Users(context.Context, []int) (map[int]userclient.User, error) {
	return map[int]userclient.User{}, nil
}

type stubPresentationCatalog struct{}

func (stubPresentationCatalog) CatalogRowsByWorkIDs(context.Context, []int, string, string) (map[int]client.CatalogWorkListItem, *legacyErrors.AppError) {
	return map[int]client.CatalogWorkListItem{}, nil
}

func TestPresenterStartNoOpWithoutHTTPSOrigin(t *testing.T) {
	cm := communityclient.New(communityclient.Config{BaseURL: "https://community.example", ClientID: "c", ClientSecret: "s"})
	p := NewPresenter(&gorm.DB{}, cm, stubPresentationUsers{}, stubPresentationCatalog{}, "", Origin("http://127.0.0.1:2334/api/auth/callback"))
	if p.Ready() {
		t.Fatal("http origin must not start the presenter")
	}
	p.Start()()
}

func TestPresenterStartNoOpUnconfiguredCommunity(t *testing.T) {
	cm := communityclient.New(communityclient.Config{BaseURL: "https://community.example"})
	p := NewPresenter(&gorm.DB{}, cm, stubPresentationUsers{}, stubPresentationCatalog{}, "", "https://www.kungal.com")
	if p.Ready() {
		t.Fatal("unconfigured community must not start the presenter")
	}
	p.Start()()
}

var (
	_ activityapiv1.Users   = stubPresentationUsers{}
	_ activityapiv1.Catalog = stubPresentationCatalog{}
)
