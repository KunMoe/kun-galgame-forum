package apiv1

import (
	"strconv"
	"strings"

	"kun-galgame-api/internal/activity/repository"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
)

type FollowingTopicTimes struct {
	BumpedAt repr.DateTime  `json:"bumped_at" doc:"The topic's bump time, as topic.bumped_at."`
	EditedAt *repr.DateTime `json:"edited_at" doc:"Time of the latest edit. null when never edited."`
}

func topicPublishID(it communityclient.ActivityItemView) (int, bool) {
	if it.Site != siteKungal {
		return 0, false
	}
	rest, ok := strings.CutPrefix(it.Key, KindOfFeed("TOPIC_CREATION")+":")
	if !ok {
		return 0, false
	}
	id, err := strconv.Atoi(rest)
	return id, err == nil && id > 0
}

func (s *Service) topicTimes(items []communityclient.ActivityItemView) (map[int]repository.TopicTimesRow, *problem.Problem) {
	ids := make([]int, 0, len(items))
	for _, it := range items {
		if id, ok := topicPublishID(it); ok {
			ids = append(ids, id)
		}
	}
	rows, err := s.repo.TopicTimes(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return rows, nil
}

func topicTimesOf(it communityclient.ActivityItemView, times map[int]repository.TopicTimesRow) *FollowingTopicTimes {
	id, ok := topicPublishID(it)
	if !ok {
		return nil
	}
	row, found := times[id]
	if !found {
		return nil
	}
	return &FollowingTopicTimes{BumpedAt: repr.Timestamp(row.StatusUpdateTime), EditedAt: repr.TimestampPtr(row.Edited)}
}
