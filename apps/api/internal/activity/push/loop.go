package push

import (
	"context"
	"log/slog"
	"sync"
	"time"

	activityapiv1 "kun-galgame-api/internal/activity/apiv1"
	cronpkg "kun-galgame-api/internal/infrastructure/cron"
	"kun-galgame-api/pkg/communityclient"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

const (
	busyPause     = 250 * time.Millisecond
	idlePoll      = 5 * time.Second
	backoffStart  = 5 * time.Second
	backoffCap    = 5 * time.Minute
	reconcileSpec = "30 4 * * *"
	tickWait      = 2 * time.Minute
)

type Pusher struct {
	db        *gorm.DB
	community *communityclient.Client
	assemble  *activityapiv1.Service
	origin    string
}

func New(db *gorm.DB, community *communityclient.Client, assemble *activityapiv1.Service, origin string) *Pusher {
	return &Pusher{db: db, community: community, assemble: assemble, origin: origin}
}

func (p *Pusher) ready() bool {
	return p != nil && p.db != nil && p.assemble != nil && p.community != nil && p.community.Configured() && originReady(p.origin)
}

func (p *Pusher) Start() func() {
	if !p.ready() {
		slog.Info("activity push: not started", "community", p != nil && p.community != nil && p.community.Configured(), "https_origin", originReady(p.origin))
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Go(func() { p.drainLoop(ctx) })
	wg.Go(func() { p.reconcileLoop(ctx) })
	return func() {
		cancel()
		wg.Wait()
	}
}

func (p *Pusher) drainLoop(ctx context.Context) {
	had, err := p.runTick(ctx)
	backoff := time.Duration(0)
	for {
		wait := idlePoll
		if err != nil {
			if backoff == 0 {
				backoff = backoffStart
			} else {
				backoff *= 2
				if backoff > backoffCap {
					backoff = backoffCap
				}
			}
			wait = backoff
		} else {
			backoff = 0
			if had {
				wait = busyPause
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		had, err = p.runTick(ctx)
	}
}

func (p *Pusher) runTick(ctx context.Context) (bool, error) {
	runCtx, cancel := context.WithTimeout(ctx, tickWait)
	defer cancel()
	return p.RunOnce(runCtx)
}

func (p *Pusher) reconcileLoop(ctx context.Context) {
	c := cron.New(cron.WithLocation(cronpkg.ScheduleLocation()))
	if _, err := c.AddFunc(reconcileSpec, func() {
		p.Reconcile(context.Background())
	}); err != nil {
		slog.Error("activity push: reconcile schedule failed", "error", err)
		return
	}
	c.Start()
	<-ctx.Done()
	stop := c.Stop()
	<-stop.Done()
}
