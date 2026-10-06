package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// Icon is a line icon in a 24×24 box, built from the same SVG markup the
// prototype uses, so icons can be copied across unchanged. It is parsed
// once into absolute cubic segments.
type Icon struct {
	segs []iconSeg
}

type iconSeg struct {
	op  byte // 'M' move, 'L' line, 'C' cubic, 'Z' close
	pts [3]f32.Point
}

// IconSVG builds an icon from SVG element markup: path, circle, rect (with
// rx), line and polyline, as in the prototype's <svg viewBox="0 0 24 24">.
// It panics on markup it can't read; icons are fixed at compile time and
// icon_test.go parses every one.
func IconSVG(markup string) *Icon {
	ic := &Icon{}
	for _, el := range splitElements(markup) {
		var err error
		switch el.name {
		case "path":
			err = ic.path(el.attr["d"])
		case "circle":
			cx, cy, r := el.num("cx"), el.num("cy"), el.num("r")
			err = ic.path(fmt.Sprintf("M%g %gA%g %g 0 1 0 %g %gA%g %g 0 1 0 %g %gZ", cx-r, cy, r, r, cx+r, cy, r, r, cx-r, cy))
		case "rect":
			x, y, w, h, rx := el.num("x"), el.num("y"), el.num("width"), el.num("height"), el.num("rx")
			if rx == 0 {
				err = ic.path(fmt.Sprintf("M%g %gH%gV%gH%gZ", x, y, x+w, y+h, x))
			} else {
				err = ic.path(fmt.Sprintf("M%g %gH%gA%g %g 0 0 1 %g %gV%gA%g %g 0 0 1 %g %gH%gA%g %g 0 0 1 %g %gV%gA%g %g 0 0 1 %g %gZ",
					x+rx, y, x+w-rx, rx, rx, x+w, y+rx, y+h-rx, rx, rx, x+w-rx, y+h, x+rx, rx, rx, x, y+h-rx, y+rx, rx, rx, x+rx, y))
			}
		case "line":
			err = ic.path(fmt.Sprintf("M%g %gL%g %g", el.num("x1"), el.num("y1"), el.num("x2"), el.num("y2")))
		case "polyline", "polygon":
			d := "M" + el.attr["points"]
			if el.name == "polygon" {
				d += "Z"
			}
			err = ic.path(d)
		default:
			err = fmt.Errorf("unsupported element <%s>", el.name)
		}
		if err != nil {
			panic(fmt.Sprintf("icon %q: %v", markup, err))
		}
	}
	return ic
}

type svgElement struct {
	name string
	attr map[string]string
}

func (e svgElement) num(key string) float32 {
	v, _ := strconv.ParseFloat(e.attr[key], 32)
	return float32(v)
}

// splitElements reads self-closing elements like <circle cx="12" r="3"/>.
func splitElements(markup string) []svgElement {
	var els []svgElement
	for _, part := range strings.Split(markup, "<")[1:] {
		part = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(part), ">")), "/"))
		name, rest, _ := strings.Cut(part, " ")
		el := svgElement{name: name, attr: map[string]string{}}
		for {
			rest = strings.TrimSpace(rest)
			key, after, ok := strings.Cut(rest, `="`)
			if !ok {
				break
			}
			val, after2, _ := strings.Cut(after, `"`)
			el.attr[strings.TrimSpace(key)] = val
			rest = after2
		}
		els = append(els, el)
	}
	return els
}

// path appends SVG path data: M L H V C S Q A Z, absolute and relative.
func (ic *Icon) path(d string) error {
	toks := tokenizePath(d)
	var cur, start, lastCtrl f32.Point
	var cmd, prev byte
	i := 0
	num := func() (float32, error) {
		if i >= len(toks) || isCmd(toks[i]) {
			return 0, fmt.Errorf("missing number after %c", cmd)
		}
		v, err := strconv.ParseFloat(toks[i], 32)
		i++
		return float32(v), err
	}
	pt := func(rel bool) (f32.Point, error) {
		x, err := num()
		if err != nil {
			return f32.Point{}, err
		}
		y, err := num()
		if rel {
			x, y = x+cur.X, y+cur.Y
		}
		return f32.Pt(x, y), err
	}
	for i < len(toks) {
		if isCmd(toks[i]) {
			cmd = toks[i][0]
			i++
		} else if cmd == 0 {
			return fmt.Errorf("path must start with a command")
		}
		rel := cmd >= 'a'
		up := cmd &^ 0x20
		switch up {
		case 'M':
			p, err := pt(rel)
			if err != nil {
				return err
			}
			ic.segs = append(ic.segs, iconSeg{op: 'M', pts: [3]f32.Point{p}})
			cur, start = p, p
			// Further pairs after a moveto are implicit linetos.
			if rel {
				cmd = 'l'
			} else {
				cmd = 'L'
			}
		case 'L', 'H', 'V':
			p := cur
			var err error
			switch up {
			case 'L':
				p, err = pt(rel)
			case 'H':
				var x float32
				x, err = num()
				if rel {
					x += cur.X
				}
				p.X = x
			case 'V':
				var y float32
				y, err = num()
				if rel {
					y += cur.Y
				}
				p.Y = y
			}
			if err != nil {
				return err
			}
			ic.segs = append(ic.segs, iconSeg{op: 'L', pts: [3]f32.Point{p}})
			cur = p
		case 'C', 'S', 'Q':
			var c1, c2, p f32.Point
			var err error
			switch up {
			case 'C':
				if c1, err = pt(rel); err != nil {
					return err
				}
				if c2, err = pt(rel); err != nil {
					return err
				}
			case 'S':
				c1 = cur
				if pu := prev &^ 0x20; pu == 'C' || pu == 'S' {
					c1 = cur.Mul(2).Sub(lastCtrl)
				}
				if c2, err = pt(rel); err != nil {
					return err
				}
			case 'Q':
				var q f32.Point
				if q, err = pt(rel); err != nil {
					return err
				}
				p0 := cur
				if p, err = pt(rel); err != nil {
					return err
				}
				c1 = p0.Add(q.Sub(p0).Mul(2.0 / 3))
				c2 = p.Add(q.Sub(p).Mul(2.0 / 3))
				ic.segs = append(ic.segs, iconSeg{op: 'C', pts: [3]f32.Point{c1, c2, p}})
				cur, lastCtrl, prev = p, q, cmd
				continue
			}
			if p, err = pt(rel); err != nil {
				return err
			}
			ic.segs = append(ic.segs, iconSeg{op: 'C', pts: [3]f32.Point{c1, c2, p}})
			cur, lastCtrl = p, c2
		case 'A':
			var v [5]float32
			for k := range v {
				// Flags may be written without separators ("0 1 0" or "011").
				if k >= 3 && i < len(toks) && len(toks[i]) > 1 && (toks[i][0] == '0' || toks[i][0] == '1') {
					v[k] = float32(toks[i][0] - '0')
					toks[i] = toks[i][1:]
					continue
				}
				var err error
				if v[k], err = num(); err != nil {
					return err
				}
			}
			p, err := pt(rel)
			if err != nil {
				return err
			}
			ic.arc(cur, p, v[0], v[1], v[2], v[3] != 0, v[4] != 0)
			cur = p
		case 'Z':
			ic.segs = append(ic.segs, iconSeg{op: 'Z'})
			cur = start
		default:
			return fmt.Errorf("unsupported path command %c", cmd)
		}
		prev = cmd
	}
	return nil
}

func isCmd(tok string) bool {
	return len(tok) == 1 && strings.ContainsRune("MmLlHhVvCcSsQqAaZz", rune(tok[0]))
}

// tokenizePath splits path data into commands and numbers, handling the
// compact forms SVG allows ("1.5.5", "2-3", "a5.5 5.5 0 00-7.78 0").
func tokenizePath(d string) []string {
	var toks []string
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			toks = append(toks, b.String())
			b.Reset()
		}
	}
	for k := 0; k < len(d); k++ {
		c := d[k]
		switch {
		case strings.IndexByte("MmLlHhVvCcSsQqAaZz", c) >= 0:
			flush()
			toks = append(toks, string(c))
		case c == ' ' || c == ',' || c == '\n' || c == '\t':
			flush()
		case c == '-':
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "e") {
				flush()
			}
			b.WriteByte(c)
		case c == '.':
			if strings.Contains(b.String(), ".") {
				flush()
			}
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	flush()
	return toks
}

// arc appends an SVG elliptical arc as cubic Béziers (SVG spec F.6).
func (ic *Icon) arc(p0, p1 f32.Point, rx, ry, rotDeg float32, large, sweep bool) {
	if p0 == p1 {
		return
	}
	if rx == 0 || ry == 0 {
		ic.segs = append(ic.segs, iconSeg{op: 'L', pts: [3]f32.Point{p1}})
		return
	}
	x0, y0, x1, y1 := float64(p0.X), float64(p0.Y), float64(p1.X), float64(p1.Y)
	rX, rY := math.Abs(float64(rx)), math.Abs(float64(ry))
	phi := float64(rotDeg) * math.Pi / 180
	sin, cos := math.Sincos(phi)
	dx, dy := (x0-x1)/2, (y0-y1)/2
	x1p := cos*dx + sin*dy
	y1p := -sin*dx + cos*dy
	if l := x1p*x1p/(rX*rX) + y1p*y1p/(rY*rY); l > 1 {
		s := math.Sqrt(l)
		rX, rY = rX*s, rY*s
	}
	num := rX*rX*rY*rY - rX*rX*y1p*y1p - rY*rY*x1p*x1p
	den := rX*rX*y1p*y1p + rY*rY*x1p*x1p
	coef := math.Sqrt(math.Max(0, num/den))
	if large == sweep {
		coef = -coef
	}
	cxp, cyp := coef*rX*y1p/rY, -coef*rY*x1p/rX
	cx := cos*cxp - sin*cyp + (x0+x1)/2
	cy := sin*cxp + cos*cyp + (y0+y1)/2
	angle := func(ux, uy, vx, vy float64) float64 {
		a := math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
		return a
	}
	th1 := angle(1, 0, (x1p-cxp)/rX, (y1p-cyp)/rY)
	dth := angle((x1p-cxp)/rX, (y1p-cyp)/rY, (-x1p-cxp)/rX, (-y1p-cyp)/rY)
	if !sweep && dth > 0 {
		dth -= 2 * math.Pi
	} else if sweep && dth < 0 {
		dth += 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(dth) / (math.Pi / 2)))
	step := dth / float64(n)
	k := 4.0 / 3 * math.Tan(step/4)
	at := func(t float64) (px, py, tx, ty float64) {
		st, ct := math.Sincos(t)
		px = cx + rX*ct*cos - rY*st*sin
		py = cy + rX*ct*sin + rY*st*cos
		tx = -rX*st*cos - rY*ct*sin
		ty = -rX*st*sin + rY*ct*cos
		return
	}
	t := th1
	for range n {
		ax, ay, atx, aty := at(t)
		bx, by, btx, bty := at(t + step)
		ic.segs = append(ic.segs, iconSeg{op: 'C', pts: [3]f32.Point{
			{X: float32(ax + k*atx), Y: float32(ay + k*aty)},
			{X: float32(bx - k*btx), Y: float32(by - k*bty)},
			{X: float32(bx), Y: float32(by)},
		}})
		t += step
	}
	// Land exactly on the endpoint.
	ic.segs[len(ic.segs)-1].pts[2] = p1
}

// Layout strokes the icon at size dp with the given stroke width in icon
// units (CSS stroke-width), with round caps and joins as the prototype.
func (ic *Icon) Layout(gtx layout.Context, size unit.Dp, strokeW float32, c color.NRGBA) layout.Dimensions {
	px := gtx.Dp(size)
	s := float32(px) / 24
	var p clip.Path
	p.Begin(gtx.Ops)
	for _, sg := range ic.segs {
		switch sg.op {
		case 'M':
			p.MoveTo(sg.pts[0].Mul(s))
		case 'L':
			p.LineTo(sg.pts[0].Mul(s))
		case 'C':
			p.CubeTo(sg.pts[0].Mul(s), sg.pts[1].Mul(s), sg.pts[2].Mul(s))
		case 'Z':
			p.Close()
		}
	}
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: strokeW * s}.Op())
	return layout.Dimensions{Size: image.Pt(px, px)}
}
