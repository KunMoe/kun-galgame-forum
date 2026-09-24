// purge-restore undoes one user-content purge: the forum's rows from
// user_purge_archive, then the target's community posts. It runs in the tools
// container, which carries the API's environment, so neither the database nor
// the S2S credentials appear on a command line:
//
//	docker compose -f docker-compose.prod.yml -p kun-visual-novel-forum-iunwa9 \
//	  run --rm tools purge-restore [-commit] <purge_id>
//
// Without -commit the forum restore is rolled back after its report is printed
// and community is not called. See docs/proj/api-v1/waves/u3c-purge.md §6.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"kun-galgame-api/internal/admin/repository"
	"kun-galgame-api/internal/admin/service"
	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	commit := flag.Bool("commit", false, "apply the restore; without it the forum restore is rolled back and community is not called")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: purge-restore [-commit] <purge_id>")
		os.Exit(2)
	}
	purgeID, err := uuid.Parse(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "purge id is not a UUID:", err)
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "load config:", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)
	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)

	clientID, clientSecret := cfg.Community.ClientID, cfg.Community.ClientSecret
	if clientID == "" {
		clientID = cfg.OAuth.ClientID
	}
	if clientSecret == "" {
		clientSecret = cfg.OAuth.ClientSecret
	}
	community := communityclient.New(communityclient.Config{
		BaseURL: cfg.Community.BaseURL, ClientID: clientID, ClientSecret: clientSecret,
	})

	svc := service.NewPurgeService(repository.NewPurgeRepository(db), nil, community, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	report, err := svc.Restore(ctx, purgeID, *commit)
	printReport(purgeID, *commit, report)
	if err != nil {
		fmt.Fprintln(os.Stderr, "restore failed:", err)
		os.Exit(1)
	}
}

func printReport(purgeID uuid.UUID, commit bool, r service.PurgeRestoreReport) {
	if r.Local.TargetUserID == 0 {
		return
	}
	fmt.Printf("purge %s, target user %d\n\n", purgeID, r.Local.TargetUserID)
	switch {
	case r.Local.AlreadyRestored:
		fmt.Println("forum database: already restored")
	case commit:
		fmt.Println("forum database: restored and committed")
	default:
		fmt.Println("forum database: dry run, rolled back")
	}
	for _, row := range r.Local.Rows {
		note := ""
		if row.Restored != row.Archived {
			note = fmt.Sprintf("  (%d skipped)", row.Archived-row.Restored)
		}
		fmt.Printf("  %-30s %-7s %6d of %6d%s\n", row.TableName, row.Operation, row.Restored, row.Archived, note)
	}
	fmt.Println()
	switch {
	case !commit:
		fmt.Println("community: not called in a dry run; rerun with -commit")
	case r.CommunityNothing:
		fmt.Printf("community: nothing to restore (%s)\n", r.CommunityResponse)
	case r.Community != nil:
		c := r.Community
		fmt.Printf("community: posts %d, reactions %d, read states %d, anchor subscriptions %d, notifications %d\n",
			c.PostsRestored, c.ReactionsRestored, c.ReadStatesRestored, c.AnchorSubscriptionsRestored, c.NotificationsRestored)
	}
}
