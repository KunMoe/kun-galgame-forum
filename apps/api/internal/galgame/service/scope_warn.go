package service

import (
	"log/slog"
	"sync/atomic"
	"time"
)

// A token that predates a newly shipped scope is the ordinary case for a while,
// and the two obvious ways to log it are both wrong. On 2026-09-08 the folder
// scopes shipped with the collections cutover: every read 403'd for the 95,286
// sessions minted before that day, and `isFavorited` logged one WARN per call —
// ~30k lines a day of noise nobody could read. `playtime.go` had already gone
// the other way and logged nothing at all, which is why the same outage ran for
// an hour with no alert and no user report: the marks simply went missing.
//
// So: one line a minute, carrying the count it swallowed. Silence means zero,
// and a nonzero count is a number somebody can act on.
type scopeWarn struct {
	last      atomic.Int64
	swallowed atomic.Int64
}

// One per call site, so a starved scope on the detail page cannot hide behind
// the session-level read having already logged this minute.
var (
	warnFoldersScope  scopeWarn
	warnFavoriteScope scopeWarn
	warnPlaytimeScope scopeWarn
)

func (w *scopeWarn) warn(msg string, args ...any) {
	now := time.Now().Unix()
	last := w.last.Load()
	if now-last < 60 || !w.last.CompareAndSwap(last, now) {
		w.swallowed.Add(1)
		return
	}
	slog.Warn(msg, append(args, "swallowed_since_last", w.swallowed.Swap(0))...)
}
