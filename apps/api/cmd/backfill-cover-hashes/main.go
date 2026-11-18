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
	"path/filepath"
	"strings"
	"time"

	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type target struct {
	table   string
	urlCol  string
	hashCol string
}

var targets = []target{
	{"doc_article", "banner", "banner_image_hash"},
	{"friend_link", "banner", "banner_image_hash"},
	{"galgame_website", "icon", "icon_image_hash"},
}

func main() {
	_ = godotenv.Load()

	dryRun := flag.Bool("dry-run", false, "Report planned changes but do not upload or write")
	preset := flag.String("preset", "topic", "image_service preset for re-uploads")
	webroot := flag.String("webroot", "", "Local dir to resolve root-relative URLs (e.g. /content/x/banner.avif) against, e.g. ../web/public")
	base := flag.String("base", "", "HTTP origin to prepend to root-relative URLs when -webroot is unset, e.g. https://www.kungal.com")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	cdnBase := strings.TrimRight(cfg.NextMoeAPI.ImageCDNBase, "/")
	imgCli := imageclient.New(imageclient.Config{
		BaseURL:      cfg.ImageClient.BaseURL,
		CDNBase:      cfg.NextMoeAPI.ImageCDNBase,
		ClientID:     cfg.ImageClient.ClientID,
		ClientSecret: cfg.ImageClient.ClientSecret,
	})

	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)
	httpClient := &http.Client{Timeout: 30 * time.Second}
	ctx := context.Background()

	totalUpdated, totalFailed := 0, 0
	for _, t := range targets {
		u, f := backfillTarget(ctx, db, imgCli, httpClient, cdnBase, t, *preset, *webroot, *base, *dryRun)
		totalUpdated += u
		totalFailed += f
	}
	fmt.Printf("封面 hash 回填完成: 成功 %d, 失败/跳过 %d\n", totalUpdated, totalFailed)
}

func backfillTarget(
	ctx context.Context, db *gorm.DB, imgCli *imageclient.Client, hc *http.Client,
	cdnBase string, t target, preset, webroot, base string, dryRun bool,
) (updated, failed int) {
	type row struct {
		ID  int
		URL string
	}
	var rows []row
	if err := db.Table(t.table).
		Select(fmt.Sprintf("id, %s AS url", t.urlCol)).
		Where(fmt.Sprintf("%s <> '' AND %s = ''", t.urlCol, t.hashCol)).
		Order("id ASC").
		Scan(&rows).Error; err != nil {
		slog.Error("查询失败", "table", t.table, "error", err)
		return 0, 0
	}
	slog.Info("开始回填", "table", t.table, "candidates", len(rows), "dry_run", dryRun)

	for _, r := range rows {
		r.URL = strings.TrimSpace(r.URL)
		if r.URL == "" {
			continue
		}

		if cdnBase != "" && strings.HasPrefix(r.URL, cdnBase) {
			hash := hashFromImageURL(r.URL)
			if hash == "" {
				slog.Error("无法从 image_service URL 解析 hash, 跳过", "table", t.table, "id", r.ID, "url", r.URL)
				failed++
				continue
			}
			if dryRun {
				slog.Info("dry-run: 解析得到 hash", "table", t.table, "id", r.ID, "hash", hash)
				continue
			}
			if writeHash(db, t, r.ID, hash) {
				updated++
			} else {
				failed++
			}
			continue
		}

		if dryRun {
			slog.Info("dry-run: 将抓取并上传", "table", t.table, "id", r.ID, "url", r.URL)
			continue
		}
		body, err := getImageBytes(ctx, hc, r.URL, webroot, base)
		if err != nil {
			slog.Error("获取图片失败, 跳过", "table", t.table, "id", r.ID, "url", r.URL, "error", err)
			failed++
			continue
		}
		res, uerr := imgCli.Upload(ctx, bytes.NewReader(body), "cover", preset)
		if uerr != nil {
			slog.Error("上传 image_service 失败, 跳过", "table", t.table, "id", r.ID, "error", uerr)
			failed++
			continue
		}
		if writeHash(db, t, r.ID, res.Hash) {
			slog.Info("已回填", "table", t.table, "id", r.ID, "old", r.URL, "hash", res.Hash)
			updated++
		} else {
			failed++
		}
	}
	return updated, failed
}

func writeHash(db *gorm.DB, t target, id int, hash string) bool {
	if err := db.Exec(
		fmt.Sprintf("UPDATE %s SET %s = ? WHERE id = ?", t.table, t.hashCol),
		hash, id,
	).Error; err != nil {
		slog.Error("写入 hash 失败", "table", t.table, "id", id, "error", err)
		return false
	}
	return true
}

func hashFromImageURL(u string) string {
	base := path.Base(u)
	base = strings.SplitN(base, ".", 2)[0]
	base = strings.SplitN(base, "_", 2)[0]
	if len(base) < 4 {
		return ""
	}
	return base
}

func getImageBytes(ctx context.Context, hc *http.Client, rawURL, webroot, base string) ([]byte, error) {
	if strings.HasPrefix(rawURL, "/") {
		if webroot != "" {
			return os.ReadFile(filepath.Join(webroot, filepath.FromSlash(rawURL)))
		}
		if base != "" {
			return fetch(ctx, hc, strings.TrimRight(base, "/")+rawURL)
		}
		return nil, fmt.Errorf("根相对 URL 需要 -webroot 或 -base 才能获取")
	}
	return fetch(ctx, hc, rawURL)
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
	return io.ReadAll(resp.Body)
}
