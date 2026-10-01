package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/relocation"
	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
)

// Moves the resources of works whose original language is neither Japanese nor
// Chinese to LetMoe. Four steps, each safe to run again:
//
//	census    what would move, and how far a previous run got
//	snapshot  record every candidate in galgame_resource_relocation
//	push      send the snapshots to LetMoe's import face and store its receipts
//	retire    delete the forum's row of every resource LetMoe confirmed
//
// The order is the contract: retire only touches a resource whose receipt is
// stored, and refuses one the uploader changed after the snapshot.
func main() {
	exclude := flag.String("exclude-works", "", "逗号分隔的 work id, 这些作品的资源不迁")
	dryRun := flag.Bool("dry-run", false, "push: 只让 LetMoe 校验不写入; retire: 只报告不删除")
	out := flag.String("out", "", "snapshot: 另存一份 JSONL 到此路径 (含链接与密码, 勿入库)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "用法: relocate-resources [flags] census|snapshot|push|retire")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	excluded, err := parseIDs(*exclude)
	if err != nil {
		fmt.Fprintln(os.Stderr, "exclude-works:", err)
		os.Exit(2)
	}

	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)
	store := relocation.NewStore(database.NewPostgres(cfg.Database, cfg.Server.Mode))
	ctx := context.Background()

	switch flag.Arg(0) {
	case "census":
		err = census(store, excluded)
	case "snapshot":
		gc := client.New(cfg.NextMoeAPI.BaseURL, cfg.NextMoeAPI.APIKey, cfg.NextMoeAPI.ImageCDNBase)
		err = snapshot(ctx, store, gc, excluded, *out)
	case "push":
		err = push(ctx, store, *dryRun)
	case "retire":
		err = retire(store, *dryRun)
	default:
		flag.Usage()
		os.Exit(2)
	}
	if err != nil {
		slog.Error("relocate-resources 失败", "step", flag.Arg(0), "error", err)
		os.Exit(1)
	}
}

func parseIDs(raw string) ([]int, error) {
	ids := []int{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("%q is not a work id", part)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func census(store *relocation.Store, excluded []int) error {
	c, err := store.Census(excluded)
	if err != nil {
		return err
	}
	slog.Info("迁移普查", "works", c.Works, "resources", c.Resources, "uploaders", c.Uploaders,
		"excluded_works", len(excluded), "snapshots", c.Snapshots, "pushed", c.Pushed, "retired", c.Retired)
	return nil
}

// The local column is a cache and the snapshot decides what leaves the forum,
// so every candidate work is asked about on catalog before one row is written.
func snapshot(ctx context.Context, store *relocation.Store, gc *client.GalgameClient, excluded []int, out string) error {
	works, err := store.CandidateWorks(excluded)
	if err != nil {
		return err
	}
	ids := make([]int64, len(works))
	for i, w := range works {
		ids[i] = int64(w.WorkID)
	}
	rendered, hidden, appErr := gc.MirrorByCatalogIDs(ctx, ids)
	if appErr != nil {
		return fmt.Errorf("catalog: %s", appErr.Message)
	}
	disagreed := 0
	for _, w := range works {
		row, ok := rendered[w.WorkID]
		if !ok {
			row, ok = hidden[w.WorkID]
		}
		if !ok || row.OriginalLanguage != w.Language {
			disagreed++
			slog.Error("本地原语言与 catalog 不一致", "work_id", w.WorkID, "local", w.Language, "catalog", row.OriginalLanguage, "answered", ok)
		}
	}
	if disagreed > 0 {
		return fmt.Errorf("%d works disagree with catalog; let the mirror settle or pass them to -exclude-works", disagreed)
	}
	n, err := store.Snapshot(excluded)
	if err != nil {
		return err
	}
	slog.Info("已记录快照", "works", len(works), "written", n)
	if out == "" {
		return nil
	}
	rows, err := store.All()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for _, row := range rows {
		if _, err := w.Write(append(row.Payload, '\n')); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := w.Flush(); err != nil {
		_ = f.Close()
		return err
	}
	slog.Info("已写出 JSONL", "path", out, "lines", len(rows))
	return f.Close()
}

func push(ctx context.Context, store *relocation.Store, dryRun bool) error {
	url, key := os.Getenv("KUN_LETMOE_IMPORT_URL"), os.Getenv("KUN_LETMOE_IMPORT_KEY")
	if url == "" || key == "" {
		return fmt.Errorf("KUN_LETMOE_IMPORT_URL and KUN_LETMOE_IMPORT_KEY must both be set")
	}
	letmoe := relocation.NewLetMoe(url, key)
	// A dry run stores nothing, so it would be handed the same first batch for ever.
	rows, err := store.Unpushed(1 << 30)
	if err != nil {
		return err
	}
	var landed, valid, failed int
	for start := 0; start < len(rows); start += relocation.BatchSize {
		batch := rows[start:min(start+relocation.BatchSize, len(rows))]
		items := make([]json.RawMessage, len(batch))
		for i, row := range batch {
			items[i] = row.Payload
		}
		receipts, err := letmoe.Import(ctx, items, dryRun)
		if err != nil {
			return fmt.Errorf("batch at %d: %w", start, err)
		}
		for i, r := range receipts {
			if r.ForumID != batch[i].ResourceID {
				return fmt.Errorf("batch at %d: receipt %d names forum_id %d, sent %d", start, i, r.ForumID, batch[i].ResourceID)
			}
			switch {
			case r.Landed():
				if err := store.SaveReceipt(r.ForumID, r.ResourceID, r.Public); err != nil {
					return err
				}
				landed++
			case dryRun && r.Result == "valid":
				valid++
			default:
				failed++
				slog.Error("LetMoe 拒收", "forum_id", r.ForumID, "result", r.Result, "error", r.Error)
			}
		}
		slog.Info("批次完成", "from", start, "size", len(batch), "landed", landed, "valid", valid, "failed", failed)
	}
	if failed > 0 {
		return fmt.Errorf("%d resources were refused", failed)
	}
	return nil
}

func retire(store *relocation.Store, dryRun bool) error {
	ids, err := store.Retirable()
	if err != nil {
		return err
	}
	if dryRun {
		slog.Info("可下架", "resources", len(ids))
		return nil
	}
	var retired, changed int
	started := time.Now()
	for _, id := range ids {
		switch err := store.Retire(id); err {
		case nil:
			retired++
		case relocation.ErrChanged:
			changed++
			slog.Warn("快照之后资源有改动或已不存在, 未下架", "resource_id", id)
		default:
			return fmt.Errorf("resource %d: %w", id, err)
		}
	}
	slog.Info("下架完成", "retired", retired, "changed", changed, "elapsed", time.Since(started).Round(time.Millisecond))
	if changed > 0 {
		return fmt.Errorf("%d resources changed after their snapshot; snapshot and push again", changed)
	}
	return nil
}
