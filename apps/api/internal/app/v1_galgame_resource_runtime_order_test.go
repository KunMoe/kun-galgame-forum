package app

import (
	"os"
	"testing"
	"time"
)

func TestMigration203MovesTyranorNextUpInStoredRuntimes(t *testing.T) {
	f := newResourceFix(t, nil)
	rows := []struct {
		id           int
		before, want string
	}{
		{g3ResMain,
			`["native-win", "native-and", "native-ios", "gamehub", "kirikiroid2", "tyranor", "tyranor-next", "other"]`,
			`["native-win", "native-and", "tyranor-next", "native-ios", "gamehub", "kirikiroid2", "tyranor", "other"]`},
		{g3ResExpired,
			`["native-win", "native-and", "kirikiroid2", "gamehub", "tyranor", "tyranor-next", "emulator"]`,
			`["native-win", "native-and", "tyranor-next", "gamehub", "kirikiroid2", "tyranor", "emulator"]`},
		{g3ResCount0,
			`["kirikiroid2", "tyranor", "tyranor-next", "emulator"]`,
			`["tyranor-next", "kirikiroid2", "tyranor", "emulator"]`},
		{g3ResNSFW, `["kirikiroid2", "gamehub"]`, `["kirikiroid2", "gamehub"]`},
	}
	stamps := map[int]time.Time{}
	for _, r := range rows {
		if err := f.db.Exec(`UPDATE galgame_resource SET runtimes = ?::jsonb WHERE id = ?`, r.before, r.id).Error; err != nil {
			t.Fatal(err)
		}
		var updated time.Time
		if err := f.db.Raw(`SELECT updated FROM galgame_resource WHERE id = ?`, r.id).Scan(&updated).Error; err != nil {
			t.Fatal(err)
		}
		stamps[r.id] = updated
	}
	up, err := os.ReadFile("../../migrations/203_resource_runtime_order.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for pass := 1; pass <= 2; pass++ {
		if err := f.db.Exec(string(up)).Error; err != nil {
			t.Fatal(err)
		}
		for _, r := range rows {
			var got struct {
				Runtimes string
				Updated  time.Time
			}
			if err := f.db.Raw(`SELECT runtimes::text AS runtimes, updated FROM galgame_resource WHERE id = ?`, r.id).Scan(&got).Error; err != nil {
				t.Fatal(err)
			}
			if got.Runtimes != r.want {
				t.Errorf("pass %d, resource %d\n got %s\nwant %s", pass, r.id, got.Runtimes, r.want)
			}
			if !got.Updated.Equal(stamps[r.id]) {
				t.Errorf("pass %d, resource %d: updated moved from %v to %v", pass, r.id, stamps[r.id], got.Updated)
			}
		}
	}
}
