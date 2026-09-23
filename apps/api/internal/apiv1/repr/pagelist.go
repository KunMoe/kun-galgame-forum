package repr

import "encoding/json"

type PageList[T any] struct {
	Object        string `json:"object" enum:"list" maxLength:"4" doc:"Type discriminant. Always list."`
	Items         []T    `json:"items" doc:"Members of this page. Empty array, never null."`
	Total         int    `json:"total" minimum:"0" doc:"Members matching the filters, under the same predicate as items. Counted up to the depth limit when total_relation is gte."`
	TotalRelation string `json:"total_relation" enum:"eq,gte" maxLength:"3" doc:"eq when total is exact, gte when it stopped at the depth limit and there are at least that many."`
}

func NewPageList[T any](items []T, total int, relation string) PageList[T] {
	if items == nil {
		items = []T{}
	}
	return PageList[T]{Object: "list", Items: items, Total: total, TotalRelation: relation}
}

func (l PageList[T]) MarshalJSON() ([]byte, error) {
	items := l.Items
	if items == nil {
		items = []T{}
	}
	type wire struct {
		Object        string `json:"object"`
		Items         []T    `json:"items"`
		Total         int    `json:"total"`
		TotalRelation string `json:"total_relation"`
	}
	return json.Marshal(wire{"list", items, l.Total, l.TotalRelation})
}
