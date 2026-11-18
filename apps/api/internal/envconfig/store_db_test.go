package envconfig

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/pkg/config"
)

func TestStoreSealsAuditsAndDeletes(t *testing.T) {
	db := testdb.Open(t)
	clean := func() {
		db.Exec(`DELETE FROM config_secret WHERE key LIKE 'KUN_ENVTEST_%'`)
		db.Exec(`DELETE FROM config_secret_audit WHERE key LIKE 'KUN_ENVTEST_%'`)
	}
	clean()
	t.Cleanup(clean)
	ctx := context.Background()
	key := bytes.Repeat([]byte{7}, 32)
	s := New(db, key)
	uid := 42

	if changed, err := s.Set(ctx, "KUN_ENVTEST_API_KEY", "nmk_live_first", "", ActionSet, &uid); err != nil || !changed {
		t.Fatalf("first set = %v, %v", changed, err)
	}
	if changed, err := s.Set(ctx, "KUN_ENVTEST_API_KEY", "nmk_live_first", "", ActionSet, &uid); err != nil || changed {
		t.Fatalf("setting the held value = %v, %v; want unchanged", changed, err)
	}
	if _, err := s.Set(ctx, "KUN_ENVTEST_API_KEY", "nmk_live_second", "rotated", ActionSet, &uid); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Set(ctx, "KUN_ENVTEST_BASE_URL", "https://checker", "", ActionImport, nil); err != nil {
		t.Fatal(err)
	}

	var cipher []byte
	db.Raw(`SELECT cipher FROM config_secret WHERE key = 'KUN_ENVTEST_API_KEY'`).Scan(&cipher)
	if bytes.Contains(cipher, []byte("nmk_live_second")) {
		t.Fatal("the stored value is in the clear")
	}
	all, err := s.All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, st := range all {
		got[st.Key] = st.Value
	}
	if got["KUN_ENVTEST_API_KEY"] != "nmk_live_second" || got["KUN_ENVTEST_BASE_URL"] != "https://checker" {
		t.Fatalf("All = %v", got)
	}

	if deleted, err := s.Delete(ctx, "KUN_ENVTEST_BASE_URL", "gone", &uid); err != nil || !deleted {
		t.Fatalf("delete = %v, %v", deleted, err)
	}
	if deleted, err := s.Delete(ctx, "KUN_ENVTEST_BASE_URL", "", &uid); err != nil || deleted {
		t.Fatalf("deleting nothing = %v, %v", deleted, err)
	}

	var rows []AuditRow
	db.Where("key LIKE 'KUN_ENVTEST_%'").Order("id").Find(&rows)
	if len(rows) != 4 {
		t.Fatalf("audit rows = %d, want 4 (set, set, import, delete): %+v", len(rows), rows)
	}
	for _, r := range rows {
		if strings.Contains(r.OldShown+r.NewShown, "nmk_live") {
			t.Errorf("audit shows a secret in the clear: %+v", r)
		}
	}
	rotate := rows[1]
	if rotate.Note != "rotated" || rotate.OldShown != s.Shown("KUN_ENVTEST_API_KEY", "nmk_live_first") ||
		rotate.NewShown != config.FingerprintWith(key, "nmk_live_second")+" …cond" {
		t.Errorf("rotation audit %+v", rotate)
	}
	if imp := rows[2]; imp.Action != ActionImport || imp.UserID != nil || imp.NewShown != "https://checker" {
		t.Errorf("import audit %+v", imp)
	}
	if del := rows[3]; del.Action != ActionDelete || del.OldShown != "https://checker" || del.NewShown != "" {
		t.Errorf("delete audit %+v", del)
	}

	if _, err := New(db, nil).Set(ctx, "KUN_ENVTEST_API_KEY", "x", "", ActionSet, &uid); err != ErrStoreOff {
		t.Errorf("a store without a key wrote: %v", err)
	}
}
