package notifytype_test

import (
	"testing"

	"kun-galgame-api/internal/message/notifytype"
	"kun-galgame-api/internal/message/service"
)

func TestV1TokensRoundTripAndMatchLocalNotificationTypes(t *testing.T) {
	all := notifytype.All()
	if len(all) != 18 {
		t.Fatalf("All() length %d, want 18", len(all))
	}
	dbSet := map[string]struct{}{}
	seen := map[notifytype.Type]struct{}{}
	for i, tok := range all {
		if _, dup := seen[tok]; dup {
			t.Fatalf("All() duplicated %q at %d", tok, i)
		}
		seen[tok] = struct{}{}
		db := notifytype.ToDB(tok)
		if db == "" {
			t.Fatalf("ToDB(%q) is empty", tok)
		}
		back, ok := notifytype.FromDB(db)
		if !ok || back != tok {
			t.Fatalf("FromDB(%q) = (%q, %v), want (%q, true)", db, back, ok, tok)
		}
		if _, dup := dbSet[db]; dup {
			t.Fatalf("two v1 tokens map to %q", db)
		}
		dbSet[db] = struct{}{}
	}
	local := map[string]struct{}{}
	for _, s := range service.LocalNotificationTypes {
		local[s] = struct{}{}
	}
	if len(local) != len(dbSet) {
		t.Fatalf("LocalNotificationTypes has %d values, mapped DB set has %d", len(local), len(dbSet))
	}
	for s := range local {
		if _, ok := dbSet[s]; !ok {
			t.Errorf("LocalNotificationTypes member %q is missing from ToDB", s)
		}
	}
	for s := range dbSet {
		if _, ok := local[s]; !ok {
			t.Errorf("ToDB value %q is not in LocalNotificationTypes", s)
		}
	}
	if _, ok := notifytype.FromDB("admin"); ok {
		t.Fatal("admin must not be a v1 type")
	}
}
