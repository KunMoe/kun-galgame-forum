package main

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"time"
)

type seedRow struct {
	WorkID  int64
	UserID  int64
	Created time.Time
}

type parseStats struct {
	Lines   int
	Skipped int
	Folded  int
}

const contributorTSVColumns = 4

func parseContributorTSV(r io.Reader) ([]seedRow, parseStats, error) {
	var stats parseStats
	rows := make([]seedRow, 0, 1024)
	index := map[[2]int64]int{}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		stats.Lines++

		parts := strings.Split(line, "\t")
		if len(parts) < contributorTSVColumns {
			stats.Skipped++
			continue
		}
		workID, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		uid, err2 := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
		created, err3 := parseLedgerTime(strings.TrimSpace(parts[3]))
		if err1 != nil || err2 != nil || err3 != nil || workID <= 0 || uid <= 0 {
			stats.Skipped++
			continue
		}

		key := [2]int64{workID, uid}
		if at, ok := index[key]; ok {
			stats.Folded++
			if created.Before(rows[at].Created) {
				rows[at].Created = created
			}
			continue
		}
		index[key] = len(rows)
		rows = append(rows, seedRow{WorkID: workID, UserID: uid, Created: created})
	}
	return rows, stats, sc.Err()
}

var ledgerTimeLayouts = []string{
	"2006-01-02 15:04:05.999999-07",
	"2006-01-02 15:04:05-07",
	time.RFC3339,
}

func parseLedgerTime(s string) (time.Time, error) {
	var err error
	for _, layout := range ledgerTimeLayouts {
		var t time.Time
		if t, err = time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}
