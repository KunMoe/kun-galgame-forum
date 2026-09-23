package calendarapiv1

import (
	"kun-galgame-api/internal/galgame/workrepr"
)

type ReleaseCalendarMonth struct {
	Object        string                 `json:"object" enum:"release_calendar_month" maxLength:"22" doc:"Type discriminant. Always release_calendar_month."`
	CalendarMonth string                 `json:"calendar_month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$" maxLength:"7" doc:"The month window as YYYY-MM."`
	Today         string                 `json:"today_date" format:"date" maxLength:"10" doc:"Today in Asia/Tokyo, YYYY-MM-DD."`
	Items         []workrepr.WorkSummary `json:"items" doc:"Works released in this window, in catalog's order. Empty array, never null. A page may be shorter than catalog's total when a row cannot be rendered."`
	PrevMonth     *string                `json:"prev_month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$" maxLength:"7" doc:"The previous calendar month. null when none."`
	NextMonth     *string                `json:"next_month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$" maxLength:"7" doc:"The next calendar month. null when none."`
	HasPrev       bool                   `json:"has_prev" doc:"Whether catalog reports a previous month of releases."`
	HasNext       bool                   `json:"has_next" doc:"Whether catalog reports a later month of releases."`
	MinMonth      *string                `json:"min_month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$" maxLength:"7" doc:"The earliest month that has a release. null when catalog does not say."`
	MaxMonth      *string                `json:"max_month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$" maxLength:"7" doc:"The latest month that has a release. null when catalog does not say."`
	ItemCount     int                    `json:"item_count" minimum:"0" doc:"Works returned after walking the window. When is_truncated is true this may be less than catalog's total."`
	IsTruncated   bool                   `json:"is_truncated" doc:"Whether the walk stopped at the 2,000-work cap."`
}

type ReleaseCalendarToday struct {
	Object     string `json:"object" enum:"release_calendar_today" maxLength:"22" doc:"Type discriminant. Always release_calendar_today."`
	Today      string `json:"today_date" format:"date" maxLength:"10" doc:"Today in Asia/Tokyo, YYYY-MM-DD."`
	HasRelease bool   `json:"has_release" doc:"Whether a work in the walked current month has a day-precise release_date equal to today."`
	ExpiresIn  int    `json:"expires_in" minimum:"0" doc:"Whole seconds until the next Asia/Tokyo midnight."`
}

type ReleaseCalendarPending struct {
	Object      string                 `json:"object" enum:"release_calendar_pending" maxLength:"24" doc:"Type discriminant. Always release_calendar_pending."`
	Year        int                    `json:"year" minimum:"1" maximum:"9999" format:"int64" doc:"The year window."`
	Items       []workrepr.WorkSummary `json:"items" doc:"Works pending a day-precise date in this year. Empty array, never null."`
	ItemCount   int                    `json:"item_count" minimum:"0" doc:"Works returned after walking the window. When is_truncated is true this may be less than catalog's total."`
	IsTruncated bool                   `json:"is_truncated" doc:"Whether the walk stopped at the 2,000-work cap."`
}

type ReleaseCalendarTBA struct {
	Object      string                 `json:"object" enum:"release_calendar_tba" maxLength:"20" doc:"Type discriminant. Always release_calendar_tba."`
	Items       []workrepr.WorkSummary `json:"items" doc:"Works with an unknown release date. Empty array, never null."`
	ItemCount   int                    `json:"item_count" minimum:"0" doc:"Works returned after walking the window. When is_truncated is true this may be less than catalog's total."`
	IsTruncated bool                   `json:"is_truncated" doc:"Whether the walk stopped at the 2,000-work cap."`
}

type ReleaseCalendarUpcomingEntry struct {
	CalendarMonth string                 `json:"calendar_month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$" maxLength:"7" doc:"The month window as YYYY-MM."`
	Items         []workrepr.WorkSummary `json:"items" doc:"Works in this month that are still upcoming. Empty array, never null."`
	IsTruncated   bool                   `json:"is_truncated" doc:"Whether this month stopped at the 500-work cap."`
}

type ReleaseCalendarUpcoming struct {
	Object    string                         `json:"object" enum:"release_calendar_upcoming" maxLength:"25" doc:"Type discriminant. Always release_calendar_upcoming."`
	Today     string                         `json:"today_date" format:"date" maxLength:"10" doc:"Today in Asia/Tokyo, YYYY-MM-DD."`
	Entries   []ReleaseCalendarUpcomingEntry `json:"entries" doc:"Months from the current month that have upcoming works. Empty months are omitted. Empty array, never null."`
	ItemCount int                            `json:"item_count" minimum:"0" doc:"Works across every entry."`
}

type monthInput struct {
	Month       string `query:"month" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$" maxLength:"7" doc:"Calendar month as YYYY-MM. Omitted means the current month in Asia/Tokyo."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false."`
}

type todayInput struct {
	IncludeNSFW bool `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false."`
}

type pendingInput struct {
	Year        int  `query:"year" minimum:"0" maximum:"9999" doc:"Calendar year. 0 or omitted means the current year in Asia/Tokyo."`
	IncludeNSFW bool `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false."`
}

type tbaInput struct {
	IncludeNSFW bool `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false."`
}

type upcomingInput struct {
	IncludeNSFW bool `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false."`
}

type monthOutput struct {
	Body ReleaseCalendarMonth
}

type todayOutput struct {
	Body ReleaseCalendarToday
}

type pendingOutput struct {
	Body ReleaseCalendarPending
}

type tbaOutput struct {
	Body ReleaseCalendarTBA
}

type upcomingOutput struct {
	Body ReleaseCalendarUpcoming
}
