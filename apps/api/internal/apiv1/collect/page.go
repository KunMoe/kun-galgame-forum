package collect

const (
	DefaultLimit = 20
)

type Page struct {
	Cursor string `query:"cursor" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque keyset cursor from a previous page of this collection."`
	Limit  int    `query:"limit" minimum:"1" maximum:"100" default:"20" doc:"Page size. 1–100, default 20. Values above 100 are rejected, not clamped."`
}

type Total struct {
	IncludeTotal bool `query:"include_total" doc:"When true, the response includes total counted under the same predicate as items."`
}
