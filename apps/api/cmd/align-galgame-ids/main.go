package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/renumber"
	"kun-galgame-api/pkg/config"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	mapPath := flag.String("map", "", "TSV from catalog_map.sql: old_id, new_id, how")
	reportDir := flag.String("report", "", "directory written by the run")
	apply := flag.Bool("apply", false, "commit (default: execute every write, then roll back)")
	checkLive := flag.Bool("check-live", false, "compare the map to CatalogWorkIDs; do not write")
	includeDM := flag.Bool("include-dm", false, "also rewrite private-message text")
	flag.Parse()
	if *mapPath == "" || *reportDir == "" {
		fmt.Fprintln(os.Stderr, "-map and -report are required")
		os.Exit(2)
	}
	if *checkLive && *apply {
		fmt.Fprintln(os.Stderr, "-check-live does not write; do not pass -apply")
		os.Exit(2)
	}

	m, err := renumber.LoadMapFile(*mapPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_ = godotenv.Load()
	if *checkLive {
		os.Exit(runCheckLive(m, *reportDir))
	}
	os.Exit(runApply(m, *reportDir, *apply, *includeDM))
}

func runApply(m *renumber.Map, reportDir string, apply, includeDM bool) int {
	db, err := openDB()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	tx := db.Begin()
	if tx.Error != nil {
		fmt.Fprintln(os.Stderr, "begin:", redactErr(tx.Error))
		return 1
	}
	rep, err := renumber.Apply(tx, m, renumber.Options{IncludeDM: includeDM})
	if rep != nil {
		if apply {
			rep.Mode = "apply"
		} else {
			rep.Mode = "dry-run"
		}
	}
	if werr := writeReport(rep, reportDir); werr != nil {
		fmt.Fprintln(os.Stderr, werr)
		_ = tx.Rollback()
		return 1
	}
	if rep != nil {
		fmt.Print(rep.Summary())
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		_ = tx.Rollback()
		return 1
	}
	if apply {
		if err := tx.Commit().Error; err != nil {
			fmt.Fprintln(os.Stderr, "commit:", redactErr(err))
			return 1
		}
		return 0
	}
	if err := tx.Rollback().Error; err != nil {
		fmt.Fprintln(os.Stderr, "rollback:", redactErr(err))
		return 1
	}
	return 0
}

func runCheckLive(m *renumber.Map, reportDir string) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "load config:", err)
		return 1
	}
	db, err := openDB()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	gc := client.New(cfg.NextMoeAPI.BaseURL, cfg.NextMoeAPI.APIKey, cfg.NextMoeAPI.ImageCDNBase)
	rep, err := renumber.CheckLive(context.Background(), db, m, func(ctx context.Context, gids []int) (map[int]int64, error) {
		got, appErr := gc.CatalogWorkIDs(ctx, gids)
		if appErr != nil {
			return nil, appErr
		}
		return got, nil
	})
	if werr := writeReport(rep, reportDir); werr != nil {
		fmt.Fprintln(os.Stderr, werr)
		return 1
	}
	if rep != nil {
		fmt.Print(rep.Summary())
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(rep.LiveMismatches) > 0 {
		fmt.Fprintf(os.Stderr, "%d live mismatch(es)\n", len(rep.LiveMismatches))
		return 1
	}
	return 0
}

func writeReport(rep *renumber.Report, dir string) error {
	if rep == nil {
		return nil
	}
	return rep.WriteDir(dir)
}

func openDB() (*gorm.DB, error) {
	dsn := os.Getenv("KUN_DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("KUN_DATABASE_URL is not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, fmt.Errorf("open database: %s", redactErr(err))
	}
	return db, nil
}

func redactErr(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	if dsn := os.Getenv("KUN_DATABASE_URL"); dsn != "" {
		s = strings.ReplaceAll(s, dsn, "<KUN_DATABASE_URL>")
	}
	return s
}
