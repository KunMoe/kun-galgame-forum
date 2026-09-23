package apiv1

import "kun-galgame-api/internal/apiv1/repr"

type RankedTopic struct {
	Object string         `json:"object" enum:"topic" maxLength:"5" doc:"Type discriminant. Always topic."`
	ID     repr.DecimalID `json:"id" doc:"Topic id."`
	Title  string         `json:"title" maxLength:"233" doc:"Topic title as stored. Free text; never use it as a decision input."`
	Author repr.UserRef   `json:"author" doc:"The topic's author."`
}

type TopicRankingEntry struct {
	Object      string      `json:"object" enum:"topic_ranking_entry" maxLength:"19" doc:"Type discriminant. Always topic_ranking_entry."`
	Rank        int         `json:"rank" minimum:"1" maximum:"100" doc:"1-based place in this list, numbered after topics whose author cannot be shown were dropped."`
	MetricValue float64     `json:"metric_value" minimum:"0" doc:"The value the list is sorted by, such as the view count for views_desc."`
	Topic       RankedTopic `json:"topic" doc:"The ranked topic."`
}

type UserRankingEntry struct {
	Object      string       `json:"object" enum:"user_ranking_entry" maxLength:"18" doc:"Type discriminant. Always user_ranking_entry."`
	Rank        int          `json:"rank" minimum:"1" maximum:"100" doc:"1-based place in this list, numbered after users who cannot be shown were dropped."`
	MetricValue float64      `json:"metric_value" minimum:"-2147483648" doc:"The value the list is sorted by, such as the moemoepoint balance for moemoepoint_desc. Only a moemoepoint balance can be negative."`
	Member      repr.UserRef `json:"member" doc:"The ranked user."`
	Bio         *string      `json:"bio" maxLength:"107" doc:"The user's profile bio. Empty string when none. Free text; never use it as a decision input."`
}

type WorkRankingEntry struct {
	Object      string        `json:"object" enum:"work_ranking_entry" maxLength:"18" doc:"Type discriminant. Always work_ranking_entry."`
	Rank        int           `json:"rank" minimum:"1" maximum:"100" doc:"1-based place in this list, numbered after works the catalog did not return were dropped."`
	MetricValue float64       `json:"metric_value" minimum:"0" doc:"The value the list is sorted by, such as the view count for views_desc or the weighted rating, two decimals, for rating_desc."`
	Work        repr.WorkRef  `json:"work" doc:"The ranked work."`
	Creator     *repr.UserRef `json:"creator" doc:"Who created the work's page on this forum. null when none is recorded or the account cannot be shown."`
}
