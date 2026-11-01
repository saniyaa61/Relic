package ui

// The guided tour: after onboarding, a welcome pop-up, then one step per
// part of the app. Each step opens its page, lights up its place (a tab
// in the bottom bar, or the Settings gear) and explains it in a small
// card with Next and Skip (owner's design, 2026-10-31, instead of the
// welcome card). Profile.ShowWelcome keeps it pending across restarts
// until it's finished or skipped.

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/store"
)

// guideTarget is what a step lights up.
type guideTarget int

const (
	guideNothing guideTarget = iota // a card in the middle
	guideTab                        // the step's tab in the bottom bar
	guideGear                       // Home's Settings gear
)

type guideStep struct {
	tab         Tab
	target      guideTarget
	title, body string
	next        string
}

var guideSteps = []guideStep{
	{TabHome, guideNothing, "Welcome to your archive",
		"Relic keeps the films, dramas, books, podcasts and music you love, and how each one made you feel. Here's a short walk through it.",
		"Show me around"},
	{TabLibrary, guideTab, "Your Library",
		"These are your shelves. A category is one kind of story: Films, Comfort rewatches, Books for the train. Its type decides what you log. Folders inside a category keep things finer.",
		"Next"},
	{TabNew, guideTab, "Log a story",
		"Add anything that stayed with you: pick its shelf, a poster from your photos, where you are in it, your rating and the feelings it left. Log each session as you go, and Relic counts the time.",
		"Next"},
	{TabFavorites, guideTab, "Favorites",
		"Tap the heart on any entry to keep it here. Each category also has an All-Time Top 5 you can rank.",
		"Next"},
	{TabDigest, guideTab, "Your Digest",
		"Your week, month and year, gently gathered: time spent in other worlds, what you finished, your streak and how you felt. Share your Year in Review card when the year is done.",
		"Next"},
	{TabHome, guideTab, "Home",
		"Where you left off: what's still with you, what you finished lately, your totals and streak. Tap Consumed to see where your time went.",
		"Next"},
	{TabHome, guideGear, "Settings",
		"Make Relic yours: six themes in light or dark, or your own colours, and your name. Export your archive now and then to keep a copy safe, or import one to bring your stories over.",
		"Next"},
	{TabHome, guideNothing, "You're all set",
		"Start with a shelf in the Library, then log a story you love. Your archive grows from there.",
		"Begin"},
}

type guide struct {
	step     int
	shownAt  time.Time
	next     widget.Clickable
	skip     widget.Clickable
	blockTag int
}

// guideRects are where the tour's targets are, noted while laying out.
type guideRects struct {
	nav  [tabCount]image.Rectangle
	gear image.Rectangle
}

// startGuide begins the tour (after onboarding, or at launch if it was
// left unfinished).
func (a *App) startGuide() {
	a.guide = &guide{shownAt: a.Now()}
	a.stack = nil
	a.Go(guideSteps[0].tab)
}

// endGuide closes the tour for good.
func (a *App) endGuide() {
	a.guide = nil
	a.Go(TabHome)
	prof := a.Lib.Profile
	prof.ShowWelcome = false
	a.Lib.Profile = prof
	a.Update(func(w store.Writer) error { return w.SaveProfile(prof) })
}

// layoutGuide draws the tour over the page and handles its buttons.
func (a *App) layoutGuide(gtx layout.Context) {
	g := a.guide
	if g == nil {
		return
	}
	if g.skip.Clicked(gtx) {
		a.endGuide()
		return
	}
	if g.next.Clicked(gtx) {
		if g.step == len(guideSteps)-1 {
			a.endGuide()
			return
		}
		g.step++
		g.shownAt = a.Now()
		a.stack = nil
		a.Go(guideSteps[g.step].tab)
	}
	st := guideSteps[g.step]
	th := a.Theme
	size := gtx.Constraints.Max
	t := easeCSS(progress(gtx, a.Now(), g.shownAt, 220*time.Millisecond))
	defer paint.PushOpacity(gtx.Ops, t).Pop()

	var hole image.Rectangle
	switch st.target {
	case guideTab:
		hole = a.guideRects.nav[st.tab]
	case guideGear:
		hole = a.guideRects.gear
	}
	if !hole.Empty() {
		hole = hole.Inset(-gtx.Dp(4))
	}
	r := gtx.Dp(14)

	// The dimmed page, with the target cut out, swallowing touches.
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(0, 0))
	p.LineTo(f32.Pt(float32(size.X), 0))
	p.LineTo(f32.Pt(float32(size.X), float32(size.Y)))
	p.LineTo(f32.Pt(0, float32(size.Y)))
	p.Close()
	if !hole.Empty() {
		roundedRectReverse(&p, hole, float32(min(r, hole.Dx()/2, hole.Dy()/2)))
	}
	paint.FillShape(gtx.Ops, shade(0.5, th.Bg), clip.Outline{Path: p.End()}.Op())
	area := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &g.blockTag)
	for {
		if _, ok := gtx.Event(pointer.Filter{Target: &g.blockTag, Kinds: pointer.Press | pointer.Release | pointer.Cancel | pointer.Drag | pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -1e6, Max: 1e6}}); !ok {
			break
		}
	}
	area.Pop()
	if !hole.Empty() {
		ring := clip.UniformRRect(hole, min(r, hole.Dx()/2, hole.Dy()/2))
		paint.FillShape(gtx.Ops, th.Accent2, clip.Stroke{Path: ring.Path(gtx.Ops), Width: float32(gtx.Dp(2))}.Op())
	}

	// The card: above a tab, under the gear, or in the middle.
	cw := min(size.X-2*gtx.Dp(gutter), gtx.Dp(340))
	cgtx := gtx
	cgtx.Constraints = layout.Constraints{Min: image.Pt(cw, 0), Max: image.Pt(cw, size.Y)}
	rec := op.Record(gtx.Ops)
	d := a.guideCard(cgtx, st)
	card := rec.Stop()
	x := (size.X - cw) / 2
	gap := gtx.Dp(14)
	var y int
	pointUp, pointDown := false, false
	switch {
	case hole.Empty():
		y = (size.Y - d.Size.Y) / 2
	case hole.Min.Y > size.Y/2:
		y = hole.Min.Y - gap - d.Size.Y
		pointDown = true
	default:
		y = hole.Max.Y + gap
		pointUp = true
	}
	// A small pointer from the card to the target.
	if pointUp || pointDown {
		cx := float32(min(max(hole.Min.X+hole.Dx()/2, x+gtx.Dp(24)), x+cw-gtx.Dp(24)))
		w, h := float32(gtx.Dp(9)), float32(gtx.Dp(8))
		var tp clip.Path
		tp.Begin(gtx.Ops)
		if pointDown {
			base := float32(y + d.Size.Y - 1)
			tp.MoveTo(f32.Pt(cx-w, base))
			tp.LineTo(f32.Pt(cx+w, base))
			tp.LineTo(f32.Pt(cx, base+h))
		} else {
			base := float32(y + 1)
			tp.MoveTo(f32.Pt(cx-w, base))
			tp.LineTo(f32.Pt(cx+w, base))
			tp.LineTo(f32.Pt(cx, base-h))
		}
		tp.Close()
		paint.FillShape(gtx.Ops, th.Card, clip.Outline{Path: tp.End()}.Op())
	}
	off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
	card.Add(gtx.Ops)
	off.Pop()
}

// guideCard is one step's card: where we are, the words, Skip and Next.
func (a *App) guideCard(gtx layout.Context, st guideStep) layout.Dimensions {
	g := a.guide
	th := a.Theme
	title := st.title
	if g.step == 0 && a.Lib.Profile.Name != "" {
		title = "Welcome, " + a.Lib.Profile.Name
	}
	last := g.step == len(guideSteps)-1
	return card(gtx, th.Card, th.Border, 16, layout.UniformInset(18), 0, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				eyebrow := "A quick tour"
				if g.step > 0 && !last {
					eyebrow += " · " + itoa(g.step) + " of " + itoa(len(guideSteps)-2)
				}
				return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.12, Upper: true, Color: th.Muted}.Layout(gtx, th, eyebrow)
			}),
			layout.Rigid(layout.Spacer{Height: 5}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Paragraph{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 18, LineHeight: 1.3, Color: th.Text}.Layout(gtx, th, title)
			}),
			layout.Rigid(layout.Spacer{Height: 6}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.6, Color: th.Muted}.Layout(gtx, th, st.body)
			}),
			layout.Rigid(layout.Spacer{Height: 14}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layout.Flex{Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if last {
							return layout.Dimensions{}
						}
						return g.skip.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(6).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return underlined(gtx, th, Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: th.Muted}, "Skip tour")
							})
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return pressable(gtx, &g.next, func(gtx layout.Context) layout.Dimensions {
							return card(gtx, th.Accent, th.Accent, 11, layout.Inset{Top: 9, Bottom: 9, Left: 20, Right: 20}, 0, func(gtx layout.Context) layout.Dimensions {
								return Text{Font: font.Font{Typeface: Display, Style: font.Italic}, Size: 14, Color: th.BtnText}.Layout(gtx, th, st.next+" ✦")
							})
						})
					}))
			}))
	})
}

// roundedRectReverse adds r's outline to p anticlockwise, so with the
// non-zero rule it cuts a hole in a clockwise shape.
func roundedRectReverse(p *clip.Path, r image.Rectangle, rad float32) {
	x0, y0, x1, y1 := float32(r.Min.X), float32(r.Min.Y), float32(r.Max.X), float32(r.Max.Y)
	const k = 0.5523 // cubic handle for a quarter circle
	c := rad * k
	p.MoveTo(f32.Pt(x0+rad, y0))
	p.CubeTo(f32.Pt(x0+rad-c, y0), f32.Pt(x0, y0+rad-c), f32.Pt(x0, y0+rad))
	p.LineTo(f32.Pt(x0, y1-rad))
	p.CubeTo(f32.Pt(x0, y1-rad+c), f32.Pt(x0+rad-c, y1), f32.Pt(x0+rad, y1))
	p.LineTo(f32.Pt(x1-rad, y1))
	p.CubeTo(f32.Pt(x1-rad+c, y1), f32.Pt(x1, y1-rad+c), f32.Pt(x1, y1-rad))
	p.LineTo(f32.Pt(x1, y0+rad))
	p.CubeTo(f32.Pt(x1, y0+rad-c), f32.Pt(x1-rad+c, y0), f32.Pt(x1-rad, y0))
	p.Close()
}
