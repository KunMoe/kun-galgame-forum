package cron

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/infrastructure/viewstats"
	"kun-galgame-api/pkg/imageclient"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

const scheduleTZ = "Asia/Shanghai"

// Every field is a bare func(), so positional arguments would put a job on
// another job's schedule with nothing to notice it.
type Jobs struct {
	GalgameClaimSync         func()
	GalgameRevisionSync      func()
	GalgameContributorSync   func()
	GalgameCatalogMirror     func()
	GalgameCatalogMirrorFill func()
	GalgameMergeSync         func()
	DlsiteCampaignRefresh    func()
	TopicMiniAppDeadlines    func()
}

func Start(
	db *gorm.DB,
	rdb *redis.Client,
	imgCli *imageclient.Client,
	jobs Jobs,
) func() {
	loc, err := time.LoadLocation(scheduleTZ)
	if err != nil {
		slog.Warn("加载定时任务时区失败, 回退到进程本地时区", "tz", scheduleTZ, "error", err)
		loc = time.Local
	}
	c := cron.New(cron.WithLocation(loc))

	schedule(c, "0 0 * * *", "每日计数重置", func() {
		resetDaily(db)
	})

	schedule(c, "0 0 * * *", "浏览量滚动统计", func() {
		if err := viewstats.RunRollup(db); err != nil {
			slog.Error("浏览量滚动统计失败", "error", err)
			return
		}
		slog.Info("浏览量滚动统计完成")
	})

	schedule(c, "0 * * * *", "上传缓存清理", func() {
		cleanupUploadCache(rdb)
	})

	if imgCli != nil {
		schedule(c, "0 4 * * *", "内容图 reference-ping", func() {
			distinct, updated, err := RunReferencePing(context.Background(), db, imgCli)
			if err != nil {
				slog.Error("内容图 reference-ping 失败", "distinct", distinct, "updated", updated, "error", err)
				return
			}
			slog.Info("内容图 reference-ping 完成", "distinct_hashes", distinct, "updated", updated)
		})
	} else {
		slog.Warn("image client 未配置, 跳过内容图 reference-ping —— 内容图存在被 image-gc 回收的风险")
	}

	if jobs.GalgameClaimSync != nil {
		schedule(c, "*/10 * * * *", "galgame claim 同步", jobs.GalgameClaimSync)
	}

	if jobs.GalgameRevisionSync != nil {
		schedule(c, "*/10 * * * *", "galgame revision 同步", jobs.GalgameRevisionSync)
	}

	if jobs.GalgameContributorSync != nil {
		schedule(c, "*/15 * * * *", "galgame contributor 同步", jobs.GalgameContributorSync)
	}

	if jobs.GalgameCatalogMirror != nil {
		schedule(c, "*/10 * * * *", "galgame catalog 镜像信道同步", jobs.GalgameCatalogMirror)
	}

	// Two minutes, not ten. The lane costs one indexed query when there is
	// nothing pending, which is the normal state, and the beat is what decides
	// two things that are not normal: how long a row created this minute is
	// shown to every SFW reader before its verdict lands, and how long a
	// backfill takes. 092 left 11.5k rows unconfirmed at once; at ten minutes
	// that sweep is two hours, at two it is twenty-five minutes, and the burst
	// per pass is the same ~30 catalog requests either way.
	if jobs.GalgameCatalogMirrorFill != nil {
		schedule(c, "*/2 * * * *", "galgame catalog 镜像增量同步", jobs.GalgameCatalogMirrorFill)
	}

	if jobs.GalgameMergeSync != nil {
		schedule(c, "*/30 * * * *", "galgame 合并同步", jobs.GalgameMergeSync)
	}

	// Every minute, not every ten: a lottery deadline is a promise to the people
	// waiting for it, and the sweep is one indexed query against two tables.
	if jobs.TopicMiniAppDeadlines != nil {
		schedule(c, "* * * * *", "话题小程序到点处理", jobs.TopicMiniAppDeadlines)
	}

	if jobs.DlsiteCampaignRefresh != nil {
		schedule(c, "*/10 * * * *", "dlsite 优惠券活动刷新", jobs.DlsiteCampaignRefresh)
	}

	c.Start()
	slog.Info("定时任务已启动")

	return func() {
		ctx := c.Stop()
		<-ctx.Done()
		slog.Info("定时任务已停止")
	}
}

// AddFunc returns an error for a malformed spec, and a job that never registers
// is indistinguishable from one that registers and does nothing — the same
// shape as the resetDaily incident below.
func schedule(c *cron.Cron, spec, name string, fn func()) {
	if _, err := c.AddFunc(spec, fn); err != nil {
		slog.Error("定时任务注册失败, 该任务不会运行", "task", name, "spec", spec, "error", err)
	}
}

// Targets `kungal_user_state`, NOT the old `"user"` table — migration 007 moved
// the daily_* columns. The original `UPDATE "user" SET daily_* = 0` silently
// errored every midnight after that migration, so users who hit their daily
// caps stayed capped indefinitely.
func resetDaily(db *gorm.DB) {
	result := db.Exec(`
		UPDATE kungal_user_state SET
			daily_check_in = 0,
			daily_image_count = 0,
			daily_toolset_upload_count = 0,
			daily_toolset_upload_bytes = 0
		WHERE daily_check_in != 0
		   OR daily_image_count != 0
		   OR daily_toolset_upload_count != 0
		   OR daily_toolset_upload_bytes != 0
	`)
	if result.Error != nil {
		slog.Error("每日重置失败", "error", result.Error)
		return
	}
	slog.Info("每日重置完成", "affected", result.RowsAffected)
}

func cleanupUploadCache(rdb *redis.Client) {
	ctx := context.Background()
	keys, err := rdb.Keys(ctx, "toolset:upload:*").Result()
	if err != nil {
		slog.Error("扫描上传缓存失败", "error", err)
		return
	}

	if len(keys) == 0 {
		return
	}

	deleted := 0
	for _, key := range keys {
		ttl, _ := rdb.TTL(ctx, key).Result()
		if ttl <= 0 {
			rdb.Del(ctx, key)
			deleted++
		}
	}

	if deleted > 0 {
		slog.Info("清理上传缓存完成", "deleted", deleted)
	}
}
