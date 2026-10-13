package playstate

const (
	Wish         = "wish"
	Doing        = "doing"
	DoneOneRoute = "done_one_route"
	DoneMain     = "done_main"
	DoneAll      = "done_all"
	OnHold       = "on_hold"
	Dropped      = "dropped"

	// Done is finished with how much left unstated, which another application
	// can record. No picker offers it; a form that read it sends it back as it
	// found it. Mapped to one route, a save that only added a rating would
	// rewrite it on every NextMoe site.
	Done = "done"
)

// All is the picker order, and the order the frontend renders.
func All() []string {
	return []string{Wish, Doing, DoneOneRoute, DoneMain, DoneAll, OnHold, Dropped}
}

// ToCatalog maps a flat value onto catalog's two axes. completion is nil for
// every state but a done with a stated completion.
func ToCatalog(s string) (state string, completion *string, ok bool) {
	switch s {
	case Done:
		return "done", nil, true
	case Wish:
		return "wish", nil, true
	case Doing:
		return "doing", nil, true
	case OnHold:
		return "on_hold", nil, true
	case Dropped:
		return "dropped", nil, true
	case DoneOneRoute:
		c := "one_route"
		return "done", &c, true
	case DoneMain:
		c := "main"
		return "done", &c, true
	case DoneAll:
		c := "all"
		return "done", &c, true
	default:
		return "", nil, false
	}
}

// FromCatalog maps catalog's two axes back to a flat value, and returns Done
// when catalog reports done without a completion. It returns "" for an empty
// or unknown state.
func FromCatalog(state string, completion *string) string {
	switch state {
	case "wish":
		return Wish
	case "doing":
		return Doing
	case "on_hold":
		return OnHold
	case "dropped":
		return Dropped
	case "done":
		if completion == nil || *completion == "" {
			return Done
		}
		switch *completion {
		case "one_route":
			return DoneOneRoute
		case "main":
			return DoneMain
		case "all":
			return DoneAll
		default:
			return Done
		}
	default:
		return ""
	}
}
