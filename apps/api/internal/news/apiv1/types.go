package apiv1

import (
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
)

type NewsItem struct {
	Object      string         `json:"object" enum:"news_item" maxLength:"9" doc:"Type discriminant. Always news_item."`
	ID          repr.DecimalID `json:"id" doc:"News item id. JSON string of a decimal integer."`
	NewsSource  string         `json:"news_source" pattern:"^[a-z0-9_]{1,40}$" maxLength:"40" doc:"Key of the partner that published the item. Name, homepage and attribution come from listNewsSources; an item shown on its own must still carry its partner's attribution."`
	Lane        string         `json:"lane" enum:"news,column" maxLength:"6" doc:"news for bulletins, column for longer pieces."`
	Title       string         `json:"title" maxLength:"500" doc:"Headline. Free text; never use it as a decision input."`
	Preview     string         `json:"preview" maxLength:"2000" doc:"The partner's own excerpt. There is no body: source_url is the only way to the full text. Free text; never use it as a decision input."`
	SourceURL   string         `json:"source_url" format:"uri" maxLength:"2048" doc:"The item on the partner's site."`
	PublishedAt repr.DateTime  `json:"published_at" doc:"When the partner published it."`
}

type NewsSource struct {
	Object       string        `json:"object" enum:"news_source" maxLength:"11" doc:"Type discriminant. Always news_source."`
	Key          string        `json:"key" pattern:"^[a-z0-9_]{1,40}$" maxLength:"40" doc:"The partner key that news items carry in news_source."`
	DisplayName  string        `json:"display_name" maxLength:"100" doc:"Partner name. Free text; never use it as a decision input."`
	HomepageURL  string        `json:"homepage_url" pattern:"^(https?://.*)?$" maxLength:"2048" doc:"The partner's homepage. Empty string if none."`
	ColumnURL    string        `json:"column_url" pattern:"^(https?://.*)?$" maxLength:"2048" doc:"The partner's column index. Empty string if none."`
	Attribution  string        `json:"attribution" maxLength:"2000" doc:"The attribution the partner requires next to its items. Free text; never use it as a decision input."`
	ForumAccount *repr.UserRef `json:"forum_account" doc:"The forum account the partner publishes under. null when there is none, it is not shown, or the account service could not be reached: it is decoration, and the directory does not fail over it."`
}

type YearCount struct {
	Year  int   `json:"year" minimum:"1970" maximum:"9999" doc:"Calendar year, Asia/Shanghai."`
	Count int64 `json:"count" minimum:"0" doc:"Items published that year."`
}

type MonthCount struct {
	Month int   `json:"month" minimum:"1" maximum:"12" doc:"Calendar month, Asia/Shanghai."`
	Count int64 `json:"count" minimum:"0" doc:"Items published that month."`
}

type DayCount struct {
	Day   int   `json:"day" minimum:"1" maximum:"31" doc:"Day of the month, Asia/Shanghai."`
	Count int64 `json:"count" minimum:"0" doc:"Items published that day."`
}

type NewsArchive struct {
	Object string       `json:"object" enum:"news_archive" maxLength:"12" doc:"Type discriminant. Always news_archive."`
	Years  []YearCount  `json:"years" maxItems:"100" doc:"Years that have items, newest first."`
	Months []MonthCount `json:"months" maxItems:"12" doc:"Months of the requested year that have items. Empty unless year was sent and is one of years."`
}

type NewsMonth struct {
	Object    string     `json:"object" enum:"news_month" maxLength:"10" doc:"Type discriminant. Always news_month."`
	Year      int        `json:"year" minimum:"1970" maximum:"9999" doc:"Calendar year, Asia/Shanghai."`
	Month     int        `json:"month" minimum:"1" maximum:"12" doc:"Calendar month, Asia/Shanghai."`
	ItemCount int        `json:"item_count" minimum:"0" doc:"Items in the whole month under the filters."`
	Days      []DayCount `json:"days" maxItems:"31" doc:"Every day of the month, empty days included."`
}

type NewsFilter struct {
	Lane       string `query:"lane" enum:"news,column" maxLength:"6" doc:"Only this lane. Omitted means both."`
	NewsSource string `query:"news_source" pattern:"^[a-z0-9_]{1,40}$" maxLength:"40" doc:"Only this partner. Omitted means every partner."`
}

type listNewsItemsInput struct {
	Cursor       string `query:"cursor" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque cursor from a previous page's next_cursor. It is bound to every filter and to limit."`
	Limit        int    `query:"limit" minimum:"1" maximum:"50" default:"20" doc:"Page size. 1–50, default 20. The news service pages at most 50; values above 50 are rejected, not clamped."`
	IncludeTotal bool   `query:"include_total" default:"false" doc:"When true, total counts every item under the same filters."`
	NewsFilter
	Year  int `query:"year" minimum:"0" maximum:"9999" doc:"Only items published in this year, Asia/Shanghai. 0 or omitted means any year."`
	Month int `query:"month" minimum:"0" maximum:"12" doc:"Only items published in this month of year. Needs year."`
}

type listNewsItemsOutput struct {
	Body repr.CountedList[NewsItem]
}

type listNewsSourcesOutput struct {
	Body repr.List[NewsSource]
}

type getNewsArchiveInput struct {
	NewsFilter
	Year int `query:"year" minimum:"0" maximum:"9999" doc:"Also break this year down by month."`
}

type getNewsArchiveOutput struct {
	Body NewsArchive
}

type NewsMonthPath struct {
	Year  int `path:"year" minimum:"1970" maximum:"9999" doc:"Calendar year, Asia/Shanghai."`
	Month int `path:"month" minimum:"1" maximum:"12" doc:"Calendar month, Asia/Shanghai."`
}

type getNewsMonthInput struct {
	NewsMonthPath
	NewsFilter
}

type getNewsMonthOutput struct {
	Body NewsMonth
}

type listNewsMonthItemsInput struct {
	NewsMonthPath
	collect.PageNumber
	NewsFilter
	Day int `query:"day" minimum:"0" maximum:"31" doc:"Only this day of the month. 0 or omitted means the whole month."`
}

type listNewsMonthItemsOutput struct {
	Body repr.PageList[NewsItem]
}
