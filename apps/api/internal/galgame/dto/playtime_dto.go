package dto

type PlaytimeMineItem struct {
	Galgame GalgameListCard `json:"galgame"`
	Minutes int             `json:"minutes"`
	Status  string          `json:"status"`
	Clients int             `json:"clients"`
}

type PlaytimeMinePage struct {
	Items         []PlaytimeMineItem `json:"items"`
	Total         int                `json:"total"`
	TotalMinutes  int                `json:"total_minutes"`
	FinishedWorks int                `json:"finished_works"`
	// Set when the user reports on more works than one sweep of the upstream
	// sync face returns; the page is then the oldest-changed slice, not all of it.
	Truncated bool `json:"truncated"`
}
