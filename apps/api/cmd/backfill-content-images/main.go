package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
)

var legacyImageRe = regexp.MustCompile(`https://image\.kungal\.com/[^\s)"'>\]\\]+`)

const trailingPunct = `.,;!?，。、）】>`

type target struct {
	table string
	col   string
}

var targets = []target{
	{"topic", "content"},
	{"topic_reply", "content"},
	{"chat_message", "content"},
	{"doc_article", "content_markdown"},
	{"galgame_toolset", "description"},
	{"message", "content"},
}

func main() {
	_ = godotenv.Load()

	dryRun := flag.Bool("dry-run", true, "TRUE (default): scan + report workload only, no network, no DB writes. Pass -dry-run=false to apply.")
	base := flag.String("base", "https://image.kungal.com", "Base to HTTP-fetch legacy originals from (override if the old host isn't reachable from here)")
	limit := flag.Int("limit", 0, "Max rows per table (0 = all); for smoke-testing -dry-run=false on a small batch")
	preset := flag.String("preset", "topic", "image_service preset to rehost under")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	imgCli := imageclient.New(imageclient.Config{
		BaseURL:      cfg.ImageClient.BaseURL,
		CDNBase:      cfg.NextMoeAPI.ImageCDNBase,
		ClientID:     cfg.ImageClient.ClientID,
		ClientSecret: cfg.ImageClient.ClientSecret,
	})
	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)

	httpClient := &http.Client{Timeout: 60 * time.Second}
	ctx := context.Background()

	migrated := map[string]string{}
	dead := map[string]bool{}
	seen := map[string]bool{}
	deadList := []string{}
	perTableRows := map[string]int{}

	var rowsUpdated, rowsSkipped, replacements, uploads int

	slog.Info("开始迁移 content 老图", "dry_run", *dryRun, "base", *base, "preset", *preset, "limit", *limit)

	for _, t := range targets {
		type row struct {
			ID      int64
			Content string
		}
		var rows []row
		q := db.Table(t.table).
			Select("id, "+t.col+" AS content").
			Where(t.col+" LIKE ?", "%image.kungal.com%").
			Order("id ASC")
		if *limit > 0 {
			q = q.Limit(*limit)
		}
		if err := q.Scan(&rows).Error; err != nil {
			slog.Error("扫描失败", "table", t.table, "error", err)
			os.Exit(1)
		}
		perTableRows[t.table] = len(rows)
		slog.Info("扫描完成", "table", t.table, "含老图行数", len(rows))

		for _, r := range rows {
			urls := extractLegacyURLs(r.Content)
			newContent := r.Content
			rowReplaced := 0

			for _, old := range urls {
				seen[old] = true
				if dead[old] {
					continue
				}
				if *dryRun {
					continue
				}
				newURL, done := migrated[old]
				if !done {
					body, ferr := fetch(ctx, httpClient, rewriteHost(old, *base))
					if ferr != nil {
						slog.Warn("抓取原图失败, 跳过(保留旧 URL)", "url", old, "error", ferr)
						dead[old] = true
						deadList = append(deadList, old)
						continue
					}
					res, uerr := imgCli.Upload(ctx, bytes.NewReader(body), path.Base(old), *preset)
					if uerr != nil {
						slog.Error("上传 image_service 失败, 跳过", "url", old, "error", uerr)
						dead[old] = true
						deadList = append(deadList, old)
						continue
					}
					newURL = "/image/" + res.Hash
					migrated[old] = newURL
					uploads++
					slog.Info("已重托管", "old", old, "new", newURL)
					if uploads%50 == 0 {
						slog.Info("进度", "已重托管去重老图", uploads, "失败/404", len(dead))
					}
				}
				if newURL != "" && newURL != old {
					n := strings.Count(newContent, old)
					newContent = strings.ReplaceAll(newContent, old, newURL)
					rowReplaced += n
				}
			}

			if *dryRun || rowReplaced == 0 {
				if !*dryRun {
					rowsSkipped++
				}
				continue
			}
			if err := db.Exec(
				"UPDATE "+t.table+" SET "+t.col+" = ? WHERE id = ?",
				newContent, r.ID,
			).Error; err != nil {
				slog.Error("更新行失败", "table", t.table, "id", r.ID, "error", err)
				rowsSkipped++
				continue
			}
			rowsUpdated++
			replacements += rowReplaced
			slog.Info("已改写", "table", t.table, "id", r.ID, "替换处数", rowReplaced)
		}
	}

	if *dryRun {
		fmt.Printf("dry-run 完成。去重老图文件 %d 张待迁移。各表含老图行数: ", len(seen))
		for _, t := range targets {
			fmt.Printf("%s=%d ", t.table, perTableRows[t.table])
		}
		fmt.Printf("\n加 -dry-run=false 执行(顺序跑, ~5.4k 张约 1 小时)。\n")
		return
	}

	slog.Info("迁移完成",
		"重托管去重老图", len(migrated), "改写行数", rowsUpdated, "替换处数", replacements,
		"跳过行数", rowsSkipped, "失败/404老图", len(dead))
	if len(deadList) > 0 {
		sort.Strings(deadList)
		slog.Warn("以下老图取不到(404/失败), 已保留旧 URL, 需人工跟进或忽略", "count", len(deadList), "urls", deadList)
	}
	fmt.Printf("迁移完成: 重托管 %d 张, 改写 %d 行(%d 处), 跳过 %d 行, 失败/404 %d 张。\n",
		len(migrated), rowsUpdated, replacements, rowsSkipped, len(dead))
}

func extractLegacyURLs(content string) []string {
	raw := legacyImageRe.FindAllString(content, -1)
	out := make([]string, 0, len(raw))
	for _, u := range raw {
		out = append(out, strings.TrimRight(u, trailingPunct))
	}
	return dedupe(out)
}

func rewriteHost(url, base string) string {
	if base == "" || base == "https://image.kungal.com" {
		return url
	}
	return strings.Replace(url, "https://image.kungal.com", strings.TrimRight(base, "/"), 1)
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func fetch(ctx context.Context, c *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}
