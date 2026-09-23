package cron

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"kun-galgame-api/pkg/imageclient"

	"gorm.io/gorm"
)

var contentImageHashRe = regexp.MustCompile(`/image/([0-9a-f]{64})`)

const refPingBatchSize = 1000

var ErrRefPingNoEffect = errors.New(
	"内容图 reference-ping 命中 0 个 hash (内容里有 token 但全部 not_found) — 疑似 image client/site 配错或服务异常",
)

func RunReferencePing(
	ctx context.Context, db *gorm.DB, imgCli *imageclient.Client,
) (distinct int, updated int64, err error) {
	if imgCli == nil {
		return 0, 0, nil
	}

	hashes, err := collectContentImageHashes(ctx, db)
	if err != nil {
		return 0, 0, err
	}

	for start := 0; start < len(hashes); start += refPingBatchSize {
		end := min(start+refPingBatchSize, len(hashes))
		res, e := imgCli.ReferencePing(ctx, hashes[start:end])
		if e != nil {
			slog.Error("内容图 reference-ping 批次失败", "from", start, "to", end, "error", e)
			if err == nil {
				err = e
			}
			continue
		}
		updated += res.Updated
	}

	if err == nil && len(hashes) > 0 && updated == 0 {
		err = ErrRefPingNoEffect
	}
	return len(hashes), updated, err
}

func collectContentImageHashes(ctx context.Context, db *gorm.DB) ([]string, error) {
	type column struct {
		TableName  string
		ColumnName string
	}
	var cols []column
	if err := db.WithContext(ctx).Raw(`
		SELECT table_name, column_name FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND data_type IN ('text', 'character varying', 'character')
	`).Scan(&cols).Error; err != nil {
		return nil, err
	}

	type contentRow struct{ Content string }
	var contents []string
	for _, c := range cols {
		col := quoteIdent(c.ColumnName)
		q := fmt.Sprintf(
			"SELECT %s AS content FROM %s WHERE %s LIKE '%%/image/%%'",
			col, quoteIdent(c.TableName), col,
		)
		var rows []contentRow
		if err := db.WithContext(ctx).Raw(q).Scan(&rows).Error; err != nil {
			slog.Warn("reference-ping 扫描列失败, 跳过",
				"table", c.TableName, "column", c.ColumnName, "error", err)
			continue
		}
		for _, r := range rows {
			contents = append(contents, r.Content)
		}
	}
	bare, err := collectBareImageHashes(ctx, db)
	if err != nil {
		return nil, err
	}
	for _, h := range bare {
		contents = append(contents, "/image/"+h)
	}
	return extractContentImageHashes(contents), nil
}

type hashColumn struct {
	Table    string `gorm:"column:table_name"`
	Column   string `gorm:"column:column_name"`
	DataType string `gorm:"column:data_type"`
}

// The text-column scan above matches /image/<hash> tokens only, so a column
// holding a bare hash was never pinged: lottery prize images, doc banners,
// friend-link banners and website icons all aged towards the image service's
// cold tier and soft delete before the 2026-09-23 audit. Bare-hash columns are
// found by name so a new one is covered the day it is added: a single hash in
// a text column named *image_hash, or a jsonb array named *image_hashes.
func bareImageHashColumns(ctx context.Context, db *gorm.DB) ([]hashColumn, error) {
	var cols []hashColumn
	err := db.WithContext(ctx).Raw(`
		SELECT table_name, column_name, data_type FROM information_schema.columns
		WHERE table_schema = 'public' AND (
			(column_name LIKE '%image\_hash' AND data_type IN ('text', 'character varying')) OR
			(column_name LIKE '%image\_hashes' AND data_type = 'jsonb'))
		ORDER BY table_name, column_name
	`).Scan(&cols).Error
	return cols, err
}

func collectBareImageHashes(ctx context.Context, db *gorm.DB) ([]string, error) {
	cols, err := bareImageHashColumns(ctx, db)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, c := range cols {
		col, table := quoteIdent(c.Column), quoteIdent(c.Table)
		q := fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE %s <> ''", col, table, col)
		if c.DataType == "jsonb" {
			q = fmt.Sprintf("SELECT DISTINCT jsonb_array_elements_text(%s) FROM %s WHERE jsonb_typeof(%s) = 'array'", col, table, col)
		}
		var rows []string
		if err := db.WithContext(ctx).Raw(q).Scan(&rows).Error; err != nil {
			return nil, err
		}
		out = append(out, rows...)
	}
	return out, nil
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func extractContentImageHashes(contents []string) []string {
	seen := make(map[string]struct{})
	for _, c := range contents {
		for _, m := range contentImageHashRe.FindAllStringSubmatch(c, -1) {
			seen[m[1]] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for h := range seen {
		out = append(out, h)
	}
	return out
}
