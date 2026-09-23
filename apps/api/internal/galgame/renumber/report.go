package renumber

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FoldRecord struct {
	NewID    int64
	Survivor int64
	Losers   []int64
	Moved    int64
	Dropped  int64
	Comments int
}

type TableStat struct {
	Name       string
	Rows       int64
	Changed    int64
	Merged     int64
	Archived   int64
	SumBefore  int64
	SumAfter   int64
	DroppedSum int64
}

type LinkStat struct {
	Source    string
	Matching  int64
	Changing  int64
	Rewritten int64
}

type ContentChange struct {
	Table  string
	ID     int64
	Before string
	After  string
}

type LiveMismatch struct {
	OldID     int64
	MapNewID  int64
	LiveNewID int64
}

type Report struct {
	Mode      string
	Started   time.Time
	Ended     time.Time
	MapLen    int
	TSVRows   int
	HowCounts map[string]int

	GalgameBefore int64
	GalgameAfter  int64
	FoldedLosers  int

	Folds []FoldRecord

	Tables map[string]*TableStat
	Links  map[string]*LinkStat

	Content []ContentChange

	TriggersDisabled []string
	TriggersEnabled  []string
	ParkOccupied     []string

	LiveMismatches []LiveMismatch
}

func newReport() *Report {
	return &Report{
		Started:   time.Now(),
		HowCounts: map[string]int{},
		Tables:    map[string]*TableStat{},
		Links:     map[string]*LinkStat{},
	}
}

func (r *Report) table(name string) *TableStat {
	s, ok := r.Tables[name]
	if !ok {
		s = &TableStat{Name: name}
		r.Tables[name] = s
	}
	return s
}

func (r *Report) link(source string) *LinkStat {
	s, ok := r.Links[source]
	if !ok {
		s = &LinkStat{Source: source}
		r.Links[source] = s
	}
	return s
}

func (r *Report) Summary() string {
	var b strings.Builder
	mode := r.Mode
	if mode == "" {
		mode = "dry-run"
	}
	fmt.Fprintf(&b, "align-galgame-ids %s\n", mode)
	fmt.Fprintf(&b, "start %s\n", r.Started.UTC().Format(time.RFC3339Nano))
	fmt.Fprintf(&b, "end   %s\n", r.Ended.UTC().Format(time.RFC3339Nano))
	fmt.Fprintf(&b, "map entries %d (tsv %d): curated=%d claim=%d unchanged=%d\n",
		r.MapLen, r.TSVRows, r.HowCounts[HowCurated], r.HowCounts[HowClaim], r.HowCounts[HowUnchanged])
	fmt.Fprintf(&b, "galgame rows before=%d after=%d folded_losers=%d folds=%d\n",
		r.GalgameBefore, r.GalgameAfter, r.FoldedLosers, len(r.Folds))
	for _, f := range r.Folds {
		fmt.Fprintf(&b, "  fold new=%d survivor=%d losers=%s moved=%d dropped=%d comments=%d\n",
			f.NewID, f.Survivor, joinIDs(f.Losers), f.Moved, f.Dropped, f.Comments)
	}
	names := make([]string, 0, len(r.Tables))
	for name := range r.Tables {
		names = append(names, name)
	}
	sort.Strings(names)
	b.WriteString("tables\n")
	for _, name := range names {
		s := r.Tables[name]
		fmt.Fprintf(&b, "  %s rows=%d changed=%d merged=%d archived=%d sum_before=%d sum_after=%d dropped_sum=%d\n",
			s.Name, s.Rows, s.Changed, s.Merged, s.Archived, s.SumBefore, s.SumAfter, s.DroppedSum)
	}
	sources := make([]string, 0, len(r.Links))
	for src := range r.Links {
		sources = append(sources, src)
	}
	sort.Strings(sources)
	b.WriteString("links\n")
	for _, src := range sources {
		s := r.Links[src]
		fmt.Fprintf(&b, "  %s matching=%d changing=%d rewritten=%d\n",
			s.Source, s.Matching, s.Changing, s.Rewritten)
	}
	fmt.Fprintf(&b, "user-content rows %d\n", len(r.Content))
	fmt.Fprintf(&b, "triggers disabled [%s]\n", strings.Join(r.TriggersDisabled, ", "))
	fmt.Fprintf(&b, "triggers enabled  [%s]\n", strings.Join(r.TriggersEnabled, ", "))
	if len(r.ParkOccupied) > 0 {
		fmt.Fprintf(&b, "park occupied: %s\n", strings.Join(r.ParkOccupied, ", "))
	}
	if len(r.LiveMismatches) > 0 {
		fmt.Fprintf(&b, "live mismatches %d\n", len(r.LiveMismatches))
	}
	return b.String()
}

func (r *Report) WriteDir(dir string) error {
	if r.Ended.IsZero() {
		r.Ended = time.Now()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "summary.txt"), []byte(r.Summary()), 0o644); err != nil {
		return err
	}
	if err := writeFoldsTSV(filepath.Join(dir, "folds.tsv"), r.Folds); err != nil {
		return err
	}
	if err := writeContentTSV(filepath.Join(dir, "user-content.tsv"), r.Content); err != nil {
		return err
	}
	if r.LiveMismatches != nil {
		if err := writeLiveTSV(filepath.Join(dir, "live-mismatch.tsv"), r.LiveMismatches); err != nil {
			return err
		}
	}
	return nil
}

func writeFoldsTSV(path string, folds []FoldRecord) error {
	var b strings.Builder
	b.WriteString("new_id\tsurvivor\tlosers\tmoved\tdropped\tcomments\n")
	for _, f := range folds {
		fmt.Fprintf(&b, "%d\t%d\t%s\t%d\t%d\t%d\n",
			f.NewID, f.Survivor, joinIDs(f.Losers), f.Moved, f.Dropped, f.Comments)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeContentTSV(path string, rows []ContentChange) error {
	var b strings.Builder
	b.WriteString("table\tid\tbefore\tafter\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "%s\t%d\t%s\t%s\n", row.Table, row.ID, escapeTSV(row.Before), escapeTSV(row.After))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeLiveTSV(path string, rows []LiveMismatch) error {
	var b strings.Builder
	b.WriteString("old_id\tmap_new_id\tlive_new_id\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "%d\t%d\t%d\n", row.OldID, row.MapNewID, row.LiveNewID)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func escapeTSV(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

func joinIDs(ids []int64) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return strings.Join(parts, ",")
}
