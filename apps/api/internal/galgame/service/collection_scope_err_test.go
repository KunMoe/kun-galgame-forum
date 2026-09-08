package service

import (
	"fmt"
	"testing"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
)

// The frontend keys its "sign in again" prompt on code 235 and nothing else;
// 205 sends it down a path that re-checks this site's own cookie, finds it
// healthy, and shows the reader nothing. Getting this pair wrong is invisible
// in production, so it is asserted here rather than noticed later.
func TestCollectionErrSeparatesANarrowGrantFromADeadSession(t *testing.T) {
	scope := collectionErr(fmt.Errorf("reading folders: %w", catalogclient.ErrInsufficientScope), "读取收藏夹失败")
	if scope == nil || scope.Code != errors.CodeReauthRequired {
		t.Fatalf("a narrow grant must map to code %d, got %+v", errors.CodeReauthRequired, scope)
	}

	dead := collectionErr(fmt.Errorf("reading folders: %w", catalogclient.ErrUnauthorized), "读取收藏夹失败")
	if dead == nil || dead.Code != errors.CodeAuth {
		t.Fatalf("an expired session must map to code %d, got %+v", errors.CodeAuth, dead)
	}

	if scope.Code == dead.Code {
		t.Fatal("the two faults must stay distinguishable to the client")
	}
}
