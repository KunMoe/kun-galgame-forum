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

const presentationReconcileSpec = "0 5 * * *"

type Presenter struct {
	db        *gorm.DB
	community *communityclient.Client
	users     activityapiv1.Users
	catalog   activityapiv1.Catalog
	cdn       string
	origin    string
}

func NewPresenter(db *gorm.DB, community *communityclient.Client, users activityapiv1.Users, catalog activityapiv1.Catalog, cdn, origin string) *Presenter {
	return &Presenter{db: db, community: community, users: users, catalog: catalog, cdn: cdn, origin: origin}
}

func (p *Presenter) Ready() bool { return p.ready() }

func (p *Presenter) ready() bool {
	return p != nil && p.db != nil && p.users != nil && p.catalog != nil &&
		p.community != nil && p.community.Configured() && originReady(p.origin)
}

func (p *Presenter) Start() func() {
	if !p.ready() {
		slog.Info("anchor presentations: not started",
			"community", p != nil && p.community != nil && p.community.Configured(),
			"https_origin", originReady(p.origin))
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

func (p *Presenter) drainLoop(ctx context.Context) {
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

func (p *Presenter) runTick(ctx context.Context) (bool, error) {
	runCtx, cancel := context.WithTimeout(ctx, tickWait)
	defer cancel()
	return p.RunOnce(runCtx)
}

func (p *Presenter) reconcileLoop(ctx context.Context) {
	c := cron.New(cron.WithLocation(cronpkg.ScheduleLocation()))
	if _, err := c.AddFunc(presentationReconcileSpec, func() {
		p.Reconcile(context.Background())
	}); err != nil {
		slog.Error("anchor presentations: reconcile schedule failed", "error", err)
		return
	}
	c.Start()
	<-ctx.Done()
	stop := c.Stop()
	<-stop.Done()
}
