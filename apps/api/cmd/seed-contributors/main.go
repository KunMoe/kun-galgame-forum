package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

const chunkSize = 500

func main() {
	_ = godotenv.Load()

	file := flag.String("file", "", "Path to the frozen wiki contributor TSV (work_id, catalog_work_id, user_id, created)")
	apply := flag.Bool("apply", false, "Write to the database (default: report only)")
	flag.Parse()

	if *file == "" {
		slog.Error("必须指定 --file")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	f, err := os.Open(*file)
	if err != nil {
		slog.Error("打开 TSV 失败", "file", *file, "error", err)
		os.Exit(1)
	}
	defer f.Close()

	rows, stats, err := parseContributorTSV(f)
	if err != nil {
		slog.Error("读取 TSV 失败", "file", *file, "error", err)
		os.Exit(1)
	}
	slog.Info("TSV 解析完成", "lines", stats.Lines, "pairs", len(rows),
		"skipped", stats.Skipped, "folded", stats.Folded)

	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)

	if !*apply {
		present, err := countPresent(db, rows)
		if err != nil {
			slog.Error("统计已存在的贡献者行失败", "error", err)
			os.Exit(1)
		}
		fmt.Printf("dry-run: 待写入 %d 对, 其中已存在 %d, 将新增 %d (加 --apply 执行)\n",
			len(rows), present, len(rows)-present)
		return
	}

	inserted, updated, err := seed(db, rows)
	if err != nil {
		slog.Error("写入贡献者失败", "inserted", inserted, "error", err)
		os.Exit(1)
	}
	workIDs, err := refreshCounts(db, rows)
	if err != nil {
		slog.Error("刷新 contributor_count 失败", "error", err)
		os.Exit(1)
	}
	fmt.Printf("seed 完成: 新增 %d, 已存在 %d, 刷新 contributor_count 的 galgame %d 个\n",
		inserted, updated, workIDs)
}

func countPresent(db *gorm.DB, rows []seedRow) (int, error) {
	present := 0
	for _, chunk := range chunks(rows) {
		args := make([]any, 0, len(chunk)*2)
		values := make([]string, 0, len(chunk))
		for _, r := range chunk {
			values = append(values, "(?::bigint, ?::bigint)")
			args = append(args, r.WorkID, r.UserID)
		}
		var n int64
		if err := db.Raw(`
			SELECT COUNT(*) FROM galgame_contributor c
			JOIN (VALUES `+strings.Join(values, ", ")+`) AS v(work_id, user_id)
			  ON c.work_id = v.work_id AND c.user_id = v.user_id`,
			args...).Scan(&n).Error; err != nil {
			return present, err
		}
		present += int(n)
	}
	return present, nil
}

func seed(db *gorm.DB, rows []seedRow) (inserted, updated int, err error) {
	for _, chunk := range chunks(rows) {
		args := make([]any, 0, len(chunk)*3)
		values := make([]string, 0, len(chunk))
		for _, r := range chunk {
			values = append(values, "(?::bigint, ?::bigint, ?::timestamptz, ?::timestamptz, 0, 0)")
			args = append(args, r.WorkID, r.UserID, r.Created, r.Created)
		}
		var flags []bool
		if err := db.Raw(`
			INSERT INTO galgame_contributor
				(work_id, user_id, first_at, last_at, revision_count, source)
			VALUES `+strings.Join(values, ", ")+`
			ON CONFLICT (work_id, user_id) DO UPDATE SET
				first_at = LEAST(galgame_contributor.first_at, excluded.first_at)
			RETURNING (xmax = 0) AS inserted`,
			args...).Scan(&flags).Error; err != nil {
			return inserted, updated, err
		}
		for _, isInsert := range flags {
			if isInsert {
				inserted++
				continue
			}
			updated++
		}
	}
	return inserted, updated, nil
}

func refreshCounts(db *gorm.DB, rows []seedRow) (int, error) {
	seen := map[int64]bool{}
	workIDs := make([]int64, 0, len(rows))
	for _, r := range rows {
		if !seen[r.WorkID] {
			seen[r.WorkID] = true
			workIDs = append(workIDs, r.WorkID)
		}
	}
	for i := 0; i < len(workIDs); i += chunkSize {
		end := min(i+chunkSize, len(workIDs))
		if err := db.Exec(`
			UPDATE galgame SET contributor_count = (
				SELECT COUNT(*) FROM galgame_contributor c WHERE c.work_id = galgame.id
			) WHERE id IN ?`, workIDs[i:end]).Error; err != nil {
			return 0, err
		}
	}
	return len(workIDs), nil
}

func chunks(rows []seedRow) [][]seedRow {
	out := make([][]seedRow, 0, len(rows)/chunkSize+1)
	for i := 0; i < len(rows); i += chunkSize {
		out = append(out, rows[i:min(i+chunkSize, len(rows))])
	}
	return out
}
