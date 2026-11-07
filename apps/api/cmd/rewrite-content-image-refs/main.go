package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type target struct {
	table string
	cols  []string
}

var targets = []target{
	{"topic", []string{"content"}},
	{"topic_reply", []string{"content"}},
	{"topic_comment", []string{"content"}},
	{"topic_draft", []string{"content"}},
	{"chat_message", []string{"content"}},
	{"galgame_resource", []string{"note"}},
	{"galgame_toolset_resource", []string{"content", "note"}},
	{"galgame_toolset", []string{"description"}},
	{"doc_article", []string{"content_markdown"}},
	{"galgame_quiz", []string{"description"}},
	{"todo", []string{"content"}},
	{"update_log", []string{"content"}},
	// feed_activity last: source-table rewrites refresh it via trg_feed_*; only orphan types remain.
	{"feed_activity", []string{"content"}},
}

const batchSize = 500

type targetStats struct {
	table           string
	scanned         int
	changed         int
	legacyRewritten int
	absoluteFolded  int
	unmappedLegacy  int
	skippedTooLong  int
}

func main() {
	_ = godotenv.Load()

	dryRun := flag.Bool("dry-run", true, "TRUE (default): scan + report only. Pass -dry-run=false to apply.")
	baseFlag := flag.String("base", "", "Absolute CDN base to rewrite (default = cfg.NextMoeAPI.ImageCDNBase)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	base := strings.TrimRight(orDefault(*baseFlag, cfg.NextMoeAPI.ImageCDNBase), "/")
	if base == "" {
		slog.Error("CDN base 为空 (设 KUN_IMAGE_PUBLIC_BASE_URL 或 -base)")
		os.Exit(1)
	}
	markdown.SetContentImageCDNBase(base)

	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)

	likes := []string{
		"%" + base + "%",
		"%sticker.kungal.com/stickers/KUNgal%",
	}
	for _, h := range imageclient.KnownHosts {
		likes = append(likes, "%"+h+"/%")
	}

	limits := map[string]map[string]int{}
	for _, t := range targets {
		limits[t.table] = map[string]int{}
		for _, col := range t.cols {
			limits[t.table][col] = columnMaxLen(db, t.table, col)
		}
	}

	slog.Info("开始把绝对 image_service URL 与遗留 sticker URL 改写为 /image/<hash>",
		"dry_run", *dryRun, "base", base)

	var all []targetStats
	for _, t := range targets {
		st := rewriteTarget(db, t, likes, limits[t.table], *dryRun)
		all = append(all, st)
		slog.Info("目标完成",
			"table", st.table,
			"scanned", st.scanned,
			"changed", st.changed,
			"legacy_rewritten", st.legacyRewritten,
			"absolute_folded", st.absoluteFolded,
			"unmapped_legacy", st.unmappedLegacy,
			"skipped_too_long", st.skippedTooLong,
		)
	}

	fmt.Println("out of scope: galgame_website.icon, friend_link.banner")
	var scanned, changed, legacy, folded, unmapped, skipped int
	for _, st := range all {
		scanned += st.scanned
		changed += st.changed
		legacy += st.legacyRewritten
		folded += st.absoluteFolded
		unmapped += st.unmappedLegacy
		skipped += st.skippedTooLong
		fmt.Printf("%s scanned=%d changed=%d legacy_rewritten=%d absolute_folded=%d unmapped_legacy=%d skipped_too_long=%d\n",
			st.table, st.scanned, st.changed, st.legacyRewritten, st.absoluteFolded, st.unmappedLegacy, st.skippedTooLong)
	}

	if *dryRun {
		fmt.Printf("dry-run 完成: 将改写 %d 行(scanned=%d, legacy_rewritten=%d, absolute_folded=%d, unmapped_legacy=%d, skipped_too_long=%d)。加 -dry-run=false 执行。\n",
			changed, scanned, legacy, folded, unmapped, skipped)
		return
	}
	slog.Info("改写完成",
		"改写行数", changed,
		"scanned", scanned,
		"legacy_rewritten", legacy,
		"absolute_folded", folded,
		"unmapped_legacy", unmapped,
		"skipped_too_long", skipped,
	)
	fmt.Printf("完成: 改写 %d 行(scanned=%d, legacy_rewritten=%d, absolute_folded=%d, unmapped_legacy=%d, skipped_too_long=%d)。\n",
		changed, scanned, legacy, folded, unmapped, skipped)
}

func rewriteTarget(db *gorm.DB, t target, likes []string, colLimits map[string]int, dryRun bool) targetStats {
	st := targetStats{table: t.table}
	lastID := int64(0)
	for {
		rows, err := fetchBatch(db, t.table, t.cols, lastID, likes, batchSize)
		if err != nil {
			slog.Error("扫描失败", "table", t.table, "error", err)
			os.Exit(1)
		}
		if len(rows) == 0 {
			return st
		}

		type pending struct {
			id   int64
			vals []string
		}
		var updates []pending

		for _, row := range rows {
			st.scanned++
			lastID = row.id

			newVals := make([]string, len(t.cols))
			tooLong := false
			changed := false
			var rowLegacy, rowAbs, rowUnmapped int
			for i, col := range t.cols {
				src := row.vals[i]
				afterLegacy := markdown.ResolveLegacyStickerRefs(src)
				dst := markdown.NormalizeImageRefs(afterLegacy)
				if lim := colLimits[col]; lim > 0 && utf8.RuneCountInString(dst) > lim {
					tooLong = true
					break
				}
				rowLegacy += countLegacy(src) - countLegacy(afterLegacy)
				rowAbs += strings.Count(dst, "/image/") - strings.Count(afterLegacy, "/image/")
				rowUnmapped += countLegacy(dst)
				newVals[i] = dst
				if dst != src {
					changed = true
				}
			}
			if tooLong {
				st.skippedTooLong++
				continue
			}
			st.legacyRewritten += rowLegacy
			st.absoluteFolded += rowAbs
			st.unmappedLegacy += rowUnmapped
			if !changed {
				continue
			}
			st.changed++
			if dryRun {
				continue
			}
			updates = append(updates, pending{id: row.id, vals: newVals})
		}

		if dryRun || len(updates) == 0 {
			if len(rows) < batchSize {
				return st
			}
			continue
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			for _, u := range updates {
				sets := make([]string, len(t.cols))
				args := make([]any, 0, len(t.cols)+1)
				for i, col := range t.cols {
					sets[i] = col + " = ?"
					args = append(args, u.vals[i])
				}
				args = append(args, u.id)
				q := "UPDATE " + t.table + " SET " + strings.Join(sets, ", ") + " WHERE id = ?"
				if err := tx.Exec(q, args...).Error; err != nil {
					slog.Error("更新行失败", "table", t.table, "id", u.id, "error", err)
					continue
				}
				slog.Info("已改写", "table", t.table, "id", u.id)
			}
			return nil
		})
		if err != nil {
			slog.Error("提交批次失败", "table", t.table, "error", err)
			os.Exit(1)
		}
		if len(rows) < batchSize {
			return st
		}
	}
}

type contentRow struct {
	id   int64
	vals []string
}

func fetchBatch(db *gorm.DB, table string, cols []string, lastID int64, likes []string, limit int) ([]contentRow, error) {
	var clauses []string
	var args []any
	for _, col := range cols {
		for _, like := range likes {
			clauses = append(clauses, col+" LIKE ?")
			args = append(args, like)
		}
	}
	q := "SELECT id"
	for _, col := range cols {
		q += ", " + col
	}
	q += " FROM " + table + " WHERE (" + strings.Join(clauses, " OR ") + ") AND id > ? ORDER BY id ASC LIMIT ?"
	args = append(args, lastID, limit)

	rows, err := db.Raw(q, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []contentRow
	for rows.Next() {
		var id int64
		nulls := make([]sql.NullString, len(cols))
		dest := make([]any, 1+len(cols))
		dest[0] = &id
		for i := range cols {
			dest[i+1] = &nulls[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		vals := make([]string, len(cols))
		for i, ns := range nulls {
			vals[i] = ns.String
		}
		out = append(out, contentRow{id: id, vals: vals})
	}
	return out, rows.Err()
}

func columnMaxLen(db *gorm.DB, table, col string) int {
	var n sql.NullInt64
	if err := db.Raw(
		`SELECT character_maximum_length FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?`,
		table, col,
	).Scan(&n).Error; err != nil {
		slog.Error("读取列长度失败", "table", table, "column", col, "error", err)
		os.Exit(1)
	}
	if !n.Valid {
		return 0
	}
	return int(n.Int64)
}

// Counts COMPLETE legacy URLs, not the bare prefix. feed_activity excerpts are
// SUBSTRING(content, 1, 100) (migration 034), so four of them end mid-URL with no
// .webp; counting the prefix reported those as unmapped_legacy=4 and made a clean
// run look broken. A fragment names no sticker and is unrepairable by construction.
func countLegacy(s string) int {
	return len(legacyURLRe.FindAllString(s, -1))
}

var legacyURLRe = regexp.MustCompile(`https?://sticker\.kungal\.com/stickers/KUNgal\d{1,2}/\d{1,3}\.webp`)

func orDefault(v, def string) string {
	if v != "" {
		return v
	}
	return def
}
