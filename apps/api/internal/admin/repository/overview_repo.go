package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type OverviewRepository struct {
	db *gorm.DB
}

func NewOverviewRepository(db *gorm.DB) *OverviewRepository {
	return &OverviewRepository{db: db}
}

type OverviewCounts struct {
	Topic           int64
	Reply           int64
	TopicComment    int64
	Work            int64
	GalgameResource int64
	GalgameComment  int64
	Website         int64
	WebsiteComment  int64
	DirectMessage   int64
}

type overviewMetric struct {
	key   string
	table string
	where string
	field func(*OverviewCounts) *int64
}

var overviewMetrics = []overviewMetric{
	{"topic", "topic", "", func(c *OverviewCounts) *int64 { return &c.Topic }},
	{"reply", "topic_reply", "", func(c *OverviewCounts) *int64 { return &c.Reply }},
	{"topic_comment", "topic_comment", "", func(c *OverviewCounts) *int64 { return &c.TopicComment }},
	{"work", "galgame", "published", func(c *OverviewCounts) *int64 { return &c.Work }},
	{"galgame_resource", "galgame_resource", "", func(c *OverviewCounts) *int64 { return &c.GalgameResource }},
	{"galgame_comment", "feed_activity", "type = 'GALGAME_COMMENT_CREATION'", func(c *OverviewCounts) *int64 { return &c.GalgameComment }},
	{"website", "galgame_website", "", func(c *OverviewCounts) *int64 { return &c.Website }},
	{"website_comment", "feed_activity", "type = 'GALGAME_WEBSITE_COMMENT_CREATION'", func(c *OverviewCounts) *int64 { return &c.WebsiteComment }},
	{"direct_message", "chat_message", "", func(c *OverviewCounts) *int64 { return &c.DirectMessage }},
}

func metricFilter(m overviewMetric, extra string) string {
	conds := make([]string, 0, 2)
	if m.where != "" {
		conds = append(conds, m.where)
	}
	if extra != "" {
		conds = append(conds, extra)
	}
	if len(conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(conds, " AND ")
}

type metricCount struct {
	Metric string `gorm:"column:metric"`
	Day    string `gorm:"column:day"`
	Count  int64  `gorm:"column:count"`
}

func (r *OverviewRepository) Totals(ctx context.Context) (OverviewCounts, error) {
	parts := make([]string, len(overviewMetrics))
	for i, m := range overviewMetrics {
		parts[i] = fmt.Sprintf("SELECT '%s' AS metric, '' AS day, COUNT(*) AS count FROM %s%s", m.key, m.table, metricFilter(m, ""))
	}
	var rows []metricCount
	if err := r.db.WithContext(ctx).Raw(strings.Join(parts, " UNION ALL ")).Scan(&rows).Error; err != nil {
		return OverviewCounts{}, err
	}
	var out OverviewCounts
	for _, row := range rows {
		for _, m := range overviewMetrics {
			if m.key == row.Metric {
				*m.field(&out) = row.Count
			}
		}
	}
	return out, nil
}

// Daily buckets rows by their calendar day in zone, written into the SQL so the
// answer does not depend on the connection's session time zone.
func (r *OverviewRepository) Daily(ctx context.Context, zone string, from, to time.Time) (map[string]OverviewCounts, error) {
	parts := make([]string, len(overviewMetrics))
	args := make([]any, 0, 2*len(overviewMetrics))
	for i, m := range overviewMetrics {
		parts[i] = fmt.Sprintf(
			"SELECT '%s' AS metric, (created AT TIME ZONE '%s')::date::text AS day, COUNT(*) AS count FROM %s%s GROUP BY 2",
			m.key, zone, m.table, metricFilter(m, "created >= ? AND created < ?"))
		args = append(args, from, to)
	}
	var rows []metricCount
	if err := r.db.WithContext(ctx).Raw(strings.Join(parts, " UNION ALL "), args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]OverviewCounts)
	for _, row := range rows {
		c := out[row.Day]
		for _, m := range overviewMetrics {
			if m.key == row.Metric {
				*m.field(&c) = row.Count
			}
		}
		out[row.Day] = c
	}
	return out, nil
}
