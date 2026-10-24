package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/internal/topic/ai"
	"kun-galgame-api/pkg/aiclient"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// Reclassifies existing topics and writes their summaries. Every step can run
// again: a topic that already has a backfill or model verdict, or a summary,
// is skipped.
//
//	sample     classify a random sample, write old → new to a TSV; writes nothing
//	classify   classify every topic without a verdict and move it there
//	summarize  write a summary for every long topic that has none
//	export     write the verdicts and summaries to JSONL
//	import     apply an export: verdicts to topics without one, summaries to
//	           topics whose body is still the one that was summarised
//
// History moves no moemoepoint: a topic moved out of or into a paid section
// keeps what its author paid or earned when it was published.
func main() {
	n := flag.Int("n", 100, "sample: how many topics")
	out := flag.String("out", "topic-ai-sample.tsv", "sample / export: output path")
	in := flag.String("in", "", "import: JSONL from export")
	model := flag.String("model", "@cf/deepseek-ai/deepseek-v4.1-flash-1017", "model id")
	workers := flag.Int("workers", 2, "concurrent model calls; the endpoint limits requests per minute")
	limit := flag.Int("limit", 0, "classify / summarize: at most this many topics, 0 for all")
	minRunes := flag.Int("min-runes", 500, "summarize: body text needed for a summary")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "用法: topic-ai-backfill [flags] sample|classify|summarize|export|import")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)
	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)
	client := aiclient.New(cfg.AI.BaseURL, cfg.AI.Token)
	needsModel := map[string]bool{"sample": true, "classify": true, "summarize": true}
	if needsModel[flag.Arg(0)] && client == nil {
		slog.Error("KUN_AI_BASE_URL / KUN_AI_TOKEN 未设置")
		os.Exit(1)
	}
	b := &backfill{db: db, client: client, model: *model, workers: *workers}
	ctx := context.Background()

	switch flag.Arg(0) {
	case "sample":
		err = b.sample(ctx, *n, *out)
	case "classify":
		err = b.classify(ctx, *limit)
	case "summarize":
		err = b.summarize(ctx, *limit, *minRunes)
	case "export":
		err = b.export(*out)
	case "import":
		err = b.importFile(*in)
	default:
		flag.Usage()
		os.Exit(2)
	}
	if err != nil {
		slog.Error("失败", "error", err)
		os.Exit(1)
	}
}

type backfill struct {
	db      *gorm.DB
	client  *aiclient.Client
	model   string
	workers int
}

type topicRow struct {
	ID      int
	Title   string
	Content string
	Legacy  string
	Current string
}

type choiceRow struct {
	Name        string
	Label       string
	Description string
	IsArchived  bool
}

const topicSelect = `SELECT t.id, t.title, t.content,
	COALESCE((SELECT string_agg(l.name, ',' ORDER BY l.position) FROM topic_section_legacy_relation l WHERE l.topic_id = t.id), '') AS legacy,
	COALESCE((SELECT ts.name FROM topic_section_relation r JOIN topic_section ts ON ts.id = r.topic_section_id WHERE r.topic_id = t.id), '') AS current
	FROM topic t`

func (b *backfill) choices() ([]choiceRow, error) {
	var rows []choiceRow
	err := b.db.Raw(`SELECT ts.name, ts.label, ts.description, ts.is_archived
		FROM topic_section ts JOIN topic_section_group g ON g.id = ts.group_id
		ORDER BY g.position, ts.position`).Scan(&rows).Error
	return rows, err
}

// choicesFor offers the archived resource-release section only to topics that
// were filed under g-other, where early users posted game resources.
func choicesFor(all []choiceRow, legacy string) []ai.SectionChoice {
	out := make([]ai.SectionChoice, 0, len(all))
	for _, c := range all {
		if c.IsArchived && !(c.Name == "resource-release" && strings.Contains(legacy, "g-other")) {
			continue
		}
		out = append(out, ai.SectionChoice{Key: c.Name, Label: c.Label, Description: c.Description})
	}
	return out
}

// each runs fn over rows with b.workers model calls at a time and stops
// handing out work after the first error it sees.
func (b *backfill) each(rows []topicRow, fn func(topicRow) error) (done int64, err error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var count atomic.Int64
	sem := make(chan struct{}, max(b.workers, 1))
	for _, r := range rows {
		mu.Lock()
		stop := err != nil
		mu.Unlock()
		if stop {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(r topicRow) {
			defer wg.Done()
			defer func() { <-sem }()
			if e := fn(r); e != nil {
				mu.Lock()
				if err == nil {
					err = fmt.Errorf("topic %d: %w", r.ID, e)
				}
				mu.Unlock()
				return
			}
			if c := count.Add(1); c%50 == 0 {
				slog.Info("进度", "done", c, "of", len(rows))
			}
		}(r)
	}
	wg.Wait()
	return count.Load(), err
}

func (b *backfill) sample(ctx context.Context, n int, out string) error {
	choices, err := b.choices()
	if err != nil {
		return err
	}
	var topics []topicRow
	if err := b.db.Raw(topicSelect+` WHERE t.status = 0 ORDER BY random() LIMIT ?`, n).Scan(&topics).Error; err != nil {
		return err
	}
	verdicts := make([]ai.Verdict, len(topics))
	errs := make([]error, len(topics))
	index := map[int]int{}
	for i, t := range topics {
		index[t.ID] = i
	}
	_, _ = b.each(topics, func(t topicRow) error {
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		i := index[t.ID]
		verdicts[i], errs[i] = ai.Classify(cctx, b.client, b.model, choicesFor(choices, t.Legacy), t.Title, t.Content)
		return nil
	})

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Comma = '\t'
	_ = w.Write([]string{"id", "legacy", "migrated", "model", "changed", "chars", "images", "reason", "title"})
	failed := 0
	for i, t := range topics {
		runes, images := ai.Signals(t.Content)
		section, reason := verdicts[i].Section, verdicts[i].Reason
		if errs[i] != nil {
			failed++
			section, reason = "ERROR", errs[i].Error()
		}
		changed := ""
		if section != t.Current {
			changed = "*"
		}
		_ = w.Write([]string{strconv.Itoa(t.ID), t.Legacy, t.Current, section, changed,
			strconv.Itoa(runes), strconv.Itoa(images), reason, t.Title})
	}
	w.Flush()
	slog.Info("完成", "topics", len(topics), "failed", failed, "out", out)
	return w.Error()
}

func (b *backfill) classify(ctx context.Context, limit int) error {
	choices, err := b.choices()
	if err != nil {
		return err
	}
	q := topicSelect + ` WHERE NOT EXISTS (SELECT 1 FROM topic_classification c WHERE c.topic_id = t.id AND c.source <> 'fallback') ORDER BY t.id`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	var topics []topicRow
	if err := b.db.Raw(q).Scan(&topics).Error; err != nil {
		return err
	}
	slog.Info("待分类", "topics", len(topics))
	done, err := b.each(topics, func(t topicRow) error {
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		v, err := ai.Classify(cctx, b.client, b.model, choicesFor(choices, t.Legacy), t.Title, t.Content)
		if err != nil {
			return err
		}
		return b.apply(t.ID, v.Section, v.Reason, b.model)
	})
	slog.Info("分类完成", "done", done, "of", len(topics))
	return err
}

// apply records the verdict and files the topic under it.
func (b *backfill) apply(topicID int, section, reason, model string) error {
	return b.db.Transaction(func(tx *gorm.DB) error {
		var target struct {
			ID             int
			LegacyCategory string
		}
		if err := tx.Raw(`SELECT ts.id, g.legacy_category FROM topic_section ts
			JOIN topic_section_group g ON g.id = ts.group_id WHERE ts.name = ?`, section).Scan(&target).Error; err != nil {
			return err
		}
		if target.ID == 0 {
			return fmt.Errorf("section %q does not exist", section)
		}
		if err := tx.Exec(`DELETE FROM topic_section_relation WHERE topic_id = ?`, topicID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO topic_section_relation (topic_id, topic_section_id, position, updated)
			VALUES (?, ?, 0, CURRENT_TIMESTAMP)`, topicID, target.ID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE topic SET category = ? WHERE id = ? AND category IS DISTINCT FROM ?`,
			target.LegacyCategory, topicID, target.LegacyCategory).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO topic_classification (topic_id, section, source, model, reason)
			VALUES (?, ?, 'backfill', ?, ?)
			ON CONFLICT (topic_id) DO UPDATE SET section = EXCLUDED.section, source = 'backfill',
				model = EXCLUDED.model, reason = EXCLUDED.reason, updated = CURRENT_TIMESTAMP`,
			topicID, section, model, reason).Error
	})
}

func (b *backfill) summarize(ctx context.Context, limit, minRunes int) error {
	q := topicSelect + ` WHERE t.ai_summary IS NULL AND t.ai_summary_at IS NULL ORDER BY t.id`
	var all []topicRow
	if err := b.db.Raw(q).Scan(&all).Error; err != nil {
		return err
	}
	var topics []topicRow
	for _, t := range all {
		if runes, _ := ai.Signals(t.Content); runes >= minRunes {
			topics = append(topics, t)
		}
		if limit > 0 && len(topics) >= limit {
			break
		}
	}
	slog.Info("待摘要", "topics", len(topics), "min_runes", minRunes)
	done, err := b.each(topics, func(t topicRow) error {
		cctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		defer cancel()
		summary, err := ai.Summarize(cctx, b.client, b.model, t.Title, t.Content)
		if err != nil {
			return err
		}
		return b.db.Exec(`UPDATE topic SET ai_summary = ?, ai_summary_model = ?, ai_summary_at = now(), ai_summary_drift = 0
			WHERE id = ?`, summary, b.model, t.ID).Error
	})
	slog.Info("摘要完成", "done", done, "of", len(topics))
	return err
}

type exportLine struct {
	TopicID       int     `json:"topic_id"`
	Section       string  `json:"section,omitempty"`
	Reason        string  `json:"reason,omitempty"`
	Model         string  `json:"model,omitempty"`
	Summary       *string `json:"summary,omitempty"`
	SummaryModel  string  `json:"summary_model,omitempty"`
	ContentSHA256 string  `json:"content_sha256"`
}

func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func (b *backfill) export(out string) error {
	var rows []struct {
		ID           int
		Content      string
		Section      *string
		Reason       *string
		Model        *string
		AISummary    *string
		AISummaryMod string `gorm:"column:ai_summary_model"`
	}
	err := b.db.Raw(`SELECT t.id, t.content, c.section, c.reason, c.model, t.ai_summary, t.ai_summary_model
		FROM topic t LEFT JOIN topic_classification c ON c.topic_id = t.id AND c.source = 'backfill'
		WHERE c.topic_id IS NOT NULL OR t.ai_summary IS NOT NULL ORDER BY t.id`).Scan(&rows).Error
	if err != nil {
		return err
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	for _, r := range rows {
		line := exportLine{TopicID: r.ID, ContentSHA256: digest(r.Content), Summary: r.AISummary}
		if r.Section != nil {
			line.Section, line.Reason, line.Model = *r.Section, deref(r.Reason), deref(r.Model)
		}
		if r.AISummary != nil {
			line.SummaryModel = r.AISummaryMod
		}
		if err := enc.Encode(line); err != nil {
			return err
		}
	}
	slog.Info("已导出", "topics", len(rows), "out", out)
	return w.Flush()
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (b *backfill) importFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<24)
	var classified, summarized, skipped int
	for sc.Scan() {
		var line exportLine
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			return err
		}
		var cur struct {
			Content     string
			HasVerdict  bool
			HasSummary  bool
			TopicExists bool
		}
		err := b.db.Raw(`SELECT t.content, TRUE AS topic_exists,
			EXISTS (SELECT 1 FROM topic_classification c WHERE c.topic_id = t.id AND c.source <> 'fallback') AS has_verdict,
			t.ai_summary IS NOT NULL AS has_summary
			FROM topic t WHERE t.id = ?`, line.TopicID).Scan(&cur).Error
		if err != nil {
			return err
		}
		if !cur.TopicExists {
			skipped++
			continue
		}
		if line.Section != "" && !cur.HasVerdict {
			if err := b.apply(line.TopicID, line.Section, line.Reason, line.Model); err != nil {
				return fmt.Errorf("topic %d: %w", line.TopicID, err)
			}
			classified++
		}
		if line.Summary != nil && !cur.HasSummary && digest(cur.Content) == line.ContentSHA256 {
			if err := b.db.Exec(`UPDATE topic SET ai_summary = ?, ai_summary_model = ?, ai_summary_at = now(), ai_summary_drift = 0
				WHERE id = ?`, *line.Summary, line.SummaryModel, line.TopicID).Error; err != nil {
				return err
			}
			summarized++
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	slog.Info("已导入", "classified", classified, "summarized", summarized, "missing_topics", skipped)
	return nil
}
