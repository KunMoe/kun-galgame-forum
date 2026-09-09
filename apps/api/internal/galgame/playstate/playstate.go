package playstate

const (
	Wish         = "wish"
	Doing        = "doing"
	DoneOneRoute = "done_one_route"
	DoneMain     = "done_main"
	DoneAll      = "done_all"
	OnHold       = "on_hold"
	Dropped      = "dropped"
)

// All is the picker order, and the order the frontend renders.
func All() []string {
	return []string{Wish, Doing, DoneOneRoute, DoneMain, DoneAll, OnHold, Dropped}
}

// Valid reports whether s is one of the seven values a client may send.
// The read-only "done" is deliberately not valid input.
func Valid(s string) bool {
	switch s {
	case Wish, Doing, DoneOneRoute, DoneMain, DoneAll, OnHold, Dropped:
		return true
	default:
		return false
	}
}

// ToCatalog maps a flat value onto catalog's two axes. completion is nil for
// every state but done. ok is false for anything Valid rejects.
func ToCatalog(s string) (state string, completion *string, ok bool) {
	switch s {
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

// FromCatalog maps catalog's two axes back to a flat value, and returns
// "done" — which is display-only and never valid input — when catalog reports
// done without a completion. It returns "" for an empty or unknown state.
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
			// catalog answers state=done with no completion when another
			// application reported a completion it does not carry.
			return "done"
		}
		switch *completion {
		case "one_route":
			return DoneOneRoute
		case "main":
			return DoneMain
		case "all":
			return DoneAll
		default:
			return "done"
		}
	default:
		return ""
	}
}
