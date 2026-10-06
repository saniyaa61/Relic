package ui

// Logging on the entry page (prototype openLogSheet, the *LogForm
// functions, saveRewatch, deleteSession, maybePromptFinished).

import (
	"strconv"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

type logSheet struct {
	entryID string
	editID  string // session or rewatch being edited; "" for a new one
	rewatch bool

	eps, from, to, mins Input
	note                Input
	date, start, end    dateField
	full                bool
	fullBtns            [2]widget.Clickable
	partial             Input
	rating              float64
	stars               starInput
	save                widget.Clickable
}

// openLogSheet opens the sheet to log (editID "") or edit a session, or a
// rewatch when rewatch is set.
func openLogSheet(a *App, entryID string, rewatch bool, editID string) *logSheet {
	e := a.Lib.Entry(entryID)
	if e == nil {
		return nil
	}
	l := &logSheet{entryID: entryID, editID: editID, rewatch: rewatch, full: true}
	for _, in := range []*Input{&l.eps, &l.from, &l.to, &l.mins, &l.partial} {
		in.Editor.SingleLine = true
		in.Editor.Filter = "0123456789"
		in.Editor.InputHint = keyHintNumeric
	}
	setNum := func(in *Input, n int) { in.Editor.SetText(strconv.Itoa(n)) }
	l.date.value = core.DateOf(a.Now(), a.Loc)
	if editID != "" {
		if rewatch {
			for _, r := range e.Rewatches {
				if r.ID == editID {
					l.date.value = core.DateOf(r.At, a.Loc)
					l.start.value, l.end.value = r.StartDate, r.EndDate
					l.full, l.rating = r.Full, r.Rating
					if !r.Full {
						setNum(&l.partial, r.Episodes)
					}
					l.note.Editor.SetText(r.Note)
				}
			}
		} else {
			for _, s := range e.Sessions {
				if s.ID != editID {
					continue
				}
				switch e.Type.Progress() {
				case core.EpisodeProgress:
					setNum(&l.eps, s.Episodes)
				case core.PageProgress:
					setNum(&l.from, s.FromPage)
					setNum(&l.to, s.ToPage)
				}
				if s.Minutes != nil {
					setNum(&l.mins, *s.Minutes)
				}
				l.note.Editor.SetText(s.Note)
			}
		}
	}
	title := "Log a session — " + e.Title
	switch {
	case editID != "":
		title = "Edit — " + e.Title
	case rewatch:
		title = "Log a " + strings.ToLower(e.Type.RewatchWord()) + " — " + e.Title
	}
	a.ShowSheet(&Sheet{Title: title, Body: l.Layout})
	return l
}

func (l *logSheet) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	e := a.Lib.Entry(l.entryID)
	if e == nil {
		a.CloseSheet()
		return layout.Dimensions{}
	}
	if l.save.Clicked(gtx) {
		l.submit(a, e)
	}
	for i, v := range []bool{true, false} {
		if l.fullBtns[i].Clicked(gtx) {
			l.full = v
		}
	}
	num := func(in *Input, label, placeholder string) layout.Widget {
		return labelled(th, label, func(gtx layout.Context) layout.Dimensions {
			return in.Layout(gtx, th, withBg(formInput(placeholder), th.Bg))
		})
	}
	thoughts := func(label, placeholder string, minH float32) layout.Widget {
		return labelled(th, label, func(gtx layout.Context) layout.Dimensions {
			return l.note.Layout(gtx, th, InputStyle{Bg: th.Bg, Radius: 9, Size: 13,
				Pad:  layout.Inset{Top: 9, Bottom: 9, Left: 11, Right: 11},
				Font: font.Font{Typeface: Serif, Style: font.Italic}, Multiline: true, MinHeight: unitDp(minH), LineHeight: 1.6,
				Placeholder: placeholder, PlaceholderAlpha: 0.7})
		})
	}
	saveLabel := "Save this session →"
	if l.rewatch {
		saveLabel = "Preserve this rewatch →"
	}
	if l.editID != "" {
		saveLabel = "Save changes →"
	}
	var rows []layout.Widget
	if l.rewatch {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.65, Color: th.Muted}.Layout(gtx, th,
					"You've been here before. Write about who you are now — and what this story means to you this time around.")
			})
		}, labelled(th, "Date", func(gtx layout.Context) layout.Dimensions { return l.date.Layout(gtx, a, "Date") }))
		if e.Type == core.Book {
			rows = append(rows, func(gtx layout.Context) layout.Dimensions {
				return pairRow(gtx,
					labelled(th, "Started", func(gtx layout.Context) layout.Dimensions { return l.start.Layout(gtx, a, "Started") }),
					labelled(th, "Finished", func(gtx layout.Context) layout.Dimensions { return l.end.Layout(gtx, a, "Finished") }), 1, 1)
			})
		}
		if e.Type.Progress() == core.EpisodeProgress {
			rows = append(rows, labelled(th, "Rewatched in full?", func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return toggleButton(gtx, th, &l.fullBtns[0], "Yes, all of it", l.full, 7)
					}),
					layout.Rigid(layout.Spacer{Width: 5}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return toggleButton(gtx, th, &l.fullBtns[1], "Partial rewatch", !l.full, 7)
					}))
			}))
			if !l.full {
				// Owner's addition: a partial rewatch counts the episodes
				// rewatched (SPEC §2, §4.5).
				rows = append(rows, num(&l.partial, "Episodes rewatched", "e.g. 4"))
			}
		}
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Baseline}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return fieldLabel(gtx, th, "New rating") }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Left: 3, Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return Text{Font: font.Font{Typeface: Sans}, Size: 10, Color: th.Muted}.Layout(gtx, th, "(optional)")
								})
							}))
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l.rating = l.stars.Layout(gtx, th, l.rating)
						return layout.Dimensions{Size: gtxSize(gtx, gtx.Sp(24)+gtx.Dp(2))}
					}))
			})
		}, thoughts("Your thoughts this time",
			"How has this changed for you? What do you see now that you didn't before? What stayed exactly the same?", 120))
	} else {
		switch e.Type.Progress() {
		case core.EpisodeProgress:
			word := "Episodes watched this session"
			if e.Type == core.Podcast {
				word = "Episodes listened this session"
			}
			rows = append(rows, num(&l.eps, word, "e.g. 2"),
				thoughts("Your thoughts right now", "Something moved you. Something stayed with you. Write it here.", 88))
		case core.PageProgress:
			toPH := "end"
			if e.Fields.TotalPages > 0 {
				toPH = strconv.Itoa(e.Fields.TotalPages)
			}
			rows = append(rows, func(gtx layout.Context) layout.Dimensions {
				return pairRow(gtx, num(&l.from, "From page", strconv.Itoa(e.PagesRead())), num(&l.to, "To page", toPH), 1, 1)
			}, num(&l.mins, "Time spent (mins)", "e.g. 45"),
				thoughts("Your thoughts right now", "What are you feeling as the story unfolds?", 88))
		default:
			rows = append(rows, num(&l.mins, "Time spent (mins)", "e.g. 30"),
				thoughts("Your thoughts right now", "What moved you? What will you remember?", 88))
		}
	}
	rows = append(rows, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return bigButton(gtx, th, &l.save, saveLabel)
		})
	})
	children := make([]layout.FlexChild, len(rows))
	for i, r := range rows {
		children[i] = layout.Rigid(r)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// optNum reads a number field: nil when blank (core fills in defaults).
func optNum(in *Input) *int {
	t := strings.TrimSpace(in.Editor.Text())
	if t == "" {
		return nil
	}
	n, err := strconv.Atoi(t)
	if err != nil {
		return nil
	}
	return &n
}

func (l *logSheet) submit(a *App, e *core.Entry) {
	note := l.note.Editor.Text()
	if l.rewatch {
		full := l.full
		in := core.RewatchInput{Date: l.date.value, Note: note, Rating: l.rating, Full: &full,
			StartDate: l.start.value, EndDate: l.end.value}
		if p := optNum(&l.partial); p != nil {
			in.Episodes = *p
		}
		var err error
		if l.editID != "" {
			err = e.EditRewatch(l.editID, in, a.Loc)
		} else {
			err = e.LogRewatch(in, a.Loc)
		}
		if err != nil {
			if err == core.ErrDateRequired {
				a.ToastError("Please choose a date.")
			} else {
				a.ToastError(err.Error())
			}
			return
		}
		if a.save(false, []*core.Entry{e}, false) {
			a.CloseSheet()
		}
		return
	}
	in := core.SessionInput{Note: note}
	switch e.Type.Progress() {
	case core.EpisodeProgress:
		in.Episodes = optNum(&l.eps)
	case core.PageProgress:
		in.FromPage, in.ToPage = optNum(&l.from), optNum(&l.to)
	}
	if e.Type.Progress() != core.EpisodeProgress {
		in.Minutes = optNum(&l.mins)
	}
	var prompt bool
	var err error
	if l.editID != "" {
		prompt, err = e.EditSession(l.editID, in)
	} else {
		prompt, err = e.LogSession(in, a.Now())
	}
	if err != nil {
		a.ToastError(err.Error())
		return
	}
	if !a.save(false, []*core.Entry{e}, false) {
		return
	}
	a.CloseSheet()
	if l.editID != "" {
		a.Toast("Session updated ✦")
	} else {
		a.Toast("Session logged ✦")
	}
	if prompt {
		reachedEndDialog(a, e.ID)
	}
}

// reachedEndDialog is "Looks like you reached the end!" (SPEC §4.4).
func reachedEndDialog(a *App, id string) {
	e := a.Lib.Entry(id)
	if e == nil {
		return
	}
	d := ConfirmDialog("Looks like you reached the end!",
		"You've caught up with every bit of "+e.Title+". Mark it as finished?", "Not yet", "Mark as finished",
		func(a *App) {
			e := a.Lib.Entry(id)
			if e == nil {
				return
			}
			e.SetStatus(core.Finished, a.Now())
			if a.save(false, []*core.Entry{e}, false) {
				a.Toast("Marked as finished ✦")
			}
		})
	d.Danger = markFinishedRed
	a.ShowDialog(d)
}

// markFinishedRed: the prototype draws "Mark as finished" in the Delete
// dialog's red. Open question for the owner (STATUS).
var markFinishedRed = true

// confirmDeleteJourneyItem asks before removing a session or rewatch.
func confirmDeleteJourneyItem(a *App, entryID, itemID string) {
	a.ShowDialog(ConfirmDialog("Delete this entry?",
		"This piece of your journey will be removed. This can't be undone.", "", "Delete",
		func(a *App) {
			e := a.Lib.Entry(entryID)
			if e == nil {
				return
			}
			var err error
			if isRewatch(e, itemID) {
				err = e.DeleteRewatch(itemID)
			} else {
				err = e.DeleteSession(itemID)
			}
			if err != nil {
				a.ToastError(err.Error())
				return
			}
			if a.save(false, []*core.Entry{e}, false) {
				a.Toast("Entry deleted")
			}
		}))
}
