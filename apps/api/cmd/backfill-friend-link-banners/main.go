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
	"path/filepath"
	"strings"
	"time"

	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dryRun := flag.Bool("dry-run", false, "Read/report planned changes but do not upload or write")
	dir := flag.String("dir", "/app/friends", "Local dir holding the baked static banners; read <dir>/<basename> (primary source)")
	base := flag.String("base", "", "If set AND -dir is empty, HTTP-fetch from <base>/<banner> instead of reading -dir")
	preset := flag.String("preset", "topic", "image_service preset to upload under (topic | galgame_banner)")
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

	type row struct {
		ID     int
		Name   string
		Banner string
	}
	var rows []row
	if err := db.Table("friend_link").
		Select("id, name, banner").
		Where("banner LIKE ?", "/friends/%").
		Order("id ASC").
		Scan(&rows).Error; err != nil {
		slog.Error("查询友链失败", "error", err)
		os.Exit(1)
	}

	source := "dir=" + *dir
	if *dir == "" {
		source = "base=" + *base
	}
	slog.Info("开始回填友链 banner",
		"candidates", len(rows), "source", source, "preset", *preset, "dry_run", *dryRun)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	ctx := context.Background()
	updated, failed := 0, 0

	for _, r := range rows {
		fname := filepath.Base(r.Banner)

		var (
			body []byte
			src  string
			rerr error
		)
		if *dir != "" {
			src = filepath.Join(*dir, fname)
			body, rerr = os.ReadFile(src)
		} else if *base != "" {
			src = strings.TrimRight(*base, "/") + r.Banner
			body, rerr = fetch(ctx, httpClient, src)
		} else {
			rerr = fmt.Errorf("既未提供 -dir 也未提供 -base")
			src = r.Banner
		}
		if rerr != nil {
			slog.Error("读取静态 banner 失败, 跳过", "id", r.ID, "name", r.Name, "src", src, "error", rerr)
			failed++
			continue
		}

		if *dryRun {
			slog.Info("dry-run: 将上传", "id", r.ID, "name", r.Name, "src", src, "bytes", len(body))
			continue
		}

		res, uerr := imgCli.Upload(ctx, bytes.NewReader(body), fname, *preset)
		if uerr != nil {
			slog.Error("上传 image_service 失败, 跳过", "id", r.ID, "name", r.Name, "error", uerr)
			failed++
			continue
		}

		if err := db.Exec(
			"UPDATE friend_link SET banner = ?, updated = now() WHERE id = ?",
			res.URL, r.ID,
		).Error; err != nil {
			slog.Error("更新 banner 失败", "id", r.ID, "name", r.Name, "new", res.URL, "error", err)
			failed++
			continue
		}

		slog.Info("已迁移", "id", r.ID, "name", r.Name, "old", r.Banner, "new", res.URL)
		updated++
	}

	if *dryRun {
		fmt.Printf("dry-run 完成: %d 个候选 (banner LIKE '/friends/%%'), %d 个读取失败\n", len(rows), failed)
	} else {
		fmt.Printf("回填完成: 成功迁移 %d, 失败/跳过 %d, 候选共 %d\n", updated, failed, len(rows))
	}
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
