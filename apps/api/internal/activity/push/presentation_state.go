package push

import (
	"context"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/userclient"
)

type resRow struct {
	ID     int `gorm:"column:id"`
	WorkID int `gorm:"column:work_id"`
	UserID int `gorm:"column:user_id"`
}

type ratingRow struct {
	ID     int `gorm:"column:id"`
	WorkID int `gorm:"column:work_id"`
	UserID int `gorm:"column:user_id"`
}

type quizRow struct {
	ID          int64  `gorm:"column:id"`
	UserID      int    `gorm:"column:user_id"`
	Question    string `gorm:"column:question"`
	HideGalgame bool   `gorm:"column:hide_galgame"`
}

type toolRow struct {
	ID     int    `gorm:"column:id"`
	Name   string `gorm:"column:name"`
	Status int    `gorm:"column:status"`
	UserID int    `gorm:"column:user_id"`
}

type siteRow struct {
	ID       int    `gorm:"column:id"`
	Name     string `gorm:"column:name"`
	URL      string `gorm:"column:url"`
	AgeLimit string `gorm:"column:age_limit"`
}

type presentationState struct {
	published map[int]bool
	resources map[int]resRow
	ratings   map[int]ratingRow
	quizzes   map[int64]quizRow
	quizWorks map[int64][]int
	toolsets  map[int]toolRow
	websites  map[int]siteRow
	users     map[int]userclient.User
	catalog   map[int]client.CatalogWorkListItem
}

func (p *Presenter) loadPresentationState(ctx context.Context, keys []presentationKey) (*presentationState, error) {
	st := &presentationState{
		published: map[int]bool{},
		resources: map[int]resRow{},
		ratings:   map[int]ratingRow{},
		quizzes:   map[int64]quizRow{},
		quizWorks: map[int64][]int{},
		toolsets:  map[int]toolRow{},
		websites:  map[int]siteRow{},
		users:     map[int]userclient.User{},
		catalog:   map[int]client.CatalogWorkListItem{},
	}
	var workIDs, resIDs, ratingIDs, toolIDs, siteIDs, userIDs []int
	var quizIDs []int64
	for _, k := range keys {
		switch {
		case k.Kind == anchorKindGame:
			if id, ok := parseStrictInt(k.ID); ok {
				workIDs = append(workIDs, id)
			}
		case k.Kind == anchorKindResource && strings.HasPrefix(k.ID, prefixResource):
			if id, ok := parseStrictInt(strings.TrimPrefix(k.ID, prefixResource)); ok {
				resIDs = append(resIDs, id)
			}
		case k.Kind == anchorKindResource && strings.HasPrefix(k.ID, prefixRating):
			if id, ok := parseStrictInt(strings.TrimPrefix(k.ID, prefixRating)); ok {
				ratingIDs = append(ratingIDs, id)
			}
		case k.Kind == anchorKindResource && strings.HasPrefix(k.ID, prefixQuiz):
			if id, ok := parseStrictInt64(strings.TrimPrefix(k.ID, prefixQuiz)); ok {
				quizIDs = append(quizIDs, id)
			}
		case k.Kind == anchorKindResource && strings.HasPrefix(k.ID, prefixToolset):
			if id, ok := parseStrictInt(strings.TrimPrefix(k.ID, prefixToolset)); ok {
				toolIDs = append(toolIDs, id)
			}
		case k.Kind == anchorKindResource && strings.HasPrefix(k.ID, prefixWebsite):
			if id, ok := parseStrictInt(strings.TrimPrefix(k.ID, prefixWebsite)); ok {
				siteIDs = append(siteIDs, id)
			}
		}
	}
	var resources []resRow
	if err := scanIntMap(p.db, `SELECT id, work_id, user_id FROM galgame_resource WHERE id IN ?`, resIDs, &resources); err != nil {
		return nil, err
	}
	for _, r := range resources {
		st.resources[r.ID] = r
		workIDs = append(workIDs, r.WorkID)
		userIDs = append(userIDs, r.UserID)
	}
	var ratings []ratingRow
	if err := scanIntMap(p.db, `SELECT id, work_id, user_id FROM galgame_rating WHERE id IN ?`, ratingIDs, &ratings); err != nil {
		return nil, err
	}
	for _, r := range ratings {
		st.ratings[r.ID] = r
		workIDs = append(workIDs, r.WorkID)
		userIDs = append(userIDs, r.UserID)
	}
	var quizzes []quizRow
	if err := scanIntMap(p.db, `SELECT id, user_id, question, hide_galgame FROM galgame_quiz WHERE id IN ?`, quizIDs, &quizzes); err != nil {
		return nil, err
	}
	for _, r := range quizzes {
		st.quizzes[r.ID] = r
		userIDs = append(userIDs, r.UserID)
	}
	var links []struct {
		QuizID int64 `gorm:"column:quiz_id"`
		WorkID int   `gorm:"column:work_id"`
	}
	if err := scanIntMap(p.db, `SELECT quiz_id, work_id FROM galgame_quiz_galgame WHERE quiz_id IN ?`, quizIDs, &links); err != nil {
		return nil, err
	}
	for _, r := range links {
		st.quizWorks[r.QuizID] = append(st.quizWorks[r.QuizID], r.WorkID)
		workIDs = append(workIDs, r.WorkID)
	}
	var tools []toolRow
	if err := scanIntMap(p.db, `SELECT id, name, status, user_id FROM galgame_toolset WHERE id IN ?`, toolIDs, &tools); err != nil {
		return nil, err
	}
	for _, r := range tools {
		st.toolsets[r.ID] = r
		userIDs = append(userIDs, r.UserID)
	}
	var sites []siteRow
	if err := scanIntMap(p.db, `SELECT id, name, url, age_limit FROM galgame_website WHERE id IN ?`, siteIDs, &sites); err != nil {
		return nil, err
	}
	for _, r := range sites {
		st.websites[r.ID] = r
	}
	var pubs []struct {
		ID        int  `gorm:"column:id"`
		Published bool `gorm:"column:published"`
	}
	if err := scanIntMap(p.db, `SELECT id, published FROM galgame WHERE id IN ?`, uniqInts(workIDs), &pubs); err != nil {
		return nil, err
	}
	for _, r := range pubs {
		st.published[r.ID] = r.Published
	}
	catIDs := uniqInts(workIDs)
	if len(catIDs) > 0 {
		rows, appErr := p.catalog.CatalogRowsByWorkIDs(ctx, catIDs, "names,covers", "all")
		if appErr != nil {
			return nil, appErr
		}
		st.catalog = rows
	}
	uids := uniqInts(userIDs)
	if len(uids) > 0 {
		users, err := p.users.Users(ctx, uids)
		if err != nil {
			return nil, err
		}
		st.users = users
	}
	return st, nil
}

func parseStrictInt(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	return n, err == nil && n > 0 && strconv.Itoa(n) == s
}

func parseStrictInt64(s string) (int64, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil && n > 0 && strconv.FormatInt(n, 10) == s
}

func uniqInts(ids []int) []int {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(ids))
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
