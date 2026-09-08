package repository

import (
	"log/slog"
	"math"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

const bayesianPriorC = 10.0

const ratingAggJoin = "LEFT JOIN (SELECT galgame_id, SUM(overall) AS rsum, " +
	"COUNT(*) AS rcnt FROM galgame_rating GROUP BY galgame_id) rt " +
	"ON rt.galgame_id = g.id"

type GalgameListRepository struct {
	db *gorm.DB
}

func NewGalgameListRepository(db *gorm.DB) *GalgameListRepository {
	return &GalgameListRepository{db: db}
}

var allProviders = []string{
	"baidu", "aliyun", "quark", "pan123", "tianyiyun",
	"caiyun", "xunlei", "uc", "lanzou", "other",
}

const viewOneDayExpr = "COALESCE((SELECT SUM(d.count) FROM galgame_view_daily d " +
	"WHERE d.entity_id = g.id AND d.day = CURRENT_DATE), 0)"

func listSortColumn(field string) string {
	switch field {
	case "created":
		return "g.created"
	case "view":
		return "g.view"
	case "view_1d":
		return viewOneDayExpr
	case "view_7d":
		return "g.view_7d"
	case "view_30d":
		return "g.view_30d"
	case "release_date":
		return "g.release_date"
	default:
		return "g.resource_update_time"
	}
}

func (r *GalgameListRepository) ListIDs(f model.GalgameListFilter) (ids []int, total int64) {
	sortCol := listSortColumn(f.SortField)
	isSubquerySort := f.SortField == "view_1d"

	ratingSort := f.SortField == "rating"
	ratingFilter := f.MinRatingCount > 0 || f.MinRating > 0

	var bayes string
	if ratingSort || ratingFilter {
		bayes = r.bayesianExpr()
	}

	order := sortDirection(f.SortOrder)
	orderClause := sortCol + " " + order
	switch {
	case ratingSort:
		orderClause = "(rt.rcnt IS NULL), " + bayes + " " + order
	case sortCol == "g.release_date":
		orderClause += " NULLS LAST"
	}
	// Ties are the rule here, not the exception: most of the table shares
	// view = 0, a busy release day shares a date, and every unmirrored row
	// shares NULL. An ORDER BY that stops at the tied column leaves the rest to
	// the planner, which is free to answer two pages differently — the same row
	// on both, another on neither.
	orderClause += ", g.id DESC"

	type idRow struct {
		ID int `gorm:"column:id"`
	}

	if !f.HasResourcePredicate() {
		build := func() *gorm.DB {
			q := applyContentLimit(applyPublished(r.db.Table("galgame g"), f), f)
			if f.RestrictIDs != nil {
				q = q.Where("g.id = ANY(?::int[])", intArrayLit(f.RestrictIDs))
			}
			if ratingSort || ratingFilter {
				q = q.Joins(ratingAggJoin)
			}
			q = applyReleaseFilter(q, f)
			q = applyGameTypeFilter(q, f)
			if ratingFilter {
				q = applyRatingFilter(q, f, bayes)
			}
			if !f.Indexed && !f.ShowNoResource {
				q = q.Where("EXISTS (SELECT 1 FROM galgame_resource gr WHERE gr.galgame_id = g.id)")
			}
			return q
		}
		build().Select("COUNT(*)").Scan(&total)
		var rows []idRow
		build().
			Select("g.id").
			Order(orderClause).
			Offset((f.Page - 1) * f.Limit).Limit(f.Limit).
			Scan(&rows)
		ids = make([]int, len(rows))
		for i, row := range rows {
			ids[i] = row.ID
		}
		return
	}

	inner := applyContentLimit(applyPublished(r.db.Table("galgame g").
		Select("DISTINCT g.id").
		Joins("JOIN galgame_resource gr ON gr.galgame_id = g.id"), f), f)
	if f.RestrictIDs != nil {
		inner = inner.Where("g.id = ANY(?::int[])", intArrayLit(f.RestrictIDs))
	}
	if ratingFilter {
		inner = inner.Joins(ratingAggJoin)
	}

	inner = applyReleaseFilter(inner, f)
	inner = applyGameTypeFilter(inner, f)
	if f.Type != "" && f.Type != "all" {
		inner = inner.Where("gr.type = ?", f.Type)
	}
	if f.Language != "" && f.Language != "all" {
		inner = inner.Where("gr.language = ?", f.Language)
	}
	if f.Platform != "" && f.Platform != "all" {
		inner = inner.Where("gr.platform = ?", f.Platform)
	}
	if len(f.IncludeProviders) > 0 {
		inner = inner.Where("gr.provider && ?", providerArrayLit(f.IncludeProviders))
	}
	if len(f.ExcludeOnlyProviders) > 0 {
		allowed := providersExcluding(f.ExcludeOnlyProviders)
		if len(allowed) > 0 {
			inner = inner.Where("gr.provider && ?", providerArrayLit(allowed))
		}
	}
	if ratingFilter {
		inner = applyRatingFilter(inner, f, bayes)
	}

	r.db.Table("(?) AS sub", inner).Select("COUNT(*)").Scan(&total)

	main := applyPublished(r.db.Table("galgame g").
		Select("g.id").
		Joins("JOIN galgame_resource gr ON gr.galgame_id = g.id"), f)
	groupBy := "g.id, " + sortCol
	if isSubquerySort {
		groupBy = "g.id"
	}
	if ratingSort {
		main = main.Joins(ratingAggJoin)
		groupBy = "g.id, rt.rsum, rt.rcnt"
	}

	var rows []idRow
	main.
		Where("gr.galgame_id IN (?)", inner).
		Group(groupBy).
		Order(orderClause).
		Offset((f.Page - 1) * f.Limit).Limit(f.Limit).
		Scan(&rows)

	ids = make([]int, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	return
}

// OrderRestrictIDs ranks one entity's catalog membership by a forum-side column.
// The members are catalog's, so a large share of them have no row in `galgame`
// at all and no value to rank on; those keep the order the walk delivered them
// in and follow everything that could be ranked. `g.id IS NULL` is what carries
// that, not NULLS LAST: view_1d COALESCEs to 0 and the bayesian score falls back
// to the prior mean, so a member the forum has never seen would otherwise tie
// with real rows instead of dropping out of the ranking.
func (r *GalgameListRepository) OrderRestrictIDs(ids []int, f model.GalgameListFilter) []int {
	if len(ids) < 2 {
		return ids
	}
	order := sortDirection(f.SortOrder)

	q := r.db.Table("unnest(?::int[]) WITH ORDINALITY AS m(id, ord)", intArrayLit(ids)).
		Joins("LEFT JOIN galgame g ON g.id = m.id")

	orderClause := listSortColumn(f.SortField) + " " + order + " NULLS LAST"
	if f.SortField == "rating" {
		q = q.Joins(ratingAggJoin)
		orderClause = "(rt.rcnt IS NULL), " + r.bayesianExpr() + " " + order
	}

	type idRow struct {
		ID int `gorm:"column:id"`
	}
	var rows []idRow
	err := q.Select("m.id").
		Order("(g.id IS NULL), " + orderClause + ", m.ord").
		Scan(&rows).Error
	if err != nil || len(rows) != len(ids) {
		// Catalog's order is the correct page either way, so the fallback costs
		// the ranking and nothing else — but it is invisible from the page, and
		// a short count would quietly drop members off the end of the list.
		slog.Error("order catalog members: 排序失败, 回退到目录顺序",
			"sort_field", f.SortField, "members", len(ids), "ranked", len(rows), "error", err)
		return ids
	}
	out := make([]int, len(rows))
	for i, row := range rows {
		out[i] = row.ID
	}
	return out
}

func (r *GalgameListRepository) bayesianExpr() string {
	var m float64
	r.db.Table("galgame_rating").Select("COALESCE(AVG(overall), 0)").Scan(&m)
	c := strconv.FormatFloat(bayesianPriorC, 'f', -1, 64)
	ms := strconv.FormatFloat(m, 'f', 6, 64)
	return "(" + c + " * " + ms + " + COALESCE(rt.rsum, 0)) / (" +
		c + " + COALESCE(rt.rcnt, 0))"
}

type RatingInfo struct {
	Score float64
	Count int
}

func (r *GalgameListRepository) BayesianRatings(ids []int) map[int]RatingInfo {
	out := make(map[int]RatingInfo, len(ids))
	if len(ids) == 0 {
		return out
	}
	var m float64
	r.db.Table("galgame_rating").Select("COALESCE(AVG(overall), 0)").Scan(&m)

	type aggRow struct {
		GalgameID int     `gorm:"column:galgame_id"`
		Rsum      float64 `gorm:"column:rsum"`
		Rcnt      int     `gorm:"column:rcnt"`
	}
	var rows []aggRow
	r.db.Table("galgame_rating").
		Select("galgame_id, SUM(overall) AS rsum, COUNT(*) AS rcnt").
		Where("galgame_id IN ?", ids).
		Group("galgame_id").
		Scan(&rows)

	for _, row := range rows {
		if row.Rcnt == 0 {
			continue
		}
		score := (bayesianPriorC*m + row.Rsum) / (bayesianPriorC + float64(row.Rcnt))
		out[row.GalgameID] = RatingInfo{
			Score: math.Round(score*10) / 10,
			Count: row.Rcnt,
		}
	}
	return out
}

func applyRatingFilter(q *gorm.DB, f model.GalgameListFilter, bayes string) *gorm.DB {
	if f.MinRatingCount > 0 {
		q = q.Where("COALESCE(rt.rcnt, 0) >= ?", f.MinRatingCount)
	}
	if f.MinRating > 0 {
		q = q.Where(bayes+" >= ?", f.MinRating)
	}
	return q
}

func applyReleaseFilter(q *gorm.DB, f model.GalgameListFilter) *gorm.DB {
	if f.ReleasedFrom != "" {
		q = q.Where("g.release_date >= ?", f.ReleasedFrom)
	}
	if f.ReleasedTo != "" {
		q = q.Where("g.release_date <= ?", f.ReleasedTo)
	}
	if len(f.ReleasedMonths) > 0 {
		q = q.Where("EXTRACT(MONTH FROM g.release_date)::int IN ?", f.ReleasedMonths)
	}
	return q
}

func applyGameTypeFilter(q *gorm.DB, f model.GalgameListFilter) *gorm.DB {
	switch f.GameType {
	case "", "all":
		return q
	case "uncategorized":
		return q.Where("NOT EXISTS (SELECT 1 FROM galgame_rating grt WHERE grt.galgame_id = g.id AND grt.galgame_type <> '[]'::jsonb)")
	default:
		return q.Where(
			"EXISTS (SELECT 1 FROM galgame_rating grt WHERE grt.galgame_id = g.id AND grt.galgame_type @> ?)",
			"[\""+f.GameType+"\"]",
		)
	}
}

// `published` is sticky since 078, so on a list that already requires a resource
// it filters exactly one thing: the rows a hide/ban unpublished. Dropping it put
// banned Galgame back on /galgame — since 070 a ban stops deleting resources.
func applyPublished(q *gorm.DB, f model.GalgameListFilter) *gorm.DB {
	if f.ShowNoResource && !f.Indexed {
		return q
	}
	return q.Where("g.published")
}

// The gate that actually hides a work is catalog's, applied when the page is
// hydrated — this only decides which ids get that far. NULL is "the sync has
// not seen this row yet" and must pass, or a fresh column empties every list.
func applyContentLimit(q *gorm.DB, f model.GalgameListFilter) *gorm.DB {
	if !f.SFWOnly {
		return q
	}
	return q.Where("g.content_limit IS NULL OR g.content_limit = 'sfw'")
}

func providerArrayLit(providers []string) string {
	return "{" + strings.Join(providers, ",") + "}"
}

// The entity pages build their filter straight from the raw query string, so
// this is the only thing standing between ?sortOrder= and the ORDER BY.
func sortDirection(order string) string {
	if strings.EqualFold(order, "asc") {
		return "ASC"
	}
	return "DESC"
}

func intArrayLit(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func providersExcluding(excluded []string) []string {
	exSet := map[string]bool{}
	for _, e := range excluded {
		exSet[e] = true
	}
	out := make([]string, 0, len(allProviders))
	for _, p := range allProviders {
		if !exSet[p] {
			out = append(out, p)
		}
	}
	return out
}
