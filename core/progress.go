package core

// Progress is always derived from sessions, never stored (SPEC §2, §4.4).

// WatchedEpisodes is the sum of session episodes, capped at the total when
// the total is given.
func (e *Entry) WatchedEpisodes() int {
	n := 0
	for _, s := range e.Sessions {
		n += s.Episodes
	}
	if t := e.Fields.TotalEpisodes; t > 0 && n > t {
		n = t
	}
	return n
}

// PagesRead is the furthest page reached in any session.
func (e *Entry) PagesRead() int {
	n := 0
	for _, s := range e.Sessions {
		if s.ToPage > n {
			n = s.ToPage
		}
	}
	return n
}

// Progress returns how far the entry is and its total. ok is false for types
// that don't track progress. total is 0 when not given.
func (e *Entry) Progress() (done, total int, ok bool) {
	switch e.Type.Progress() {
	case EpisodeProgress:
		return e.WatchedEpisodes(), e.Fields.TotalEpisodes, true
	case PageProgress:
		return e.PagesRead(), e.Fields.TotalPages, true
	}
	return 0, 0, false
}

// ProgressPercent is progress as a whole percentage, 0 when there's no total.
func (e *Entry) ProgressPercent() int {
	done, total, ok := e.Progress()
	if !ok || total <= 0 {
		return 0
	}
	p := (done*100 + total/2) / total
	if p > 100 {
		p = 100
	}
	return p
}

// ReachedEnd reports whether progress has reached a given total.
func (e *Entry) ReachedEnd() bool {
	done, total, ok := e.Progress()
	return ok && total > 0 && done >= total
}

// ShouldPromptFinished reports whether to offer "Mark as finished?": the
// entry has reached the end but is still ongoing. The same check drives the
// "✓ Finished?" badge on Still With You cards.
func (e *Entry) ShouldPromptFinished() bool {
	return e.Status == Ongoing && e.ReachedEnd()
}
